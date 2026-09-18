package repository

import (
	"context"
)

type FavoriteRow struct {
	ID             int64  `json:"id"`
	PhotographerID int32  `json:"photographerId"`
	Name           string `json:"name"`
	Avatar         string `json:"avatar"`
	Location       string `json:"location"`
	Rating         string `json:"rating"`
	AddedAt        string `json:"addedAt"`
}

const insertFavorite = `-- name: InsertFavorite :exec
INSERT INTO photographer_favorites (user_id, photographer_id) VALUES ($1, $2)
ON CONFLICT (user_id, photographer_id) DO NOTHING
`

func (q *Queries) InsertFavorite(ctx context.Context, userID int64, photographerID int32) error {
	_, err := q.db.Exec(ctx, insertFavorite, userID, photographerID)
	return err
}

const deleteFavorite = `-- name: DeleteFavorite :exec
DELETE FROM photographer_favorites WHERE user_id = $1 AND photographer_id = $2
`

func (q *Queries) DeleteFavorite(ctx context.Context, userID int64, photographerID int32) error {
	_, err := q.db.Exec(ctx, deleteFavorite, userID, photographerID)
	return err
}

const listFavorites = `-- name: ListFavorites :many
SELECT f.id, p.id, p.name, p.avatar, COALESCE(p.location, ''), COALESCE(p.rating::text, '0'), f.created_at::text
FROM photographer_favorites f
JOIN photographers p ON p.id = f.photographer_id
WHERE f.user_id = $1
ORDER BY f.created_at DESC
`

func (q *Queries) ListFavorites(ctx context.Context, userID int64) ([]FavoriteRow, error) {
	rows, err := q.db.Query(ctx, listFavorites, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []FavoriteRow
	for rows.Next() {
		var i FavoriteRow
		if err := rows.Scan(&i.ID, &i.PhotographerID, &i.Name, &i.Avatar, &i.Location, &i.Rating, &i.AddedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
