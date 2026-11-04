package service

import (
	"context"
	"errors"
	"math"
	"strings"

	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/jackc/pgx/v5"
)

// PhotographerListItem is the list-page row: the shared photographer fields plus the
// price roll-up the card renders (「¥399 起 / 互勉 / 面议 / 暂未设置」). It is a
// separate type on purpose — adding these to PhotographerItem itself would leak zero
// values into the home and detail responses, which never fetch them.
type PhotographerListItem struct {
	PhotographerItem
	MinPrice         *int32 `json:"minPrice"`         // 最低整套价；无整套价时 null
	MinPricePerPhoto *int32 `json:"minPricePerPhoto"` // 按张最低单价；无按张套餐时 null
	HasFree          bool   `json:"hasFree"`          // 有 0 元（互勉）套餐
	HasNegotiable    bool   `json:"hasNegotiable"`    // 有面议套餐
	ServiceCount     int32  `json:"serviceCount"`     // 上架套餐数；0 = 暂未设置
	// IsSelf 标记「这条就是我自己的档案」，列表页据此渲染「这是你」角标。
	// 刻意不在 List()/SQL 里算：那会让缓存按访问者分裂（每个登录用户一份），
	// 也会把匿名与登录切成两套 key。改由 handler 在 c.JSON 之前按当前 token 现算。
	IsSelf bool `json:"isSelf"`
}

type PhotographerListResponse struct {
	List     []PhotographerListItem `json:"list"`
	Total    int                    `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"pageSize"`
}

type PhotographerDetail struct {
	PhotographerItem
	Description string        `json:"description"`
	Services    []ServiceItem `json:"services"`
	Works       []WorkItem    `json:"works"`
	Reviews     []ReviewItem  `json:"reviews"`
}

// ServiceItem is the public shape of a package, used by the photographer detail
// page and the platform-template list. It deliberately carries no shelf state:
// templates are not sellable, and the detail query only returns packages that are
// on the shelf anyway — emitting a zero-valued isActive there would be a lie.
type ServiceItem struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Price       *int32 `json:"price"`
	Description string `json:"description"`
	Duration    int32  `json:"duration"`
	// PriceUnit 决定 Price 的含义：fixed=整套价、per_photo=按张、per_hour=按小时、
	// per_day=按天。前端必须按它给价格加单位后缀，否则 ¥10/张 会被读成 ¥10 一整套。
	PriceUnit string `json:"priceUnit"`
	// PhotoCount 仅 per_photo 有意义：该套餐起拍/包含的张数；null = 按需议定。
	PhotoCount *int32 `json:"photoCount"`
}

// MyServiceItem is what the owner sees in 我的套餐: the public fields plus shelf
// state, so the page can render the real 上架/下架 tag and offer the toggle.
type MyServiceItem struct {
	ServiceItem
	IsActive  bool  `json:"isActive"`
	SortOrder int32 `json:"sortOrder"`
}

type ReviewItem struct {
	ID               int64    `json:"id"`
	PhotographerID   int32    `json:"photographerId"`
	PhotographerName string   `json:"photographerName,omitempty"`
	UserID           int32    `json:"userId"`
	UserName         string   `json:"userName"`
	UserAvatar       string   `json:"userAvatar"`
	Rating           int32    `json:"rating"`
	Content          string   `json:"content"`
	Images           []string `json:"images"`
	CreatedAt        string   `json:"createdAt"`
}

type PhotographerService struct {
	queries *repository.Queries
	works   worksStore
	users   userNameSync
}

// userNameSync 是「账号昵称」侧的最小写入口，*repository.UserRepo 满足它。
// 单独留一个缝而不是塞进 worksStore：Activate/UpdateProfile 的测试只关心
// 「有没有把名字同步过去」，不该为了它拉起 pgxpool。为 nil 时同步动作静默跳过
// （例如只装配了 Queries 的单元测试）。
type userNameSync interface {
	UpdateNickname(ctx context.Context, userID int64, name string) error
}

