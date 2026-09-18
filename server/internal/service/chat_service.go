package service

import (
	"context"

	"github.com/Maluslock/comic/server/internal/repository"
)

type SessionItem struct {
	ID          int64  `json:"id"`
	PeerID      int64  `json:"peerId"`
	PeerName    string `json:"peerName"`
	PeerAvatar  string `json:"peerAvatar"`
	LastMessage string `json:"lastMessage"`
	LastTime    string `json:"lastTime"`
	UnreadCount int64  `json:"unreadCount"`
}

type MessageItem struct {
	ID        int64  `json:"id"`
	SenderID  int64  `json:"senderId"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

type ChatService struct {
	queries *repository.Queries
}

func NewChatService(queries *repository.Queries) *ChatService {
	return &ChatService{queries: queries}
}

func minMax(a, b int64) (int64, int64) {
	if a < b {
		return a, b
	}
	return b, a
}

func (s *ChatService) GetOrCreateSession(ctx context.Context, userID, otherUserID int64) (int64, error) {
	blocked, err := s.queries.IsBlockedPair(ctx, userID, otherUserID)
	if err != nil {
		return 0, err
	}
	if blocked {
		return 0, ErrForbidden
	}
	exists, err := s.queries.UserExists(ctx, otherUserID)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, ErrUserNotFound
	}
	u1, u2 := minMax(userID, otherUserID)
	return s.queries.UpsertSession(ctx, u1, u2)
}

func (s *ChatService) ListSessions(ctx context.Context, userID int64) ([]SessionItem, error) {
	rows, err := s.queries.ListSessions(ctx, userID)
	if err != nil {
		return nil, err
	}
	items := make([]SessionItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, SessionItem{
			ID:          r.ID,
			PeerID:      r.PeerID,
			PeerName:    r.PeerName,
			PeerAvatar:  r.PeerAvatar,
			LastMessage: r.LastMessage,
			LastTime:    r.LastTime,
			UnreadCount: r.UnreadCount,
		})
	}
	return items, nil
}

func (s *ChatService) ListMessagesForUser(ctx context.Context, sessionID, userID int64) ([]MessageItem, error) {
	if err := s.assertParticipant(ctx, sessionID, userID); err != nil {
		return nil, err
	}
	return s.ListMessages(ctx, sessionID)
}

func (s *ChatService) SendMessageForUser(ctx context.Context, sessionID, senderID int64, content string) (int64, error) {
	if err := s.assertParticipant(ctx, sessionID, senderID); err != nil {
		return 0, err
	}
	return s.SendMessage(ctx, sessionID, senderID, content)
}

func (s *ChatService) assertParticipant(ctx context.Context, sessionID, userID int64) error {
	user1ID, user2ID, err := s.queries.GetChatSessionParticipants(ctx, sessionID)
	if err != nil {
		return ErrSessionNotFound
	}
	if userID != user1ID && userID != user2ID {
		return ErrForbidden
	}
	return nil
}

func (s *ChatService) ListMessages(ctx context.Context, sessionID int64) ([]MessageItem, error) {
	rows, err := s.queries.ListMessages(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	items := make([]MessageItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, MessageItem{
			ID:        r.ID,
			SenderID:  r.SenderID,
			Content:   r.Content,
			CreatedAt: r.CreatedAt,
		})
	}
	return items, nil
}

func (s *ChatService) SendMessage(ctx context.Context, sessionID, senderID int64, content string) (int64, error) {
	id, err := s.queries.InsertMessage(ctx, sessionID, senderID, content)
	if err != nil {
		if isForeignKeyViolation(err) {
			return 0, ErrSessionNotFound
		}
		return 0, err
	}
	return id, nil
}

func (s *ChatService) MarkSessionRead(ctx context.Context, sessionID, userID int64) error {
	return s.queries.MarkSessionRead(ctx, sessionID, userID)
}

func (s *ChatService) TotalUnread(ctx context.Context, userID int64) (int64, error) {
	return s.queries.CountUnreadForUser(ctx, userID)
}
