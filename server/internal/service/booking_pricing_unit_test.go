package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/Maluslock/comic/server/internal/repository"
)

// per_photo 是这一批改动里唯一会做乘法的分支，也是最容易算错钱的地方：
// 单价 × 张数。张数有三种来源（显式填写 / 套餐起拍张数 / 单张兜底），
// 三条路径都必须落一个确定的张数快照 —— 历史上「张数留空」会写 NULL，
// 结果订单显示 ¥10 却被当成成交价，双方都没有补救入口。

func TestDeriveBookingPricing_PerPhotoOneShot(t *testing.T) {
	svc := repository.Service{Name: "按张", Price: int32Ptr(10), Duration: 30, PriceUnit: "per_photo"}
	one := int32(1)

	p := deriveBookingPricing(svc, &one)
	if p.selectionErr != nil {
		t.Fatalf("unexpected selectionErr: %v", p.selectionErr)
	}
	if p.totalPrice != 10 {
		t.Errorf("totalPrice = %d, want 10 (10/张 × 1 张)", p.totalPrice)
	}
	if p.priceMode != "fixed" || p.priceStatus != "agreed" {
		t.Errorf("priceMode/Status = %q/%q, want fixed/agreed", p.priceMode, p.priceStatus)
	}
	if p.priceUnit != "per_photo" {
		t.Errorf("priceUnit = %q, want per_photo (snapshot must carry the unit)", p.priceUnit)
	}
	if p.photoCount == nil || *p.photoCount != 1 {
		t.Errorf("photoCount = %v, want 1", p.photoCount)
	}
}

func TestDeriveBookingPricing_PerPhotoMultiplies(t *testing.T) {
	svc := repository.Service{Name: "按张", Price: int32Ptr(10), Duration: 30, PriceUnit: "per_photo"}
	three := int32(3)

	p := deriveBookingPricing(svc, &three)
	if p.selectionErr != nil {
		t.Fatalf("unexpected selectionErr: %v", p.selectionErr)
	}
	if p.totalPrice != 30 {
		t.Errorf("totalPrice = %d, want 30 (10/张 × 3 张)", p.totalPrice)
	}
	if p.photoCount == nil || *p.photoCount != 3 {
		t.Errorf("photoCount = %v, want 3", p.photoCount)
	}
}

// 套餐填了「起拍/包含张数」，下单却没填张数 → 用套餐的张数。
// 这是 services.photo_count 唯一真正生效的地方：此前这里硬编码 1 张，
// 摄影师填的 5 张起拍只出现在详情页，实际成交按 1 张算钱。
func TestDeriveBookingPricing_PerPhotoUsesPackageMinPhotos(t *testing.T) {
	five := int32(5)
	svc := repository.Service{Name: "按张", Price: int32Ptr(10), Duration: 30, PriceUnit: "per_photo", PhotoCount: &five}

	p := deriveBookingPricing(svc, nil)
	if p.selectionErr != nil {
		t.Fatalf("unexpected selectionErr: %v", p.selectionErr)
	}
	if p.totalPrice != 50 {
		t.Errorf("totalPrice = %d, want 50 (10/张 × 套餐起拍 5 张)", p.totalPrice)
	}
	if p.photoCount == nil || *p.photoCount != 5 {
		t.Errorf("photoCount = %v, want 5 (套餐起拍张数必须落进快照)", p.photoCount)
	}
}

// 套餐没填起拍张数、下单也没填 → 单张起拍，且张数照样写快照。
func TestDeriveBookingPricing_PerPhotoDefaultsToOnePhoto(t *testing.T) {
	svc := repository.Service{Name: "按张", Price: int32Ptr(10), Duration: 30, PriceUnit: "per_photo"}

	p := deriveBookingPricing(svc, nil)
	if p.selectionErr != nil {
		t.Fatalf("unexpected selectionErr: %v", p.selectionErr)
	}
	if p.totalPrice != 10 {
		t.Errorf("totalPrice = %d, want 10 (10/张 × 1 张)", p.totalPrice)
	}
	if p.photoCount == nil || *p.photoCount != 1 {
		t.Errorf("photoCount = %v, want 1 — 按张订单落库必须有确定张数，不能留 NULL", p.photoCount)
	}
}