// worksStore is the works-relevant subset of *repository.Queries; the
// PhotographerService falls back to queries when works is nil. Tests inject
// fakes through this seam without a database.
type worksStore interface {
	GetPhotographerByUserID(ctx context.Context, userID int64) (repository.PhotographerWithTags, error)
	UpdatePhotographerProfile(ctx context.Context, id int64, name, description, location, mode string, mutualIntro *string, avatar string) error
	InsertWork(ctx context.Context, photographerID int64, title string, images []string, description string) (int64, error)
	GetWorksByPhotographer(ctx context.Context, photographerID int32) ([]repository.Work, error)
	GetAllWorksByPhotographer(ctx context.Context, photographerID int32) ([]repository.Work, error)
	GetWorkPhotographerID(ctx context.Context, id int64) (int32, error)
	DeleteWorkByID(ctx context.Context, id int64) error
	DeleteWorkByIDAndPhotographer(ctx context.Context, id, photographerID int64) error
	UpdateWorkByIDAndPhotographer(ctx context.Context, id, photographerID int64, title string, images []string, description string) error
}

func (s *PhotographerService) workStore() worksStore {
	if s.works != nil {
		return s.works
	}
	return s.queries
}

func NewPhotographerService(queries *repository.Queries) *PhotographerService {
	return &PhotographerService{queries: queries}
}

// WithUserNames 注入账号昵称的同步器（*repository.UserRepo）。main 里装配一次；
// 不注入时摄影师名照常写入 photographers，只是不会再同步 users.name。
func (s *PhotographerService) WithUserNames(users userNameSync) *PhotographerService {
	s.users = users
	return s
}

var (
	ErrAlreadyActivated  = errors.New("photographer already activated")
	ErrWorkNotFound      = errors.New("work not found")
	ErrWorkForbidden     = errors.New("work belongs to another photographer")
	ErrInvalidMode       = errors.New("invalid mode")
	ErrProfileNotFound   = errors.New("photographer profile not found")
	ErrServiceNotFound   = errors.New("service not found")
	ErrServiceForbidden  = errors.New("cannot modify another photographer's service")
	ErrInvalidPrice      = errors.New("invalid price")
	ErrInvalidPriceUnit  = errors.New("invalid price unit")
	ErrInvalidPhotoCount = errors.New("invalid photo count")
	// ErrPerPhotoPriceRequired 与 ErrInvalidPrice 分开：按张 0 元 与 负价 的修法不同
	// （前者是「必须填一个正单价」，后者是「价格不能是负数」），复用一条文案会把用户带偏。
	ErrPerPhotoPriceRequired = errors.New("per_photo price must be > 0")
)

// ProfileUpdate is the self-service profile edit payload.
type ProfileUpdate struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Location    string  `json:"location"`
	Mode        string  `json:"mode"`
	MutualIntro *string `json:"mutualIntro"`
	Avatar      string  `json:"avatar"`
}

type ServiceUpsertRequest struct {
	Name        string `json:"name" binding:"required"`
	Price       *int32 `json:"price"`
	Description string `json:"description"`
	Duration    int32  `json:"duration"`
	// IsActive / SortOrder are optional. On update an omitted field keeps its stored
	// value (the statement COALESCEs nil); on create it falls back to published / 0.
	IsActive  *bool  `json:"isActive"`
	SortOrder *int32 `json:"sortOrder"`
	// PriceUnit 空字符串 = 未指定：创建时默认 fixed，更新时保持库里原值。
	PriceUnit string `json:"priceUnit"`
	// PhotoCount 仅 per_photo 有意义，可选（nil = 按需议定）。填了必须 > 0。
	PhotoCount *int32 `json:"photoCount"`
}

// priceUnits 是 services.price_unit 的合法取值，与迁移 000041 的 CHECK 约束一一对应。
// 数据库层也挡，但这里先挡能给出 400 + 明确文案，而不是把 23514 变成 500。
var priceUnits = map[string]bool{
	"fixed":     true,
	"per_photo": true,
	"per_hour":  true,
	"per_day":   true,
}

// normalizePriceUnit 校验并归一化计价单位。空字符串代表「未指定」：创建时按 fixed
// 处理，更新时返回 nil 让 SQL 的 COALESCE 保留库里原值 —— 老客户端（不知道
// priceUnit 存在）编辑套餐名时，不能把一个按张计价的套餐悄悄退回整套价。
func normalizePriceUnit(unit string, forUpdate bool) (*string, error) {
	if unit == "" {
		if forUpdate {
			return nil, nil
		}
		v := "fixed"
		return &v, nil
	}
	if !priceUnits[unit] {
		return nil, ErrInvalidPriceUnit
	}
	return &unit, nil
}

