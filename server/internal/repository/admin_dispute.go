package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AdminDispute is the admin-facing row of a dispute: dispute fields plus the
// booking context (order id / both party nicknames) so the console can render
// a list without a second round-trip.
type AdminDispute struct {
	ID               int64    `json:"id"`
	BookingID        int64    `json:"bookingId"`
	OpenedBy         int64    `json:"openedBy"`
	OpenedByRole     string   `json:"openedByRole"`
	Type             string   `json:"type"`
	Reason           string   `json:"reason"`
	Status           string   `json:"status"`
	RespondDeadline  string   `json:"respondDeadline"`
	EvidenceDeadline string   `json:"evidenceDeadline"`
	Resolution       *string  `json:"resolution"`
	AdminID          *int64   `json:"adminId"`
	ReviewedAt       *string  `json:"reviewedAt"`
	CreatedAt        string   `json:"createdAt"`
	UpdatedAt        string   `json:"updatedAt"`
	CoserName        string   `json:"coserName"`
	PhotographerName string   `json:"photographerName"`
	BookingStatus    string   `json:"bookingStatus"`
	BookingSubStatus string   `json:"bookingSubStatus"`
	Evidence         []string `json:"evidence"`
}

// AdminDisputeEvent is one entry of the dispute timeline.
type AdminDisputeEvent struct {
	ID        int64   `json:"id"`
	DisputeID int64   `json:"disputeId"`
	ActorID   *int64  `json:"actorId"`
	ActorRole *string `json:"actorRole"`
	EventType string  `json:"eventType"`
	Payload   string  `json:"payload"`
	CreatedAt string  `json:"createdAt"`
}

// AdminDisputeDetail is the dispute row enriched with its timeline and the
// delivery evidence attached to the underlying booking.
type AdminDisputeDetail struct {
	AdminDispute
	Events           []AdminDisputeEvent `json:"events"`
	DeliveryEvidence []string            `json:"deliveryEvidence"`
}

// AdminDisputeRepo serves the admin dispute console queries. It is written by
// hand (not sqlc generated) because it joins bookings/users/photographers and
// extracts the evidence array out of dispute_events payloads.
type AdminDisputeRepo struct {
	pool *pgxpool.Pool
}

func NewAdminDisputeRepo(pool *pgxpool.Pool) *AdminDisputeRepo {
	return &AdminDisputeRepo{pool: pool}
}

// disputeStatusFilter builds the shared WHERE fragment for the list/count
// queries. The console filters are a superset of the raw DB statuses:
//   - ""                    : no filter
//   - open/responding/escalated/closed : exact DB status
//   - resolved              : terminal disputes that carry a resolution
//   - rejected              : closed disputes whose resolution starts with 驳回
const disputeStatusFilter = `
	WHERE (
		$1::text = ''
		OR ($1 = 'open' AND d.status = 'open')
		OR ($1 = 'responding' AND d.status = 'responding')
		OR ($1 = 'escalated' AND d.status = 'escalated')
		OR ($1 = 'closed' AND d.status = 'closed')
		OR ($1 = 'resolved' AND d.status IN ('resolved', 'closed') AND d.resolution IS NOT NULL)
		OR ($1 = 'rejected' AND d.status = 'closed' AND d.resolution LIKE '驳回%')
	)`

const adminDisputeColumns = `
	SELECT d.id, d.booking_id, d.opened_by, d.opened_by_role, d.type, d.reason, d.status,
	       d.respond_deadline::text, d.evidence_deadline::text,
	       d.resolution, d.admin_id, d.reviewed_at::text,
	       d.created_at::text, d.updated_at::text,
	       COALESCE(cu.name, ''), COALESCE(p.name, ''),
	       b.status, COALESCE(b.sub_status, ''),
	       COALESCE(ev.open_evidence, '{}')
	FROM disputes d
	JOIN bookings b ON b.id = d.booking_id
	LEFT JOIN users cu ON cu.id = b.coser_id
	LEFT JOIN photographers p ON p.id = b.photographer_id
	LEFT JOIN LATERAL (
		SELECT array_agg(elem) AS open_evidence
		FROM (
			SELECT jsonb_array_elements_text(COALESCE(de.payload->'evidence', '[]'::jsonb)) AS elem
			FROM dispute_events de
			WHERE de.dispute_id = d.id AND de.event_type = 'open'
		) t
	) ev ON TRUE`

