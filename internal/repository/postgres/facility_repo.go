// Package postgres provides PostgreSQL-backed implementations of the domain
// repository ports. It is intentionally minimal: the in-memory implementation
// is the default for offline runs and tests. The postgres implementations are
// available for production deployments; they require a live PostgreSQL
// instance configured via DATABASE_URL.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cry047/baseline/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool is a thin alias for the platform pool.
type Pool = pgxpool.Pool

// ErrRowNotFound is returned when a query returns no rows.
var ErrRowNotFound = pgx.ErrNoRows

// ErrDuplicate is returned when a unique constraint is violated.
var ErrDuplicate = func(e *pgconn.PgError) bool {
	return e.Code == "23505"
}

// wrapPgErr converts a pgx error to a domain.Error with the appropriate code.
func wrapPgErr(err error, entity, id string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound(entity, id)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return domain.ErrConflict("唯一约束冲突: "+entity, err)
		}
	}
	return domain.WrapErr(domain.CodeInternal, "数据库错误", "", err)
}

// FacilityRepository is the PostgreSQL-backed implementation.
type FacilityRepository struct {
	pool *Pool
	now  func() time.Time
}

// NewFacilityRepository returns a FacilityRepository backed by a pgx pool.
func NewFacilityRepository(pool *Pool) *FacilityRepository {
	return &FacilityRepository{pool: pool, now: time.Now}
}

// Create inserts a facility.
func (r *FacilityRepository) Create(ctx context.Context, f *domain.Facility) error {
	if f.CreatedAt.IsZero() {
		f.CreatedAt = r.now()
	}
	if f.UpdatedAt.IsZero() {
		f.UpdatedAt = f.CreatedAt
	}
	if f.Version == 0 {
		f.Version = 1
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO facilities (id, place_id, name, code, category, responsible_person_id, criticality, status, description, alternative_facility_id, created_at, updated_at, version)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		f.ID, f.PlaceID, f.Name, f.Code, f.Category, f.ResponsiblePersonID, f.Criticality, f.Status, f.Description, f.AlternativeFacilityID, f.CreatedAt, f.UpdatedAt, f.Version)
	return wrapPgErr(err, "Facility", f.ID)
}

// Update modifies a facility using optimistic concurrency control.
func (r *FacilityRepository) Update(ctx context.Context, f *domain.Facility) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE facilities SET place_id=$2, name=$3, code=$4, category=$5, responsible_person_id=$6, criticality=$7, status=$8, description=$9, alternative_facility_id=$10, updated_at=$11, version=$12
		 WHERE id=$1 AND version=$11 - 1`,
		f.ID, f.PlaceID, f.Name, f.Code, f.Category, f.ResponsiblePersonID, f.Criticality, f.Status, f.Description, f.AlternativeFacilityID, r.now(), f.Version)
	if err != nil {
		return wrapPgErr(err, "Facility", f.ID)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrConflict("版本冲突: 设施 "+f.ID, nil)
	}
	return nil
}

// UpdateStatus updates only the status column with optimistic concurrency.
func (r *FacilityRepository) UpdateStatus(ctx context.Context, id string, status domain.FacilityStatus, version int) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE facilities SET status=$2, updated_at=$3, version=version+1 WHERE id=$1 AND version=$4`,
		id, status, r.now(), version)
	if err != nil {
		return wrapPgErr(err, "Facility", id)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrConflict("版本冲突: 设施 "+id, nil)
	}
	return nil
}

// Delete removes a facility.
func (r *FacilityRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM facilities WHERE id=$1`, id)
	if err != nil {
		return wrapPgErr(err, "Facility", id)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound("Facility", id)
	}
	return nil
}

// Get retrieves a facility by id.
func (r *FacilityRepository) Get(ctx context.Context, id string) (*domain.Facility, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, place_id, name, code, category, responsible_person_id, criticality, status, description, alternative_facility_id, created_at, updated_at, version
		 FROM facilities WHERE id=$1`, id)
	f := &domain.Facility{}
	err := row.Scan(&f.ID, &f.PlaceID, &f.Name, &f.Code, &f.Category, &f.ResponsiblePersonID, &f.Criticality, &f.Status, &f.Description, &f.AlternativeFacilityID, &f.CreatedAt, &f.UpdatedAt, &f.Version)
	if err != nil {
		return nil, wrapPgErr(err, "Facility", id)
	}
	return f, nil
}