// 下单张数少于套餐起拍张数 → 拒绝。静默按起拍价收费会让用户以为只买 1 张
// 却被收 5 张的钱，比报错糟糕得多。
func TestDeriveBookingPricing_PerPhotoRejectsBelowMinPhotos(t *testing.T) {
	five := int32(5)
	three := int32(3)
	svc := repository.Service{Name: "按张", Price: int32Ptr(10), Duration: 30, PriceUnit: "per_photo", PhotoCount: &five}

	p := deriveBookingPricing(svc, &three)
	if !errors.Is(p.selectionErr, ErrInvalidPhotoSelection) {
		t.Fatalf("selectionErr = %v, want ErrInvalidPhotoSelection (3 < 起拍 5)", p.selectionErr)
	}
	// 原因要能被人读懂：前端直接把这段文案展示给用户。
	if msg := p.selectionErr.Error(); !strings.Contains(msg, "起拍张数 5") {
		t.Errorf("error = %q, want it to mention 起拍张数 5", msg)
	}
}

// 下单张数 >= 起拍张数 → 正常按张数计价（等于起拍数也要能过）。
func TestDeriveBookingPricing_PerPhotoAcceptsAtOrAboveMinPhotos(t *testing.T) {
	five := int32(5)
	svc := repository.Service{Name: "按张", Price: int32Ptr(10), Duration: 30, PriceUnit: "per_photo", PhotoCount: &five}

	for _, n := range []int32{5, 8} {
		v := n
		p := deriveBookingPricing(svc, &v)
		if p.selectionErr != nil {
			t.Fatalf("photoCount=%d: unexpected selectionErr: %v", n, p.selectionErr)
		}
		if want := int32(10) * n; p.totalPrice != want {
			t.Errorf("photoCount=%d: totalPrice = %d, want %d", n, p.totalPrice, want)
		}
		if p.photoCount == nil || *p.photoCount != n {
			t.Errorf("photoCount=%d: snapshot = %v", n, p.photoCount)
		}
	}
}

// 收口断言：任何成功算出总价的按张订单都不允许留下空张数。
// 张数来源不同（显式 / 起拍 / 兜底）但结果必须一致 —— 这是 B3 的核心。
func TestDeriveBookingPricing_NeverLeavesPhotoCountNull(t *testing.T) {
	one := int32(1)
	five := int32(5)
	three := int32(3)

	cases := []struct {
		name       string
		photoCount *int32
		requested  *int32
		wantPhotos int32
	}{
		{"套餐无起拍 + 不填张数", nil, nil, 1},
		{"套餐起拍 1 + 不填张数", &one, nil, 1},
		{"套餐起拍 5 + 不填张数", &five, nil, 5},
		{"套餐起拍 1 + 显式 3 张", &one, &three, 3},
		{"套餐无起拍 + 显式 3 张", nil, &three, 3},
	}
	for _, c := range cases {
		svc := repository.Service{Name: "按张", Price: int32Ptr(10), Duration: 30, PriceUnit: "per_photo", PhotoCount: c.photoCount}
		p := deriveBookingPricing(svc, c.requested)
		if p.selectionErr != nil {
			t.Fatalf("%s: unexpected selectionErr: %v", c.name, p.selectionErr)
		}
		if p.photoCount == nil {
			t.Fatalf("%s: photoCount = nil — 按张订单必须带确定张数", c.name)
		}
		if *p.photoCount != c.wantPhotos {
			t.Errorf("%s: photoCount = %d, want %d", c.name, *p.photoCount, c.wantPhotos)
		}
		if want := int32(10) * c.wantPhotos; p.totalPrice != want {
			t.Errorf("%s: totalPrice = %d, want %d", c.name, p.totalPrice, want)
		}
	}
}

