package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// 这些用例针对的是「必须由 DB 保证」的两件事：时段唯一约束、评分聚合。
// 应用层的检查是 TOCTOU（实测并发 12 个请求曾成功 7 个），只有真库能证明约束真的挡住了。

func isUniqueViolationErr(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// 同一摄影师的同一时段：未取消的订单只能有一条；取消后时段释放。
func TestIntegration_BookingSlot_UniqueAmongActiveBookings(t *testing.T) {
	q, ctx := integrationQueries(t)
	pid := insertPhotographerFixture(t, q, ctx)

	date, err := time.Parse("2006-01-02", "2099-03-03")
	if err != nil {
		t.Fatalf("parse date: %v", err)
	}
	insertBookingFixture(t, q, ctx, pid, 1, "pending", "2099-03-03", "10:00")

	// 第二个未取消订单必须被唯一索引拒绝（应用层检查此时已被绕过，模拟并发下的败者）。
	var dupID int64
	err = q.db.QueryRow(ctx,
		`INSERT INTO bookings (photographer_id, coser_id, service_id, date, time, status, total_price)
		 VALUES ($1, 2, 0, $2::date, '10:00', 'confirmed', 0) RETURNING id`, pid, date).Scan(&dupID)
	if !isUniqueViolationErr(err) {
		t.Fatalf("want unique violation for a duplicate active slot, got %v", err)
	}

	// 已取消的重复时段是允许的（历史取消单可以堆叠，取消要释放时段）。
	if _, err := q.db.Exec(ctx,
		`INSERT INTO bookings (photographer_id, coser_id, service_id, date, time, status, total_price)
		 VALUES ($1, 2, 0, $2::date, '10:00', 'cancelled', 0)`, pid, date); err != nil {
		t.Fatalf("a cancelled booking must not conflict: %v", err)
	}

	// 把第一条取消掉之后，同一时段可以重新被预约。
	if _, err := q.db.Exec(ctx,
		`UPDATE bookings SET status = 'cancelled' WHERE photographer_id = $1 AND date = $2::date`, pid, date); err != nil {
		t.Fatalf("cancel fixture: %v", err)
	}
	if _, err := q.db.Exec(ctx,
		`INSERT INTO bookings (photographer_id, coser_id, service_id, date, time, status, total_price)
		 VALUES ($1, 3, 0, $2::date, '10:00', 'pending', 0)`, pid, date); err != nil {
		t.Fatalf("slot must be free after cancellation, got %v", err)
	}
}

// 评分与评价数必须按 reviews 表重算，而不是停留在种子数字上。
func TestIntegration_RecomputePhotographerRating(t *testing.T) {
	q, ctx := integrationQueries(t)
	pid := insertPhotographerFixture(t, q, ctx)
	if _, err := q.db.Exec(ctx, `UPDATE photographers SET rating = 4.9, review_count = 234 WHERE id = $1`, pid); err != nil {
		t.Fatalf("seed stale rating: %v", err)
	}

	insertReviewFixture(t, q, ctx, pid, 11, 2)
	insertReviewFixture(t, q, ctx, pid, 12, 4)

	if err := q.RecomputePhotographerRating(ctx, pid); err != nil {
		t.Fatalf("RecomputePhotographerRating: %v", err)
	}

	var rating float64
	var count int32
	if err := q.db.QueryRow(ctx, `SELECT rating::float8, review_count FROM photographers WHERE id = $1`, pid).
		Scan(&rating, &count); err != nil {
		t.Fatalf("read back: %v", err)
	}
	if count != 2 {
		t.Errorf("review_count = %d, want 2 (recomputed from reviews)", count)
	}
	if rating < 2.99 || rating > 3.01 {
		t.Errorf("rating = %v, want 3.0 (avg of 2 and 4)", rating)
	}
}

// 「必须存在已完成订单」的判定。
func TestIntegration_HasCompletedBookingWith(t *testing.T) {
	q, ctx := integrationQueries(t)
	pid := insertPhotographerFixture(t, q, ctx)
	const coserID = int32(21)

	insertBookingFixture(t, q, ctx, pid, coserID, "pending", "2099-04-04", "10:00")
	if ok, err := q.HasCompletedBookingWith(ctx, pid, coserID); err != nil || ok {
		t.Fatalf("pending booking must not count as completed: ok=%v err=%v", ok, err)
	}

	if _, err := q.db.Exec(ctx,
		`UPDATE bookings SET status = 'completed' WHERE photographer_id = $1 AND coser_id = $2`, pid, coserID); err != nil {
		t.Fatalf("complete fixture: %v", err)
	}
	if ok, err := q.HasCompletedBookingWith(ctx, pid, coserID); err != nil || !ok {
		t.Fatalf("completed booking must count: ok=%v err=%v", ok, err)
	}

	// 另一个 coser 不该被算进来。
	if ok, err := q.HasCompletedBookingWith(ctx, pid, 22); err != nil || ok {
		t.Fatalf("another coser must not inherit the completed booking: ok=%v err=%v", ok, err)
	}
}

// 列表排序键真的生效，且排序稳定（分页不重复不遗漏）。
func TestIntegration_SearchPhotographers_SortKeysAndStablePaging(t *testing.T) {
	q, ctx := integrationQueries(t)
	newest := insertPhotographerFixture(t, q, ctx)
	middle := insertPhotographerFixture(t, q, ctx)
	oldest := insertPhotographerFixture(t, q, ctx)

	// 刻意给「最新入驻」的评分最低、单量最少，好让不同排序键产生不同顺序。
	setStats(t, q, ctx, newest, 3.0, 10, 1)
	setStats(t, q, ctx, middle, 4.0, 30, 2)
	setStats(t, q, ctx, oldest, 5.0, 20, 3)

	idsForSort := func(sort string) []int64 {
		rows, err := q.SearchPhotographers(ctx, SearchPhotographersParams{
			Limit: 50, ExcludeUserIDs: []int64{}, Sort: &sort,
		})
		if err != nil {
			t.Fatalf("SearchPhotographers(sort=%s): %v", sort, err)
		}
		// 只保留夹具，避免受演示数据影响。
		fixtures := []int64{newest, middle, oldest}
		out := []int64{}
		for _, r := range rows {
			for _, f := range fixtures {
				if int64(r.ID) == f {
					out = append(out, f)
				}
			}
		}
		return out
	}

	assertOrder := func(name string, got, want []int64) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s: got %d fixtures, want %d (%v)", name, len(got), len(want), got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s: order = %v, want %v", name, got, want)
				return
			}
		}
	}

	assertOrder("rating 降序", idsForSort("rating"), []int64{oldest, middle, newest})
	assertOrder("order 降序（单量）", idsForSort("order"), []int64{middle, oldest, newest})
	assertOrder("new 降序（入驻时间）", idsForSort("new"), []int64{newest, middle, oldest})

	// 分页稳定性：排序键相同（这里把三者评分设成一样）时，逐页取回不应有重复。
	if _, err := q.db.Exec(ctx,
		`UPDATE photographers SET rating = 4.2 WHERE id = ANY($1::bigint[])`,
		[]int64{newest, middle, oldest}); err != nil {
		t.Fatalf("flatten ratings: %v", err)
	}
	seen := map[int64]int{}
	for page := 0; page < 3; page++ {
		rating := "rating"
		rows, err := q.SearchPhotographers(ctx, SearchPhotographersParams{
			Limit: 1, Offset: int32(page), ExcludeUserIDs: []int64{}, Sort: &rating,
		})
		if err != nil {
			t.Fatalf("paged search: %v", err)
		}
		for _, r := range rows {
			seen[int64(r.ID)]++
		}
	}
	for _, f := range []int64{newest, middle, oldest} {
		if seen[f] > 1 {
			t.Errorf("photographer %d appeared %d times across pages — paging is not stable", f, seen[f])
		}
	}
}

func setStats(t *testing.T, q *Queries, ctx context.Context, id int64, rating float64, orderCount int32, daysAgo int) {
	t.Helper()
	if _, err := q.db.Exec(ctx,
		`UPDATE photographers SET rating = $2, order_count = $3, review_count = $3,
		 activated_at = NOW() - make_interval(days => $4) WHERE id = $1`,
		id, rating, orderCount, daysAgo); err != nil {
		t.Fatalf("set stats: %v", err)
	}
}

func insertReviewFixture(t *testing.T, q *Queries, ctx context.Context, photographerID int64, userID, rating int32) int64 {
	t.Helper()
	var id int64
	if err := q.db.QueryRow(ctx,
		`INSERT INTO reviews (photographer_id, user_id, user_name, rating, content, images)
		 VALUES ($1, $2, '__integrity_it__', $3, 'integrity fixture', '{}') RETURNING id`,
		photographerID, userID, rating).Scan(&id); err != nil {
		t.Fatalf("insert review fixture: %v", err)
	}
	t.Cleanup(func() {
		_, _ = q.db.Exec(context.Background(), `DELETE FROM reviews WHERE id = $1`, id)
	})
	return id
}
