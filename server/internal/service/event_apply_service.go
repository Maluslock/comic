package service

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Maluslock/comic/server/internal/repository"
)

var (
	ErrEventApplyNotPhotographer = errors.New("user is not a photographer")
	ErrEventNotFound             = errors.New("event not found")
	ErrEventEnded                = errors.New("event already ended")
	ErrEventApplyDuplicate       = errors.New("event application already exists")
	ErrEventApplyNotFound        = errors.New("event application not found")
	ErrEventApplyInvalidStatus   = errors.New("invalid review status")
	ErrEventApplyReasonRequired  = errors.New("review reason required")
	ErrEventApplyAlreadyReviewed = errors.New("event application already reviewed")
)

// eventApplyStore 是 event_photographers 的数据访问面（*repository.Queries 实现）。
type eventApplyStore interface {
	ApplyEventPhotographer(ctx context.Context, eventID, photographerID, userID int64, message *string) (repository.EventApplication, error)
	GetMyEventApplication(ctx context.Context, eventID, userID int64) (repository.EventApplication, error)
	ListMyEventApplications(ctx context.Context, userID int64) ([]repository.MyEventApplication, error)
	WithdrawEventApplication(ctx context.Context, eventID, userID int64) (int64, error)
	ListApprovedPhotographersByEvent(ctx context.Context, eventID int64) ([]repository.PhotographerWithTags, error)
	ReviveWithdrawnEventApplication(ctx context.Context, eventID, photographerID int64, message *string) (repository.EventApplication, error)
	CountApprovedByEvent(ctx context.Context, eventID int64) (int64, error)
	ListEventApplications(ctx context.Context, status string, page, size int32) ([]repository.AdminEventApplication, int64, error)
	CountEventApplications(ctx context.Context, status string) (int64, error)
	ReviewEventApplication(ctx context.Context, id int64, status, reviewReason string, adminID int64) (int64, error)
	GetEventApplicationByID(ctx context.Context, id int64) (repository.EventApplication, error)
	GetEventById(ctx context.Context, id int64) (repository.ComicEvent, error)
}

type EventApplyService struct {
	store         eventApplyStore
	photographers photographerLookup
	notifier      certNotifier
}

func NewEventApplyService(store eventApplyStore, photographers photographerLookup, notifier certNotifier) *EventApplyService {
	return &EventApplyService{store: store, photographers: photographers, notifier: notifier}
}

// MyEventApplicationDTO 是 C 端单条申请的响应体（含漫展卡片字段，前端不必二次拉取）。
type MyEventApplicationDTO struct {
	ID           int64      `json:"id"`
	EventID      int64      `json:"eventId"`
	Status       string     `json:"status"`
	Message      *string    `json:"message,omitempty"`
	ReviewReason *string    `json:"reviewReason,omitempty"`
	AppliedAt    time.Time  `json:"appliedAt"`
	ReviewedAt   *time.Time `json:"reviewedAt,omitempty"`
	EventName    string     `json:"eventName"`
	StartDate    *time.Time `json:"startDate,omitempty"`
	EndDate      *time.Time `json:"endDate,omitempty"`
	Location     *string    `json:"location,omitempty"`
	Venue        *string    `json:"venue,omitempty"`
	CoverURL     *string    `json:"coverUrl,omitempty"`
}

func toMyEventApplicationDTO(a repository.MyEventApplication) MyEventApplicationDTO {
	return MyEventApplicationDTO{
		ID:           a.ID,
		EventID:      a.EventID,
		Status:       a.Status,
		Message:      a.Message,
		ReviewReason: a.ReviewReason,
		AppliedAt:    a.AppliedAt,
		ReviewedAt:   a.ReviewedAt,
		EventName:    a.EventName,
		StartDate:    a.StartDate,
		EndDate:      a.EndDate,
		Location:     a.EventLocation,
		Venue:        a.Venue,
		CoverURL:     a.CoverURL,
	}
}

// resolvePhotographer 把 token 身份解析成摄影师档案 id。
// 未开通摄影师身份 → ErrEventApplyNotPhotographer（handler 映射 403）。
func (s *EventApplyService) resolvePhotographer(ctx context.Context, userID int64) (int64, error) {
	profile, err := s.photographers.GetPhotographerByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrEventApplyNotPhotographer
		}
		return 0, err
	}
	return int64(profile.ID), nil
}

// assertEventOpen 校验漫展存在且未结束。
// 已结束（end_date 早于 now）不允许申请入驻 —— 但列表里能查到的历史漫展仍可查看。
func (s *EventApplyService) assertEventOpen(ctx context.Context, eventID int64, now time.Time) error {
	event, err := s.store.GetEventById(ctx, eventID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrEventNotFound
		}
		return err
	}
	if event.EndDate.Before(now) {
		return ErrEventEnded
	}
	return nil
}