func TestDeriveBookingPricing_PerPhotoRejectsNonPositiveCount(t *testing.T) {
	svc := repository.Service{Name: "按张", Price: int32Ptr(10), Duration: 30, PriceUnit: "per_photo"}

	for _, n := range []int32{0, -1} {
		v := n
		p := deriveBookingPricing(svc, &v)
		if !errors.Is(p.selectionErr, ErrInvalidPhotoSelection) {
			t.Errorf("photoCount=%d: selectionErr = %v, want ErrInvalidPhotoSelection", n, p.selectionErr)
		}
	}
}

func TestDeriveBookingPricing_PerPhotoOverflowRejected(t *testing.T) {
	// int32 溢出会算出一个负总价，那比拒绝下单危险得多。
	svc := repository.Service{Name: "天价", Price: int32Ptr(1 << 30), Duration: 30, PriceUnit: "per_photo"}
	big := int32(3)

	p := deriveBookingPricing(svc, &big)
	if !errors.Is(p.selectionErr, ErrInvalidPhotoSelection) {
		t.Errorf("selectionErr = %v, want ErrInvalidPhotoSelection on overflow", p.selectionErr)
	}
}

// 非按张的计价单位必须保持历史行为：total_price 就是单价本身，不做任何乘法。
// 尤其 per_hour：若在这里乘 duration，「4 小时 299」会被算成 1196。
func TestDeriveBookingPricing_NonPerPhotoKeepsUnitPrice(t *testing.T) {
	cases := []struct {
		unit string
	}{
		{"fixed"},
		{"per_hour"},
		{"per_day"},
	}
	for _, c := range cases {
		svc := repository.Service{Name: "套餐", Price: int32Ptr(299), Duration: 240, PriceUnit: c.unit}
		p := deriveBookingPricing(svc, nil)
		if p.selectionErr != nil {
			t.Fatalf("%s: unexpected selectionErr: %v", c.unit, p.selectionErr)
		}
		if p.totalPrice != 299 {
			t.Errorf("%s: totalPrice = %d, want 299 (unit price, no multiplier)", c.unit, p.totalPrice)
		}
		if p.photoCount != nil {
			t.Errorf("%s: photoCount = %v, want nil", c.unit, *p.photoCount)
		}
		if p.priceUnit != c.unit {
			t.Errorf("%s: priceUnit = %q", c.unit, p.priceUnit)
		}
	}
}

func TestDeriveBookingPricing_NegotiableAndMutualUnchanged(t *testing.T) {
	neg := deriveBookingPricing(repository.Service{Name: "面议", Price: nil, Duration: 60, PriceUnit: "fixed"}, nil)
	if neg.priceMode != "negotiable" || neg.priceStatus != "awaiting_quote" || neg.totalPrice != 0 {
		t.Errorf("negotiable = %+v, want negotiable/awaiting_quote/0", neg)
	}

	zero := int32(0)
	mut := deriveBookingPricing(repository.Service{Name: "互勉", Price: &zero, Duration: 60, PriceUnit: "fixed"}, nil)
	if mut.priceMode != "mutual" || mut.totalPrice != 0 {
		t.Errorf("mutual = %+v, want mutual/0", mut)
	}
}

// 000041 之前的历史行 price_unit 都是 'fixed'；Go 侧读到空串只可能是漏 SELECT，
// 兜底成 fixed 才与数据库默认值一致，不会让金额变化。
func TestDeriveBookingPricing_EmptyUnitFallsBackToFixed(t *testing.T) {
	p := deriveBookingPricing(repository.Service{Name: "老套餐", Price: int32Ptr(399), Duration: 120}, nil)
	if p.priceUnit != "fixed" {
		t.Errorf("priceUnit = %q, want fixed fallback", p.priceUnit)
	}
	if p.totalPrice != 399 {
		t.Errorf("totalPrice = %d, want 399", p.totalPrice)
	}
}