// validatePricing 是定价语义的唯一入口，create 与 update 共用：
//
//   - price 为负 → ErrInvalidPrice（历史规则，保留）
//   - 按张计价必须 price > 0 —— 0 元/张 等于免费且张数无限，是录入错误而不是营销；
//     真要免费请用 fixed + 0 元（互勉）
//   - photoCount 若填必须 > 0（0 张的套餐不可下单）
func validatePricing(price *int32, unit *string, photoCount *int32) error {
	if price != nil && *price < 0 {
		return ErrInvalidPrice
	}
	if photoCount != nil && *photoCount <= 0 {
		return ErrInvalidPhotoCount
	}
	if unit != nil && *unit == "per_photo" {
		if price == nil || *price <= 0 {
			return ErrPerPhotoPriceRequired
		}
	}
	return nil
}

// Activate 幂等：重复调用不会重复建档案，返回的永远是同一个 id。
//
// 「幂等」不等于「丢弃入参」：老用户改了昵称再点一次「成为摄影师」，旧实现把
// 新名字整条扔掉，档案永远停在第一次激活时的值（线上症状：列表里一直是 '11'）。
// 所以命中已有档案时按入参增量覆盖，但**只覆盖占位符名字**，见 refreshActivatedProfile。
func (s *PhotographerService) Activate(ctx context.Context, userID int64, name string, mode string, intro string) (int64, error) {
	existing, err := s.queries.GetPhotographerByUserID(ctx, userID)
	if err == nil {
		if err := s.refreshActivatedProfile(ctx, existing, name, mode, intro); err != nil {
			return 0, err
		}
		return int64(existing.ID), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	var desc *string
	if name != "" {
		desc = &name
	}
	var introPtr *string
	if intro != "" {
		introPtr = &intro
	}
	p, err := s.queries.InsertPhotographer(ctx, repository.InsertPhotographerParams{
		Name: name, Description: desc, Mode: mode, MutualIntro: introPtr, UserID: &userID,
	})
	if err != nil {
		return 0, err
	}
	// 首次激活是「本人主动填的对外展示名」，同步给账号昵称，避免「我的」页还叫
	// 「微信用户」而列表里已是新名字。方向恒为 photographers → users。
	s.syncAccountName(ctx, userID, name)
	return int64(p.ID), nil
}

// refreshActivatedProfile 处理「已有档案的重复激活」。逐字段规则：
//
//   - name：**只有当前展示名是占位符**（空白 / 纯数字 / 「微信用户」）时才用入参覆盖。
//     摄影师自己起的艺名是核心品牌资产，不能被一次重复提交换掉。
//   - mode / intro：非空且不同才覆盖（空串 = 本次没填，保持原值）。
func (s *PhotographerService) refreshActivatedProfile(ctx context.Context, existing repository.PhotographerWithTags, name, mode, intro string) error {
	newName := existing.Name
	nameOverridden := false
	if name != "" && name != existing.Name && isPlaceholderName(existing.Name) {
		newName = name
		nameOverridden = true
	}
	newMode := existing.Mode
	if mode != "" && mode != existing.Mode {
		if mode != "free" && mode != "pay" && mode != "both" {
			return ErrInvalidMode
		}
		newMode = mode
	}
	newIntro := existing.MutualIntro
	if intro != "" && (existing.MutualIntro == nil || intro != *existing.MutualIntro) {
		newIntro = &intro
	}
	if newName == existing.Name && newMode == existing.Mode && introSame(newIntro, existing.MutualIntro) {
		return nil // 一个字都没变，不必白跑一次 UPDATE
	}

	avatar := derefString(existing.Avatar)
	if err := s.workStore().UpdatePhotographerProfile(ctx, int64(existing.ID), newName,
		derefString(existing.Description), derefString(existing.Location), newMode, newIntro, avatar); err != nil {
		if errors.Is(err, repository.ErrProfileNotFound) {
			return ErrProfileNotFound
		}
		return err
	}
	if nameOverridden {
		// 这里 photographers.name 已在上面单独写过，只需把 users.name 补齐。
		s.syncAccountName(ctx, derefInt64(existing.UserID), newName)
	}
	return nil
}

// isPlaceholderName 判断展示名是不是「明显误填 / 占位」而不是真人艺名：
//   - 空白
//   - 纯数字（线上那条 '11'）
//   - 微信授权默认昵称「微信用户」
//   - '摄影师' + 纯数字（如「摄影师15」，迁移 000042 给两边都是占位值的档案
//     落的系统兜底名；它必须可被覆盖，否则用户填了真名也改不掉）
//
// 只有这种名字才允许被账号昵称或重复激活覆盖。
func isPlaceholderName(name string) bool {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" || trimmed == "微信用户" {
		return true
	}
	if rest, ok := strings.CutPrefix(trimmed, "摄影师"); ok {
		return isDigits(rest)
	}
	return isDigits(trimmed)
}

