package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
)

const insertNotification = `-- name: InsertNotification :one
INSERT INTO notifications (user_id, type, title, content, link_type, link_id) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id
`

func (q *Queries) InsertNotification(ctx context.Context, userID int64, typ, title, content string, linkType *string, linkID *int64) (int64, error) {
	var id int64
	err := q.db.QueryRow(ctx, insertNotification, userID, typ, title, content, linkType, linkID).Scan(&id)
	return id, err
}

const insertBroadcast = `-- name: InsertBroadcast :one
INSERT INTO notifications (type, title, content) VALUES ($1, $2, $3)
RETURNING id
`

func (q *Queries) InsertBroadcast(ctx context.Context, typ, title, content string) (int64, error) {
	var id int64
	err := q.db.QueryRow(ctx, insertBroadcast, typ, title, content).Scan(&id)
	return id, err
}

const listNotifications = `-- name: ListNotifications :many
SELECT id, user_id, type, title, content, read, created_at::text, link_type, link_id
FROM notifications
WHERE (user_id = $1 OR user_id IS NULL)
ORDER BY created_at DESC
LIMIT 20
`

type NotificationRow struct {
	ID        int64   `json:"id"`
	UserID    *int64  `json:"userId"`
	Type      string  `json:"type"`
	Title     string  `json:"title"`
	Content   string  `json:"content"`
	Read      bool    `json:"read"`
	CreatedAt string  `json:"createdAt"`
	LinkType  *string `json:"linkType"`
	LinkID    *int64  `json:"linkId"`
}

func (q *Queries) ListNotifications(ctx context.Context, userID int64) ([]NotificationRow, error) {
	rows, err := q.db.Query(ctx, listNotifications, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []NotificationRow
	for rows.Next() {
		var i NotificationRow
		if err := rows.Scan(&i.ID, &i.UserID, &i.Type, &i.Title, &i.Content, &i.Read, &i.CreatedAt, &i.LinkType, &i.LinkID); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

const listAllNotifications = `-- name: ListAllNotifications :many
SELECT id, user_id, type, title, content, read, created_at::text, link_type, link_id
FROM notifications
ORDER BY created_at DESC
LIMIT $1 OFFSET $2
`

const countNotifications = `-- name: CountNotifications :one
SELECT count(*) FROM notifications
`

func (q *Queries) ListAllNotifications(ctx context.Context, limit, offset int) ([]NotificationRow, int64, error) {
	rows, err := q.db.Query(ctx, listAllNotifications, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []NotificationRow
	for rows.Next() {
		var i NotificationRow
		if err := rows.Scan(&i.ID, &i.UserID, &i.Type, &i.Title, &i.Content, &i.Read, &i.CreatedAt, &i.LinkType, &i.LinkID); err != nil {
			return nil, 0, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int64
	if err := q.db.QueryRow(ctx, countNotifications).Scan(&total); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

const markNotificationRead = `-- name: MarkNotificationRead :exec
UPDATE notifications SET read = true WHERE id = $1 AND user_id = $2
`

func (q *Queries) MarkNotificationRead(ctx context.Context, id, userID int64) error {
	tag, err := q.db.Exec(ctx, markNotificationRead, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

const deleteNotification = `-- name: DeleteNotification :exec
DELETE FROM notifications WHERE id = $1
`

func (q *Queries) DeleteNotification(ctx context.Context, id int64) error {
	tag, err := q.db.Exec(ctx, deleteNotification, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
