package service

import (
	"context"
	"errors"

	"github.com/Maluslock/comic/server/internal/repository"
)

// ErrInvalidStyleSelection 表示风格选择不合法：多于 5 个、tagId 不存在。
// handler 把它映射成 400 {"error":"invalid style selection"}。
var ErrInvalidStyleSelection = errors.New("invalid style selection")

// maxStylePrefs 是风格偏好的数量上限（产品约束：最多勾 5 个风格标签）。
const maxStylePrefs = 5

// StyleTagItem 是 GET /me/style-prefs 响应里的一行标签。
type StyleTagItem struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// StylePrefsResponse 是 GET/PUT /api/v1/me/style-prefs 的响应体。
// tags 为 null 会让前端 v-for 抛错，因此恒为空切片兜底。
// chosen 只认 users.style_pref_chosen 列：tags 为空可能是没选过（chosen=false），
// 也可能是主动清空（chosen=true），两者对首启弹窗的语义完全不同，不许推断。
type StylePrefsResponse struct {
	Tags   []StyleTagItem `json:"tags"`
	Chosen bool           `json:"chosen"`
}

// stylePrefsStore 是风格偏好需要的数据访问面，由 *repository.Queries 实现，
// 单测用 fake 注入（同 city_service.go 的 cityStore 写法）。
// 注意：chosen 的置位由 ReplaceStylePrefs 在**同一事务内**完成，
// 所以这里不再单独暴露 MarkStylePrefChosen —— 分开调用会留下
// 「偏好写成功但 chosen 没置上，下次进首页又被弹一次」的半成功态。
type stylePrefsStore interface {
	ListStyleTagsByUser(ctx context.Context, userID int64) ([]repository.StyleTagRow, error)
	ReplaceStylePrefs(ctx context.Context, userID int64, tagIDs []int64) error
	GetStylePrefChosen(ctx context.Context, userID int64) (bool, error)
	CountTags(ctx context.Context, tagIDs []int64) (int, error)
}

// StylePrefService 处理 C 端首启风格偏好的读写。
type StylePrefService struct {
	store stylePrefsStore
}

func NewStylePrefService(store stylePrefsStore) *StylePrefService {
	return &StylePrefService{store: store}
}

// Get 返回用户当前的风格偏好与「是否明确做过选择」。
func (s *StylePrefService) Get(ctx context.Context, userID int64) (StylePrefsResponse, error) {
	chosen, err := s.store.GetStylePrefChosen(ctx, userID)
	if err != nil {
		return StylePrefsResponse{}, err
	}
	rows, err := s.store.ListStyleTagsByUser(ctx, userID)
	if err != nil {
		return StylePrefsResponse{}, err
	}
	tags := make([]StyleTagItem, 0, len(rows))
	for _, r := range rows {
		tags = append(tags, StyleTagItem{ID: r.ID, Name: r.Name})
	}
	return StylePrefsResponse{Tags: tags, Chosen: chosen}, nil
}

// Replace 全量替换用户的风格偏好（空数组 = 主动清空，允许且同样算「做过选择」）。
//
// 校验顺序：先去重（重复勾选不算超限），再判上限 1..5，最后验 tagId 全部存在。
// PUT 成功无论是否为空，都把 users.style_pref_chosen 置 true。
func (s *StylePrefService) Replace(ctx context.Context, userID int64, tagIDs []int64) (StylePrefsResponse, error) {
	// 去重：保序去重，重复勾选是前端多选框的正常抖动，不该报 400。
	seen := make(map[int64]struct{}, len(tagIDs))
	uniq := make([]int64, 0, len(tagIDs))
	for _, id := range tagIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}

	if len(uniq) > maxStylePrefs {
		return StylePrefsResponse{}, ErrInvalidStyleSelection
	}

	// tagId 必须全部存在于 tags 表：选了一个不存在的风格 = 校验失败。
	// 空数组跳过（COUNT(*) 不用查，且 ANY('') 的绑定语义容易踩坑）。
	if len(uniq) > 0 {
		n, err := s.store.CountTags(ctx, uniq)
		if err != nil {
			return StylePrefsResponse{}, err
		}
		if n != len(uniq) {
			return StylePrefsResponse{}, ErrInvalidStyleSelection
		}
	}

	// 偏好替换与 chosen 置位由同一个事务保证原子性（见 repository.ReplaceStylePrefs）。
	if err := s.store.ReplaceStylePrefs(ctx, userID, uniq); err != nil {
		return StylePrefsResponse{}, err
	}
	return s.Get(ctx, userID)
}
