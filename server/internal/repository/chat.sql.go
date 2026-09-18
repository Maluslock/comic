package repository

import (
	"context"
)

const upsertSession = `-- name: UpsertSession :one
INSERT INTO chat_sessions (user1_id, user2_id) VALUES ($1, $2)
ON CONFLICT (user1_id, user2_id) DO UPDATE SET updated_at = NOW()
RETURNING id
`

func (q *Queries) UpsertSession(ctx context.Context, user1ID, user2ID int64) (int64, error) {
	var id int64
	err := q.db.QueryRow(ctx, upsertSession, user1ID, user2ID).Scan(&id)
	return id, err
}

const getChatSessionParticipants = `-- name: GetChatSessionParticipants :one
SELECT user1_id, user2_id FROM chat_sessions WHERE id = $1
`
func (q *Queries) GetChatSessionParticipants(ctx context.Context, sessionID int64) (int64, int64, error) {
	var user1ID, user2ID int64
	err := q.db.QueryRow(ctx, getChatSessionParticipants, sessionID).Scan(&user1ID, &user2ID)
	return user1ID, user2ID, err
}

const userExists = `-- name: UserExists :one
SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)
`

func (q *Queries) UserExists(ctx context.Context, id int64) (bool, error) {
	var exists bool
	err := q.db.QueryRow(ctx, userExists, id).Scan(&exists)
	return exists, err
}

const listSessions = `-- name: ListSessions :many
SELECT s.id,
       CASE WHEN s.user1_id = $1 THEN s.user2_id ELSE s.user1_id END AS peer_id,
       COALESCE(u.name, '') AS peer_name, COALESCE(u.avatar, '') AS peer_avatar,
       COALESCE(m.content, '') AS last_message,
       COALESCE(m.created_at::text, '') AS last_time,
       (SELECT count(*) FROM chat_messages um
          WHERE um.session_id = s.id AND um.sender_id <> $1
            AND um.created_at > COALESCE(rs.last_read_at, TIMESTAMPTZ 'epoch'))::bigint AS unread_count
FROM chat_sessions s
LEFT JOIN LATERAL (
  SELECT content, created_at FROM chat_messages WHERE session_id = s.id ORDER BY created_at DESC LIMIT 1
) m ON true
LEFT JOIN users u ON u.id = CASE WHEN s.user1_id = $1 THEN s.user2_id ELSE s.user1_id END
LEFT JOIN chat_read_state rs ON rs.session_id = s.id AND rs.user_id = $1
WHERE s.user1_id = $1 OR s.user2_id = $1
ORDER BY s.updated_at DESC
`

type SessionRow struct {
	ID          int64  `json:"id"`
	PeerID      int64  `json:"peerId"`
	PeerName    string `json:"peerName"`
	PeerAvatar  string `json:"peerAvatar"`
	LastMessage string `json:"lastMessage"`
	LastTime    string `json:"lastTime"`
	UnreadCount int64  `json:"unreadCount"`
}

func (q *Queries) ListSessions(ctx context.Context, userID int64) ([]SessionRow, error) {
	rows, err := q.db.Query(ctx, listSessions, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []SessionRow
	for rows.Next() {
		var i SessionRow
		if err := rows.Scan(&i.ID, &i.PeerID, &i.PeerName, &i.PeerAvatar, &i.LastMessage, &i.LastTime, &i.UnreadCount); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const markSessionRead = `-- name: MarkSessionRead :exec
INSERT INTO chat_read_state (session_id, user_id, last_read_at) VALUES ($1, $2, NOW())
ON CONFLICT (session_id, user_id) DO UPDATE SET last_read_at = NOW()
`

func (q *Queries) MarkSessionRead(ctx context.Context, sessionID, userID int64) error {
	_, err := q.db.Exec(ctx, markSessionRead, sessionID, userID)
	return err
}

const countUnreadForUser = `-- name: CountUnreadForUser :one
SELECT count(*)::bigint FROM chat_messages m
JOIN chat_sessions s ON s.id = m.session_id
LEFT JOIN chat_read_state rs ON rs.session_id = s.id AND rs.user_id = $1
WHERE (s.user1_id = $1 OR s.user2_id = $1)
  AND m.sender_id <> $1
  AND m.created_at > COALESCE(rs.last_read_at, TIMESTAMPTZ 'epoch')
`

func (q *Queries) CountUnreadForUser(ctx context.Context, userID int64) (int64, error) {
	var n int64
	err := q.db.QueryRow(ctx, countUnreadForUser, userID).Scan(&n)
	return n, err
}

const listMessages = `-- name: ListMessages :many
SELECT id, sender_id, content, created_at::text
FROM chat_messages
WHERE session_id = $1
ORDER BY created_at ASC
`

type MessageRow struct {
	ID        int64  `json:"id"`
	SenderID  int64  `json:"senderId"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

func (q *Queries) ListMessages(ctx context.Context, sessionID int64) ([]MessageRow, error) {
	rows, err := q.db.Query(ctx, listMessages, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []MessageRow
	for rows.Next() {
		var i MessageRow
		if err := rows.Scan(&i.ID, &i.SenderID, &i.Content, &i.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const insertMessage = `-- name: InsertMessage :one
WITH ins AS (
  INSERT INTO chat_messages (session_id, sender_id, content) VALUES ($1, $2, $3)
  RETURNING id, session_id
)
UPDATE chat_sessions SET updated_at = NOW()
WHERE id = (SELECT session_id FROM ins)
RETURNING (SELECT id FROM ins) AS id
`

func (q *Queries) InsertMessage(ctx context.Context, sessionID, senderID int64, content string) (int64, error) {
	var id int64
	err := q.db.QueryRow(ctx, insertMessage, sessionID, senderID, content).Scan(&id)
	return id, err
}