func TestPriceUnitValidation(t *testing.T) {
	valid := []string{"fixed", "per_photo", "per_hour", "per_day"}
	for _, u := range valid {
		got, err := normalizePriceUnit(u, false)
		if err != nil || got == nil || *got != u {
			t.Errorf("normalizePriceUnit(%q) = %v, %v; want %q", u, got, err, u)
		}
	}

	if _, err := normalizePriceUnit("per_kg", false); !errors.Is(err, ErrInvalidPriceUnit) {
		t.Errorf("invalid unit must be rejected, got %v", err)
	}

	// 空串在创建时补 fixed，在更新时返回 nil（让 SQL COALESCE 保留库里的原值）——
	// 老客户端编辑套餐名不能把按张套餐退回整套价。
	created, err := normalizePriceUnit("", false)
	if err != nil || created == nil || *created != "fixed" {
		t.Errorf("create with empty unit = %v, %v; want fixed", created, err)
	}
	updated, err := normalizePriceUnit("", true)
	if err != nil || updated != nil {
		t.Errorf("update with empty unit = %v, %v; want nil (keep stored value)", updated, err)
	}
}

func TestValidatePricing(t *testing.T) {
	zero := int32(0)
	ten := int32(10)
	neg := int32(-5)
	perPhoto := "per_photo"
	perHour := "per_hour"
	one := int32(1)
	badCount := int32(0)

	cases := []struct {
		name  string
		price *int32
		unit  *string
		count *int32
		want  error
	}{
		{"fixed 0 元互勉", &zero, nil, nil, nil},
		{"fixed 正常价", &ten, nil, nil, nil},
		{"负价", &neg, nil, nil, ErrInvalidPrice},
		{"按张 10/张", &ten, &perPhoto, nil, nil},
		{"按张 0 元", &zero, &perPhoto, nil, ErrPerPhotoPriceRequired},
		{"按张 无价", nil, &perPhoto, nil, ErrPerPhotoPriceRequired},
		{"按张 带张数", &ten, &perPhoto, &one, nil},
		{"张数 0", &ten, nil, &badCount, ErrInvalidPhotoCount},
		{"时价不要求乘数", &ten, &perHour, nil, nil},
	}
	for _, c := range cases {
		if got := validatePricing(c.price, c.unit, c.count); !errors.Is(got, c.want) && !(got == nil && c.want == nil) {
			t.Errorf("%s: validatePricing = %v, want %v", c.name, got, c.want)
		}
	}
}

// 按张计价必须真的走通 Create：参数顺序（photo_count / price_unit 是第 13、14 个）
// 一旦错位，落库的就是别的字段，而这种错位单测以外很难发现。
func TestCreateBooking_PerPhotoWritesTotalAndSnapshot(t *testing.T) {
	owner := int64(2)
	db := &bookingRecorder{rows: []pgx.Row{
		fakeRow{err: errors.New("no photographer profile")},
		fakeRow{values: []any{int64(0)}},
		fakeRow{values: []any{owner}},
		fakeRow{values: []any{int64(1), "按张套餐", int32Ptr(10), (*string)(nil), int32(30), "per_photo", (*int32)(nil)}},
		fakeRow{values: bookingScanValues(11, "pending")},
		fakeRow{err: errors.New("no photographer profile")},
	}}
	svc := NewBookingService(repository.New(db))
	three := int32(3)

	if _, err := svc.Create(context.Background(), CreateBookingRequest{
		PhotographerID: 2,
		CoserID:        5,
		ServiceID:      1,
		Date:           futureDate(1),
		Time:           "10:00",
		PhotoCount:     &three,
	}); err != nil {
		t.Fatalf("Create unexpected error: %v", err)
	}

	if got := createBookingInt32Arg(t, db, 6); got != 30 {
		t.Errorf("TotalPrice = %d, want 30 (10/张 × 3 张)", got)
	}
	if got := createBookingArg(t, db, 12); got == nil {
		t.Error("photo_count must be written, got nil")
	} else if n, ok := got.(*int32); !ok || n == nil || *n != 3 {
		t.Errorf("photo_count = %v, want *3", got)
	}
	if got := createBookingArg(t, db, 13); got != "per_photo" {
		t.Errorf("price_unit = %v, want per_photo", got)
	}
}