// isDigits 判断 s 非空且全部是 ASCII 数字（"" 返回 false）。
func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// syncAccountName 只写 users.name（账号昵称），永远不写 photographers.name。
// 方向是单向的：摄影师主动改艺名 / 首次激活 → 账号昵称跟随。
// 反向（普通用户改账号昵称 → 覆盖摄影师艺名）被刻意禁止，否则任何人改自己
// 昵称就等于「改名」别人的摄影师档案。
func (s *PhotographerService) syncAccountName(ctx context.Context, userID int64, name string) {
	if s.users == nil || userID == 0 || name == "" {
		return
	}
	_ = s.users.UpdateNickname(ctx, userID, name)
}

func introSame(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func photographerItemFromRow(p repository.PhotographerWithTags) *PhotographerItem {
	tags := p.Tags
	if tags == nil {
		tags = []string{}
	}

	return &PhotographerItem{
		ID:          int64(p.ID),
		Name:        p.Name,
		Avatar:      derefString(p.Avatar),
		Location:    derefString(p.Location),
		Description: derefString(p.Description),
		Rating:      math.Round(numericToFloat64(p.Rating)*10) / 10,
		ReviewCount: derefInt32(p.ReviewCount),
		OrderCount:  derefInt32(p.OrderCount),
		UserID:      derefInt64(p.UserID),
		Mode:        p.Mode,
		MutualIntro: derefString(p.MutualIntro),
		Certified:   p.Certified,
		Tags:        tags,
	}
}

func (s *PhotographerService) GetByUser(ctx context.Context, userID int64) (*PhotographerItem, error) {
	p, err := s.queries.GetPhotographerByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return photographerItemFromRow(p), nil
}

// MyProfile returns the caller's own photographer profile; non-photographers get ErrNotPhotographer.
func (s *PhotographerService) MyProfile(ctx context.Context, userID int64) (*PhotographerItem, error) {
	p, err := s.workStore().GetPhotographerByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotPhotographer
		}
		return nil, err
	}
	return photographerItemFromRow(p), nil
}

func (s *PhotographerService) UpdateProfile(ctx context.Context, userID int64, req ProfileUpdate) error {
	store := s.workStore()
	profile, err := store.GetPhotographerByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotPhotographer
		}
		return err
	}
	if req.Mode != "free" && req.Mode != "pay" && req.Mode != "both" {
		return ErrInvalidMode
	}
	avatar := req.Avatar
	if avatar == "" {
		avatar = derefString(profile.Avatar)
	}
	if err := store.UpdatePhotographerProfile(ctx, int64(profile.ID), req.Name, req.Description, req.Location, req.Mode, req.MutualIntro, avatar); err != nil {
		if errors.Is(err, repository.ErrProfileNotFound) {
			return ErrProfileNotFound
		}
		return err
	}
	// 摄影师名是「对外展示名」，改完让账号昵称跟上（方向：photographers → users），
	// 否则用户在「我的」页还叫「微信用户」、在列表里叫新名字，看起来像两个人。
	s.syncAccountName(ctx, userID, req.Name)
	return nil
}

func (s *PhotographerService) HiddenUserIDs(ctx context.Context, userID int64) ([]int64, error) {
	return s.queries.ListHiddenUserIDs(ctx, userID)
}