// ListDisputes returns the admin dispute list plus the total row count.
func (r *AdminDisputeRepo) ListDisputes(ctx context.Context, status string, limit, offset int32) ([]AdminDispute, int64, error) {
	var total int64
	if err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM disputes d"+disputeStatusFilter, status).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		adminDisputeColumns+disputeStatusFilter+" ORDER BY d.updated_at DESC LIMIT $2 OFFSET $3",
		status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]AdminDispute, 0, limit)
	for rows.Next() {
		var i AdminDispute
		if err := rows.Scan(&i.ID, &i.BookingID, &i.OpenedBy, &i.OpenedByRole, &i.Type, &i.Reason, &i.Status,
			&i.RespondDeadline, &i.EvidenceDeadline, &i.Resolution, &i.AdminID, &i.ReviewedAt,
			&i.CreatedAt, &i.UpdatedAt, &i.CoserName, &i.PhotographerName,
			&i.BookingStatus, &i.BookingSubStatus, &i.Evidence); err != nil {
			return nil, 0, err
		}
		if i.Evidence == nil {
			i.Evidence = []string{}
		}
		items = append(items, i)
	}
	return items, total, rows.Err()
}

// GetDisputeDetail returns a single dispute with its timeline and the
// delivery evidence of the underlying booking.
func (r *AdminDisputeRepo) GetDisputeDetail(ctx context.Context, id int64) (AdminDisputeDetail, error) {
	var d AdminDisputeDetail
	err := r.pool.QueryRow(ctx,
		adminDisputeColumns+" WHERE d.id = $1", id).Scan(
		&d.ID, &d.BookingID, &d.OpenedBy, &d.OpenedByRole, &d.Type, &d.Reason, &d.Status,
		&d.RespondDeadline, &d.EvidenceDeadline, &d.Resolution, &d.AdminID, &d.ReviewedAt,
		&d.CreatedAt, &d.UpdatedAt, &d.CoserName, &d.PhotographerName,
		&d.BookingStatus, &d.BookingSubStatus, &d.Evidence)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminDisputeDetail{}, ErrManageNotFound
		}
		return AdminDisputeDetail{}, err
	}
	if d.Evidence == nil {
		d.Evidence = []string{}
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, dispute_id, actor_id, actor_role, event_type,
		       COALESCE(payload::text, '{}'), created_at::text
		FROM dispute_events
		WHERE dispute_id = $1
		ORDER BY created_at ASC, id ASC`, id)
	if err != nil {
		return AdminDisputeDetail{}, err
	}
	defer rows.Close()

	events := make([]AdminDisputeEvent, 0, 8)
	for rows.Next() {
		var e AdminDisputeEvent
		if err := rows.Scan(&e.ID, &e.DisputeID, &e.ActorID, &e.ActorRole, &e.EventType,
			&e.Payload, &e.CreatedAt); err != nil {
			return AdminDisputeDetail{}, err
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return AdminDisputeDetail{}, err
	}
	d.Events = events

	if err := r.pool.QueryRow(ctx,
		"SELECT COALESCE(delivery_evidence, '{}') FROM bookings WHERE id = $1",
		d.BookingID).Scan(&d.DeliveryEvidence); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return AdminDisputeDetail{}, err
	}
	if d.DeliveryEvidence == nil {
		d.DeliveryEvidence = []string{}
	}
	return d, nil
}