// 套餐是按张计价，但下单走的是「不填张数」这条老路径时，必须按套餐起拍张数
// 落单并写快照 —— 这条用例从 Create 入口验证 B2，而不只是纯函数。
func TestCreateBooking_PerPhotoUsesPackageMinPhotos(t *testing.T) {
	owner := int64(2)
	// RETURNING 回来的订单也要带上 5 张 + per_photo，否则响应里读到的快照是空的。
	created := bookingScanValues(12, "pending")
	created[25] = int32Ptr(5) // photo_count
	created[26] = "per_photo" // price_unit

	db := &bookingRecorder{rows: []pgx.Row{
		fakeRow{err: errors.New("no photographer profile")},
		fakeRow{values: []any{int64(0)}},
		fakeRow{values: []any{owner}},
		// GetServiceById: id, name, price, description, duration, price_unit, photo_count
		fakeRow{values: []any{int64(1), "五张起拍", int32Ptr(10), (*string)(nil), int32(30), "per_photo", int32Ptr(5)}},
		fakeRow{values: created},
		fakeRow{err: errors.New("no photographer profile")},
	}}
	svc := NewBookingService(repository.New(db))

	data, err := svc.Create(context.Background(), CreateBookingRequest{
		PhotographerID: 2,
		CoserID:        5,
		ServiceID:      1,
		Date:           futureDate(1),
		Time:           "10:00",
	})
	if err != nil {
		t.Fatalf("Create unexpected error: %v", err)
	}

	if got := createBookingInt32Arg(t, db, 6); got != 50 {
		t.Errorf("TotalPrice = %d, want 50 (10/张 × 套餐起拍 5 张)", got)
	}
	if got := createBookingArg(t, db, 12); got == nil {
		t.Error("photo_count must be written even when the client sent nothing")
	} else if n, ok := got.(*int32); !ok || n == nil || *n != 5 {
		t.Errorf("photo_count = %v, want *5", got)
	}
	// 张数已确定，就不能再标「待议定」。
	if data.PricePending {
		t.Error("pricePending must stay false for a fully priced per_photo booking")
	}
	if data.PhotoCount == nil || *data.PhotoCount != 5 {
		t.Errorf("response photoCount = %v, want 5", data.PhotoCount)
	}
}

func TestCreateBooking_PerPhotoZeroCountRejected(t *testing.T) {
	owner := int64(2)
	db := &bookingRecorder{rows: []pgx.Row{
		fakeRow{err: errors.New("no photographer profile")},
		fakeRow{values: []any{int64(0)}},
		fakeRow{values: []any{owner}},
		fakeRow{values: []any{int64(1), "按张套餐", int32Ptr(10), (*string)(nil), int32(30), "per_photo", (*int32)(nil)}},
	}}
	svc := NewBookingService(repository.New(db))
	zero := int32(0)

	_, err := svc.Create(context.Background(), CreateBookingRequest{
		PhotographerID: 2,
		CoserID:        5,
		ServiceID:      1,
		Date:           futureDate(1),
		Time:           "10:00",
		PhotoCount:     &zero,
	})
	if !errors.Is(err, ErrInvalidPhotoSelection) {
		t.Fatalf("err = %v, want ErrInvalidPhotoSelection", err)
	}
	for _, a := range db.args {
		if len(a) == 14 {
			t.Fatal("a 0-shot booking must be rejected before the INSERT")
		}
	}
}