// List 返回摄影师列表。sort 是列表页筛选栏的排序键：
// ""/"all" 默认（评分优先）、"hot" 热门（评价数）、"rating" 评分最高、
// "order" 接单最多、"new" 最新入驻。未知值一律退回默认排序（不让调用方拼 SQL）。
func (s *PhotographerService) List(ctx context.Context, keyword, location, tag string, page, size int, excludeUserIDs []int64, sort string) (*PhotographerListResponse, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 50 {
		size = 10
	}
	offset := (page - 1) * size

	var kw, loc, tg *string
	if keyword != "" {
		kw = &keyword
	}
	if location != "" {
		loc = &location
	}
	if tag != "" {
		tg = &tag
	}
	if excludeUserIDs == nil {
		excludeUserIDs = []int64{}
	}
	var sortKey *string
	switch sort {
	case "hot", "rating", "order", "new":
		sortKey = &sort
	}

	// For search, fetch extra to estimate total (if fewer than limit returned, total is known)
	photographers, err := s.queries.SearchPhotographers(ctx, repository.SearchPhotographersParams{
		Keyword:        kw,
		Location:       loc,
		TagName:        tg,
		Limit:          int32(size + 1),
		Offset:         int32(offset),
		ExcludeUserIDs: excludeUserIDs,
		Sort:           sortKey,
	})
	if err != nil {
		return nil, err
	}

	hasMore := len(photographers) > size
	if hasMore {
		photographers = photographers[:size]
	}

	items := make([]PhotographerListItem, 0, len(photographers))
	for _, it := range mapPhotographers(photographers) {
		items = append(items, PhotographerListItem{PhotographerItem: it})
	}
	// 一次批量取回本页的价格汇总（不是逐行查询）；无上架套餐的摄影师不会出现在结果里，
	// 保持零值 → 前端显示「暂未设置」。
	ids := make([]int64, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}
	if len(ids) > 0 {
		sums, err := s.queries.GetServicePriceSummaries(ctx, ids)
		if err != nil {
			return nil, err
		}
		items = mergePriceSummaries(items, sums)
	}
	s.attachCredit(ctx, items)

	total := offset + len(photographers)
	if hasMore {
		total = offset + size + 1
	}

	return &PhotographerListResponse{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: size,
	}, nil
}

// MarkSelf 给列表打「这是你」标记。刻意做成纯函数、由 handler 在 c.JSON 之前
// 调用，而不是在 List() 里按 viewer 计算：
//   - List() 的结果要进 Redis 缓存，缓存键里没有 viewer → 一旦把 IsSelf 算进
//     缓存内容，A 用户会看到 B 用户的「这是你」；
//   - 反过来把 viewer 塞进缓存键，缓存会被切成「每个登录用户一份」，
//     匿名与登录还会分裂成两套 key，等于把原来的缓存全废掉。
//
// viewerUserID <= 0（未登录）时保持字段原样（全 false）。
func MarkSelf(items []PhotographerListItem, viewerUserID int64) []PhotographerListItem {
	if viewerUserID <= 0 {
		return items
	}
	for i := range items {
		items[i].IsSelf = items[i].UserID != 0 && items[i].UserID == viewerUserID
	}
	return items
}

// mergePriceSummaries attaches the price roll-up to a page of list rows. Pure, so it
// is unit-testable without a database.
func mergePriceSummaries(items []PhotographerListItem, sums []repository.ServicePriceSummary) []PhotographerListItem {
	byID := make(map[int64]repository.ServicePriceSummary, len(sums))
	for _, s := range sums {
		byID[s.PhotographerID] = s
	}
	for i := range items {
		s, ok := byID[items[i].ID]
		if !ok {
			continue
		}
		items[i].MinPrice = s.MinPrice
		items[i].MinPricePerPhoto = s.MinPricePerPhoto
		items[i].HasFree = s.HasFree
		items[i].HasNegotiable = s.HasNegotiable
		items[i].ServiceCount = s.ActiveCount
	}
	return items
}

func (s *PhotographerService) GetDetail(ctx context.Context, id int32) (*PhotographerDetail, error) {
	p, err := s.queries.GetPhotographerById(ctx, id)
	if err != nil {
		return nil, err
	}

	services, err := s.queries.GetActiveServicesByPhotographer(ctx, id)
	if err != nil {
		services = []repository.Service{}
	}

	works, err := s.queries.GetWorksByPhotographer(ctx, id)
	if err != nil {
		works = []repository.Work{}
	}

	reviews, err := s.queries.GetReviewsByPhotographer(ctx, id)
	if err != nil {
		reviews = []repository.Review{}
	}

	tags := p.Tags
	if tags == nil {
		tags = []string{}
	}

	rating := numericToFloat64(p.Rating)

	detail := &PhotographerDetail{
		PhotographerItem: PhotographerItem{
			ID:          int64(p.ID),
			Name:        p.Name,
			Avatar:      derefString(p.Avatar),
			Location:    derefString(p.Location),
			Rating:      math.Round(rating*10) / 10,
			ReviewCount: derefInt32(p.ReviewCount),
			OrderCount:  derefInt32(p.OrderCount),
			UserID:      derefInt64(p.UserID),
			Mode:        p.Mode,
			MutualIntro: derefString(p.MutualIntro),
			Certified:   p.Certified,
			Tags:        tags,
		},
		Description: derefString(p.Description),
		Services:    mapServiceItems(services),
		Works:       mapWorkItems(works),
		Reviews:     mapReviewItems(reviews),
	}
	s.attachCreditOne(ctx, &detail.PhotographerItem)
	return detail, nil
}

