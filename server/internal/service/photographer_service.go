package service

import (
	"context"
	"errors"
	"math"

	"github.com/Maluslock/comic/server/internal/repository"
	"github.com/jackc/pgx/v5"
)

// PhotographerListItem is the list-page row: the shared photographer fields plus the
// price roll-up the card renders (「¥399 起 / 互勉 / 面议 / 暂未设置」). It is a
// separate type on purpose — adding these to PhotographerItem itself would leak zero
// values into the home and detail responses, which never fetch them.
type PhotographerListItem struct {
	PhotographerItem
	MinPrice      *int32 `json:"minPrice"`      // 最低固定价；无固定价时 null
	HasFree       bool   `json:"hasFree"`       // 有 0 元（互勉）套餐
	HasNegotiable bool   `json:"hasNegotiable"` // 有面议套餐
	ServiceCount  int32  `json:"serviceCount"`  // 上架套餐数；0 = 暂未设置
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

var (
	ErrAlreadyActivated = errors.New("photographer already activated")
	ErrWorkNotFound     = errors.New("work not found")
	ErrWorkForbidden    = errors.New("work belongs to another photographer")
	ErrInvalidMode      = errors.New("invalid mode")
	ErrProfileNotFound  = errors.New("photographer profile not found")
	ErrServiceNotFound  = errors.New("service not found")
	ErrServiceForbidden = errors.New("cannot modify another photographer's service")
	ErrInvalidPrice     = errors.New("invalid price")
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
}

func (s *PhotographerService) Activate(ctx context.Context, userID int64, name string, mode string, intro string) (int64, error) {
	existing, err := s.queries.GetPhotographerByUserID(ctx, userID)
	if err == nil {
		return int64(existing.ID), nil // idempotent
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
	return int64(p.ID), nil
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
	return nil
}

func (s *PhotographerService) HiddenUserIDs(ctx context.Context, userID int64) ([]int64, error) {
	return s.queries.ListHiddenUserIDs(ctx, userID)
}

func (s *PhotographerService) List(ctx context.Context, keyword, location, tag string, page, size int, excludeUserIDs []int64) (*PhotographerListResponse, error) {
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

	// For search, fetch extra to estimate total (if fewer than limit returned, total is known)
	photographers, err := s.queries.SearchPhotographers(ctx, repository.SearchPhotographersParams{
		Keyword:        kw,
		Location:       loc,
		TagName:        tg,
		Limit:          int32(size + 1),
		Offset:         int32(offset),
		ExcludeUserIDs: excludeUserIDs,
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

	return &PhotographerDetail{
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
	}, nil
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
	if req.Price != nil && *req.Price < 0 {
		return 0, ErrInvalidPrice
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
	})
}

func (s *PhotographerService) UpdateService(ctx context.Context, userID int64, serviceID int64, req ServiceUpsertRequest) error {
	if req.Price != nil && *req.Price < 0 {
		return ErrInvalidPrice
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

func (s *PhotographerService) ListTemplates(ctx context.Context) ([]ServiceItem, error) {
	rows, err := s.queries.GetServices(ctx)
	if err != nil {
		return nil, err
	}
	return mapServiceItems(rows), nil
}