// Apply 提交「摄影师入驻某漫展」申请。
// 前置：必须是已激活摄影师 + 漫展存在且未结束。
// 重复：命中部分唯一索引 uniq_event_photographer_active 的 23505 → 409。
func (s *EventApplyService) Apply(ctx context.Context, userID, eventID int64, message string) (repository.EventApplication, error) {
	photographerID, err := s.resolvePhotographer(ctx, userID)
	if err != nil {
		return repository.EventApplication{}, err
	}
	if err := s.assertEventOpen(ctx, eventID, time.Now()); err != nil {
		return repository.EventApplication{}, err
	}

	var msg *string
	if trimmed := strings.TrimSpace(message); trimmed != "" {
		msg = &trimmed
	}

	app, err := s.store.ApplyEventPhotographer(ctx, eventID, photographerID, userID, msg)
	if err != nil {
		// 撞部分唯一索引 uniq_event_photographer_active（23505）。
		// withdrawn 也被该索引视为活跃，所以「撤回后再申请」先尝试复用旧行，
		// 复用不到（真的已有 pending/approved）才是 409 重复申请。
		if isUniqueViolation(err) {
			if revived, reviveErr := s.store.ReviveWithdrawnEventApplication(ctx, eventID, photographerID, msg); reviveErr == nil {
				return revived, nil
			}
			return repository.EventApplication{}, ErrEventApplyDuplicate
		}
		// 漫展/摄影师在 check 之后被删：外键 23503 → 404 语义（漫展不存在）。
		if isForeignKeyViolation(err) {
			return repository.EventApplication{}, ErrEventNotFound
		}
		return repository.EventApplication{}, err
	}
	return app, nil
}

// MyApplication 返回调用者在某漫展的最近一条申请；从未申请返回 (zero, pgx.ErrNoRows)。
func (s *EventApplyService) MyApplication(ctx context.Context, userID, eventID int64) (repository.EventApplication, error) {
	if _, err := s.resolvePhotographer(ctx, userID); err != nil {
		return repository.EventApplication{}, err
	}
	return s.store.GetMyEventApplication(ctx, eventID, userID)
}

// MyApplications 返回调用者的全部入驻申请（含已驳回/已撤回历史），按提交时间倒序。
func (s *EventApplyService) MyApplications(ctx context.Context, userID int64) ([]MyEventApplicationDTO, error) {
	if _, err := s.resolvePhotographer(ctx, userID); err != nil {
		return nil, err
	}
	rows, err := s.store.ListMyEventApplications(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]MyEventApplicationDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, toMyEventApplicationDTO(r))
	}
	return out, nil
}

// Withdraw 撤回自己的申请（pending/approved → withdrawn）。
// 没有可撤回的行时：若存在 rejected 历史 → ErrEventApplyNotFound（无可撤回项），
// 一律映射 404，避免泄漏「别人申请过什么」。
func (s *EventApplyService) Withdraw(ctx context.Context, userID, eventID int64) error {
	if _, err := s.resolvePhotographer(ctx, userID); err != nil {
		return err
	}
	n, err := s.store.WithdrawEventApplication(ctx, eventID, userID)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrEventApplyNotFound
	}
	return nil
}

// ApprovedPhotographers 返回该漫展真实的参展摄影师（approved 列表）。
func (s *EventApplyService) ApprovedPhotographers(ctx context.Context, eventID int64) ([]repository.PhotographerWithTags, error) {
	return s.store.ListApprovedPhotographersByEvent(ctx, eventID)
}

// AdminList 是管理端分页列表；page/pageSize 归一化后交给 repo。
func (s *EventApplyService) AdminList(ctx context.Context, status string, page, pageSize int) ([]repository.AdminEventApplication, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.store.ListEventApplications(ctx, status, int32(page), int32(pageSize))
}

// Review 审核一条入驻申请。
//   - 只接受 approved / rejected
//   - rejected 必须给 reason
//   - 条件更新（WHERE status='pending'）返回 0 行 → 重读区分 409（已审）/404（不存在）
//   - 结果写一条 best-effort 站内通知给申请人（失败只记日志，不影响审核结果）
func (s *EventApplyService) Review(ctx context.Context, id int64, status, reason string, adminID int64) error {
	if status != "approved" && status != "rejected" {
		return ErrEventApplyInvalidStatus
	}
	if status == "rejected" && strings.TrimSpace(reason) == "" {
		return ErrEventApplyReasonRequired
	}

	app, err := s.store.GetEventApplicationByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrEventApplyNotFound
		}
		return err
	}
	if app.Status != "pending" {
		return ErrEventApplyAlreadyReviewed
	}

	n, err := s.store.ReviewEventApplication(ctx, id, status, reason, adminID)
	if err != nil {
		return err
	}
	if n == 0 {
		// 并发复审：再读一次，仍 pending 说明是别的原因（理论上不会发生）
		cur, readErr := s.store.GetEventApplicationByID(ctx, id)
		if readErr != nil {
			if errors.Is(readErr, pgx.ErrNoRows) {
				return ErrEventApplyNotFound
			}
			return readErr
		}
		if cur.Status != "pending" {
			return ErrEventApplyAlreadyReviewed
		}
		return ErrEventApplyAlreadyReviewed
	}

	if status == "approved" {
		s.notify(ctx, app.UserID, "success", "漫展入驻已通过",
			"您申请入驻的漫展已通过审核，现在可以在「参展摄影师」列表中被大家看到啦")
	} else {
		s.notify(ctx, app.UserID, "warning", "漫展入驻未通过",
			"很遗憾，您的漫展入驻申请未通过。原因："+strings.TrimSpace(reason))
	}
	return nil
}

func (s *EventApplyService) notify(ctx context.Context, userID int64, typ, title, content string) {
	if s.notifier == nil {
		return
	}
	if _, err := s.notifier.Create(ctx, userID, typ, title, content); err != nil {
		log.Printf("event apply notify failed: %v", err)
	}
}