func mapServiceItems(services []repository.Service) []ServiceItem {
	items := make([]ServiceItem, 0, len(services))
	for _, s := range services {
		items = append(items, ServiceItem{
			ID:          s.ID,
			Name:        s.Name,
			Price:       s.Price,
			Description: derefString(s.Description),
			Duration:    s.Duration,
			PriceUnit:   s.PriceUnit,
			PhotoCount:  s.PhotoCount,
		})
	}
	return items
}

// mapMyServiceItems is the owner-facing mapper: the public fields plus shelf state.
// Only the `mine` query selects is_active / sort_order, so only this mapper may
// read them.
func mapMyServiceItems(services []repository.Service) []MyServiceItem {
	items := make([]MyServiceItem, 0, len(services))
	for _, s := range services {
		items = append(items, MyServiceItem{
			ServiceItem: ServiceItem{
				ID:          s.ID,
				Name:        s.Name,
				Price:       s.Price,
				Description: derefString(s.Description),
				Duration:    s.Duration,
				PriceUnit:   s.PriceUnit,
				PhotoCount:  s.PhotoCount,
			},
			IsActive:  s.IsActive,
			SortOrder: s.SortOrder,
		})
	}
	return items
}

func mapWorkItems(works []repository.Work) []WorkItem {
	items := make([]WorkItem, 0, len(works))
	for _, w := range works {
		images := w.Images
		if images == nil {
			images = []string{}
		}
		items = append(items, WorkItem{
			ID:               w.ID,
			Title:            w.Title,
			Images:           images,
			PhotographerName: "",
			Description:      derefString(w.Description),
			Status:           w.Status,
			CreatedAt:        w.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return items
}

func mapReviewItems(reviews []repository.Review) []ReviewItem {
	items := make([]ReviewItem, 0, len(reviews))
	for _, r := range reviews {
		images := r.Images
		if images == nil {
			images = []string{}
		}
		items = append(items, ReviewItem{
			ID:               r.ID,
			PhotographerID:   r.PhotographerID,
			PhotographerName: r.PhotographerName,
			UserID:           r.UserID,
			UserName:         derefString(r.UserName),
			UserAvatar:       derefString(r.UserAvatar),
			Rating:           r.Rating,
			Content:          derefString(r.Content),
			Images:           images,
			CreatedAt:        r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return items
}

func (s *PhotographerService) CreateWork(ctx context.Context, userID int64, title string, images []string, description string) (int64, error) {
	profile, err := s.workStore().GetPhotographerByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNotPhotographer
		}
		return 0, err
	}
	return s.workStore().InsertWork(ctx, int64(profile.ID), title, images, description)
}

func (s *PhotographerService) UpdateWork(ctx context.Context, userID, workID int64, title string, images []string, description string) error {
	profile, err := s.workStore().GetPhotographerByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotPhotographer
		}
		return err
	}

	err = s.workStore().UpdateWorkByIDAndPhotographer(ctx, workID, int64(profile.ID), title, images, description)
	if err == nil {
		return nil
	}
	if !errors.Is(err, repository.ErrWorkNotFound) {
		return err
	}

	ownerID, ownerErr := s.workStore().GetWorkPhotographerID(ctx, workID)
	if ownerErr != nil {
		if errors.Is(ownerErr, pgx.ErrNoRows) {
			return ErrWorkNotFound
		}
		return ownerErr
	}
	if int64(ownerID) != int64(profile.ID) {
		return ErrWorkForbidden
	}
	return ErrWorkNotFound
}

func (s *PhotographerService) MyWorks(ctx context.Context, userID int64) ([]WorkItem, error) {
	profile, err := s.workStore().GetPhotographerByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotPhotographer
		}
		return nil, err
	}
	// Own management list is unfiltered so downed works stay visible/recoverable.
	works, err := s.workStore().GetAllWorksByPhotographer(ctx, profile.ID)
	if err != nil {
		return nil, err
	}
	return mapWorkItems(works), nil
}