// List returns a paginated list of facilities.
func (r *FacilityRepository) List(ctx context.Context, q domain.PageQuery) (*domain.PageResult[*domain.Facility], error) {
	q.Normalize()
	// Whitelist order_by.
	allowed := map[string]string{
		"name": "name", "code": "code", "criticality": "criticality", "status": "status", "created_at": "created_at",
	}
	col := allowed[q.OrderBy]
	if col == "" {
		col = "created_at"
	}
	dir := "ASC"
	if q.Order == "desc" {
		dir = "DESC"
	}
	// Build a WHERE clause from whitelisted filters.
	where := "1=1"
	args := []any{}
	if v, ok := q.Filters["place_id"]; ok && v != "" {
		args = append(args, v)
		where += fmt.Sprintf(" AND place_id=$%d", len(args))
	}
	if v, ok := q.Filters["criticality"]; ok && v != "" {
		args = append(args, v)
		where += fmt.Sprintf(" AND criticality=$%d", len(args))
	}
	if v, ok := q.Filters["status"]; ok && v != "" {
		args = append(args, v)
		where += fmt.Sprintf(" AND status=$%d", len(args))
	}
	if v, ok := q.Filters["responsible_person_id"]; ok && v != "" {
		args = append(args, v)
		where += fmt.Sprintf(" AND responsible_person_id=$%d", len(args))
	}
	if v, ok := q.Filters["name"]; ok && v != "" {
		args = append(args, "%"+v+"%")
		where += fmt.Sprintf(" AND name ILIKE $%d", len(args))
	}
	args = append(args, q.Limit, q.Offset)
	rows, err := r.pool.Query(ctx,
		fmt.Sprintf(`SELECT id, place_id, name, code, category, responsible_person_id, criticality, status, description, alternative_facility_id, created_at, updated_at, version
		             FROM facilities WHERE %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
			where, col, dir, len(args)-1, len(args)),
		args...)
	if err != nil {
		return nil, wrapPgErr(err, "Facility", "")
	}
	defer rows.Close()
	items := []*domain.Facility{}
	for rows.Next() {
		f := &domain.Facility{}
		if err := rows.Scan(&f.ID, &f.PlaceID, &f.Name, &f.Code, &f.Category, &f.ResponsiblePersonID, &f.Criticality, &f.Status, &f.Description, &f.AlternativeFacilityID, &f.CreatedAt, &f.UpdatedAt, &f.Version); err != nil {
			return nil, wrapPgErr(err, "Facility", "")
		}
		items = append(items, f)
	}
	// Count total using the same WHERE clause.
	var total int64
	if err := r.pool.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM facilities WHERE %s`, where)).Scan(&total); err != nil {
		return nil, wrapPgErr(err, "Facility", "")
	}
	return &domain.PageResult[*domain.Facility]{
		Items: items,
		Total: total,
		Limit: q.Limit,
		Page:  q.Offset/q.Limit + 1,
	}, nil
}

// ListByPlace returns facilities under a place.
func (r *FacilityRepository) ListByPlace(ctx context.Context, placeID string) ([]*domain.Facility, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, place_id, name, code, category, responsible_person_id, criticality, status, description, alternative_facility_id, created_at, updated_at, version
		 FROM facilities WHERE place_id=$1 ORDER BY code ASC`, placeID)
	if err != nil {
		return nil, wrapPgErr(err, "Facility", "")
	}
	defer rows.Close()
	out := []*domain.Facility{}
	for rows.Next() {
		f := &domain.Facility{}
		if err := rows.Scan(&f.ID, &f.PlaceID, &f.Name, &f.Code, &f.Category, &f.ResponsiblePersonID, &f.Criticality, &f.Status, &f.Description, &f.AlternativeFacilityID, &f.CreatedAt, &f.UpdatedAt, &f.Version); err != nil {
			return nil, wrapPgErr(err, "Facility", "")
		}
		out = append(out, f)
	}
	return out, nil
}

// ListByResponsiblePerson returns facilities for a responsible person.
func (r *FacilityRepository) ListByResponsiblePerson(ctx context.Context, personID string) ([]*domain.Facility, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, place_id, name, code, category, responsible_person_id, criticality, status, description, alternative_facility_id, created_at, updated_at, version
		 FROM facilities WHERE responsible_person_id=$1 ORDER BY code ASC`, personID)
	if err != nil {
		return nil, wrapPgErr(err, "Facility", "")
	}
	defer rows.Close()
	out := []*domain.Facility{}
	for rows.Next() {
		f := &domain.Facility{}
		if err := rows.Scan(&f.ID, &f.PlaceID, &f.Name, &f.Code, &f.Category, &f.ResponsiblePersonID, &f.Criticality, &f.Status, &f.Description, &f.AlternativeFacilityID, &f.CreatedAt, &f.UpdatedAt, &f.Version); err != nil {
			return nil, wrapPgErr(err, "Facility", "")
		}
		out = append(out, f)
	}
	return out, nil
}
