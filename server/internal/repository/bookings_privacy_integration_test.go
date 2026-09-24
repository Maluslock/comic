package repository

import (
	"context"
	"testing"
)

// B-06：coser 手机号是个人信息，摄影师在**确认接单前**没有正当必要看到它。
//
// 遮罩写在 SQL 里（而不是只藏前端），因为只藏前端等于没藏 —— 完整号码仍在响应体里，
// 抓一次包就能拿到。也正因为如此，喂假行数据的单测证明不了这件事：它只能证明「我给什么
// 就返回什么」。所以这条必须连真库跑。
func TestIntegration_GetBookingsByPhotographer_MasksCoserPhoneUntilConfirmed(t *testing.T) {
	q, ctx := integrationQueries(t)

	// coser 取一个真实用户：手机号来自 users 表的 join，否则 LEFT JOIN 出来是空串，
	// 测试会「因为拿不到号码」而假通过。
	var coserID int32
	var coserPhone string
	if err := q.db.QueryRow(ctx,
		`SELECT id, phone FROM users WHERE phone <> '' ORDER BY id LIMIT 1`).Scan(&coserID, &coserPhone); err != nil {
		t.Fatalf("pick coser fixture: %v", err)
	}

	photographerID := insertPhotographerFixture(t, q, ctx)

	cases := []struct {
		status      string
		wantVisible bool
		date        string
		time        string
	}{
		{"pending", false, "2099-01-01", "10:00"},
		{"confirmed", true, "2099-01-02", "11:00"},
		{"completed", true, "2099-01-03", "12:00"},
		{"cancelled", false, "2099-01-04", "13:00"},
	}
	for _, c := range cases {
		insertBookingFixture(t, q, ctx, photographerID, coserID, c.status, c.date, c.time)
	}

	rows, err := q.GetBookingsByPhotographer(ctx, int32(photographerID))
	if err != nil {
		t.Fatalf("GetBookingsByPhotographer: %v", err)
	}
	if len(rows) != len(cases) {
		t.Fatalf("want %d bookings, got %d", len(cases), len(rows))
	}

	byStatus := make(map[string]BookingWithCoser, len(rows))
	for _, r := range rows {
		byStatus[r.Status] = r
	}

	for _, c := range cases {
		got, ok := byStatus[c.status]
		if !ok {
			t.Errorf("no booking returned for status %q", c.status)
			continue
		}
		if c.wantVisible {
			if got.CoserPhone != coserPhone {
				t.Errorf("status %q: want coser phone %q, got %q", c.status, coserPhone, got.CoserPhone)
			}
			continue
		}
		if got.CoserPhone != "" {
			t.Errorf("status %q must not disclose the coser phone, got %q", c.status, got.CoserPhone)
		}
		// 遮罩只针对手机号，别的字段照常返回（别顺手把整行藏了）。
		if got.CoserName == "" {
			t.Errorf("status %q: masking the phone must not blank out other join fields", c.status)
		}
	}
}

func insertBookingFixture(t *testing.T, q *Queries, ctx context.Context, photographerID int64, coserID int32, status, date, tm string) int64 {
	t.Helper()
	var id int64
	if err := q.db.QueryRow(ctx,
		`INSERT INTO bookings (photographer_id, coser_id, service_id, date, time, status, total_price)
		 VALUES ($1, $2, 0, $3::date, $4, $5, 0) RETURNING id`,
		photographerID, coserID, date, tm, status).Scan(&id); err != nil {
		t.Fatalf("insert booking fixture: %v", err)
	}
	// 注册在摄影师夹具之后 → LIFO 先删订单，否则删摄影师会被外键挡住。
	t.Cleanup(func() {
		_, _ = q.db.Exec(context.Background(), `DELETE FROM bookings WHERE id = $1`, id)
	})
	return id
}