func (s *PhotographerService) DeleteWork(ctx context.Context, userID, workID int64) error {
	profile, err := s.workStore().GetPhotographerByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotPhotographer
		}
		return err
	}
	ownerID, err := s.workStore().GetWorkPhotographerID(ctx, workID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrWorkNotFound
		}
		return err
	}
	if int64(ownerID) != int64(profile.ID) {
		return ErrWorkForbidden
	}
	if err := s.workStore().DeleteWorkByIDAndPhotographer(ctx, workID, int64(profile.ID)); err != nil {
		if errors.Is(err, repository.ErrWorkNotFound) {
			return ErrWorkNotFound
		}
		return err
	}
	return nil
}

func (s *PhotographerService) MyServices(ctx context.Context, userID int64) ([]MyServiceItem, error) {
	p, err := s.workStore().GetPhotographerByUserID(ctx, userID)
	if err != nil {
		return nil, ErrForbidden
	}
	rows, err := s.queries.GetAllServicesByPhotographer(ctx, int32(p.ID))
	if err != nil {
		return nil, err
	}
	return mapMyServiceItems(rows), nil
}

func (s *PhotographerService) CreateService(ctx context.Context, userID int64, req ServiceUpsertRequest) (int64, error) {
	unit, err := normalizePriceUnit(req.PriceUnit, false)
	if err != nil {
		return 0, err
	}
	if err := validatePricing(req.Price, unit, req.PhotoCount); err != nil {
		return 0, err
	}
	p, err := s.workStore().GetPhotographerByUserID(ctx, userID)
	if err != nil {
		return 0, ErrForbidden
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	var order int32
	if req.SortOrder != nil {
		order = *req.SortOrder
	}
	return s.queries.InsertService(ctx, repository.InsertServiceParams{
		Name:           req.Name,
		Price:          req.Price,
		Description:    strPtr(req.Description),
		Duration:       req.Duration,
		PhotographerID: int64(p.ID),
		IsActive:       active,
		SortOrder:      order,
		PriceUnit:      *unit,
		PhotoCount:     req.PhotoCount,
	})
}

func (s *PhotographerService) UpdateService(ctx context.Context, userID int64, serviceID int64, req ServiceUpsertRequest) error {
	unit, err := normalizePriceUnit(req.PriceUnit, true)
	if err != nil {
		return err
	}
	if err := validatePricing(req.Price, unit, req.PhotoCount); err != nil {
		return err
	}
	p, err := s.workStore().GetPhotographerByUserID(ctx, userID)
	if err != nil {
		return ErrForbidden
	}
	// Deliberately no defaulting here: nil means "leave it as stored", which the
	// statement COALESCEs. Substituting true / 0 is exactly what used to re-publish
	// an unpublished package and wipe its ordering on every edit.
	n, err := s.queries.UpdateService(ctx, repository.UpdateServiceParams{
		ID:             serviceID,
		PhotographerID: int64(p.ID),
		Name:           req.Name,
		Price:          req.Price,
		Description:    strPtr(req.Description),
		Duration:       req.Duration,
		IsActive:       req.IsActive,
		SortOrder:      req.SortOrder,
		PriceUnit:      unit,
		PhotoCount:     req.PhotoCount,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		if pid, err := s.queries.GetServicePhotographerID(ctx, serviceID); err == nil && pid != nil {
			return ErrServiceForbidden
		}
		return ErrServiceNotFound
	}
	return nil
}

func (s *PhotographerService) DeleteService(ctx context.Context, userID int64, serviceID int64) error {
	p, err := s.workStore().GetPhotographerByUserID(ctx, userID)
	if err != nil {
		return ErrForbidden
	}
	n, err := s.queries.DeleteService(ctx, serviceID, int64(p.ID))
	if err != nil {
		return err
	}
	if n == 0 {
		if pid, err := s.queries.GetServicePhotographerID(ctx, serviceID); err == nil && pid != nil {
			return ErrServiceForbidden
		}
		return ErrServiceNotFound
	}
	return nil
}

// servicePricePtr 是模板价格的取址助手。生产代码里只此一处需要「字面量 → *int32」，
// 因此就近定义；名字带 service 前缀是为了避开 _test.go 里同包的 int32Ptr。
func servicePricePtr(v int32) *int32 { return &v }

// platformServiceTemplates 是「一键预填平台模板」下发的定价参考，纯静态常量。
//
// 历史实现是 `GetServices` 全表返回：把**真实摄影师的套餐**当成「平台模板」下发给
// 所有人 —— 既是数据泄漏（别人的售价/说明被整份抄走），语义也不对（那是竞品的价目表，
// 不是平台给的定价建议）。而且某个摄影师一改价，所有人的“模板”跟着变。
//
// 模板是产品文案：不入库、不可售（下单引用会因「无主套餐」被拒），改价改文案走发版，
// 不牵扯数据迁移，也不会被任何摄影师的套餐改动影响。
//
// ID 是虚拟标识（9001 起）：只用于前端列表 key 与稳定顺序，**不对应 services 表任何行**，
// 因此不复用真实套餐 id，避免前端把它当可下单的 serviceId。
//
// 三档覆盖「三天去三场漫展」的实际卖法：按场次（半天/全天）零售，或按天打包。
var platformServiceTemplates = []ServiceItem{
	{
		ID:    9001,
		Name:  "漫展单场跟拍（半天）",
		Price: servicePricePtr(299),
		Description: "单场漫展半天跟拍（约 4 小时），含 15 张精修。" +
			"一天只打算逛半天、或只拍一两套衣服时选它。",
		Duration:  240,
		PriceUnit: "fixed",
	},
	{
		ID:    9002,
		Name:  "漫展单日跟拍（全天）",
		Price: servicePricePtr(599),
		Description: "单日漫展全天跟拍（约 8 小时），含 30 张精修，含场馆内外多处取景。" +
			"客户想从入场拍到闭馆时选它。",
		Duration:  480,
		PriceUnit: "fixed",
	},
	{
		ID:    9003,
		Name:  "漫展多日通拍（3 天打包）",
		Price: servicePricePtr(1499),
		Description: "同一场漫展连续 3 天打包价，每天按全天跟拍算（3 天共约 24 小时），含 100 张精修。" +
			"比单日单订三天便宜，适合连拍三天、换多套装的客户。",
		Duration:  1440,
		PriceUnit: "fixed",
	},
}

// ListTemplates 返回平台定价模板。刻意不碰数据库：模板一旦要从表里读，就又回到
// 「谁写进去谁就是模板」的老问题。返回副本，调用方改切片不会污染全局常量。
func (s *PhotographerService) ListTemplates(ctx context.Context) ([]ServiceItem, error) {
	out := make([]ServiceItem, len(platformServiceTemplates))
	copy(out, platformServiceTemplates)
	return out, nil
}

// attachCredit fills the optional credit badge for a page of photographers in a
// single batch query — never one query per row.
func (s *PhotographerService) attachCredit(ctx context.Context, items []PhotographerListItem) {
	userIDs := make([]int64, 0, len(items))
	for _, it := range items {
		if it.UserID > 0 {
			userIDs = append(userIDs, it.UserID)
		}
	}
	if len(userIDs) == 0 {
		return
	}
	scores, err := s.queries.ListCreditScoresByUserIDs(ctx, userIDs)
	if err != nil {
		// Credit is decorative here: a failed lookup must not break the list.
		return
	}
	byUser := make(map[int64]int32, len(scores))
	for _, sc := range scores {
		byUser[sc.UserID] = sc.Score
	}
	for i := range items {
		score, ok := byUser[items[i].UserID]
		if !ok {
			continue
		}
		v := score
		items[i].CreditScore = &v
		items[i].CreditLevel = CreditLevel(score)
	}
}

// attachCreditOne is the single-row variant used by the detail page.
func (s *PhotographerService) attachCreditOne(ctx context.Context, item *PhotographerItem) {
	if item == nil || item.UserID <= 0 {
		return
	}
	scores, err := s.queries.ListCreditScoresByUserIDs(ctx, []int64{item.UserID})
	if err != nil || len(scores) == 0 {
		return
	}
	v := scores[0].Score
	item.CreditScore = &v
	item.CreditLevel = CreditLevel(v)
}
