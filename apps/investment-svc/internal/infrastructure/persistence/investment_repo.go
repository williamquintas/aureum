// Package persistence provides PostgreSQL-based repository implementations.
package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/investment-svc/internal/domain"
)

// InvestmentRepo implements domain.InvestmentRepository using PostgreSQL.
type InvestmentRepo struct {
	pool *pgxpool.Pool
}

// NewInvestmentRepo creates a new InvestmentRepo.
func NewInvestmentRepo(pool *pgxpool.Pool) *InvestmentRepo {
	return &InvestmentRepo{pool: pool}
}

// WithTx executes a function within a database transaction.
func (r *InvestmentRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return withTx(ctx, r.pool, fn)
}

// Save persists a new investment entity.
func (r *InvestmentRepo) Save(ctx context.Context, investment *domain.Investment) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}

	query := `INSERT INTO investments (id, user_id, name, ticker, asset_type, quantity, ` +
		`average_price, total_invested, status, broker, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := q.Exec(ctx, query,
		investment.ID, investment.UserID, investment.Name, investment.Ticker,
		string(investment.AssetType), investment.Quantity, investment.AveragePrice,
		investment.TotalInvested, string(investment.Status), investment.Broker,
		investment.CreatedAt, investment.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert investment: %w", err)
	}
	return nil
}

// FindByID retrieves a single investment by its ID and user ID.
func (r *InvestmentRepo) FindByID(ctx context.Context, id, userID string) (*domain.Investment, error) {
	query := `SELECT id, user_id, name, ticker, asset_type, quantity, average_price, ` +
		`total_invested, status, broker, created_at, updated_at, deleted_at
		 FROM investments WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`
	row := r.pool.QueryRow(ctx, query,
		id, userID,
	)

	var inv domain.Investment
	var assetType, status string
	var deletedAt *time.Time
	err := row.Scan(
		&inv.ID, &inv.UserID, &inv.Name, &inv.Ticker,
		&assetType, &inv.Quantity, &inv.AveragePrice, &inv.TotalInvested,
		&status, &inv.Broker, &inv.CreatedAt, &inv.UpdatedAt, &deletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("find investment by id: %w", err)
	}

	inv.AssetType = domain.AssetType(assetType)
	inv.Status = domain.InvestmentStatus(status)
	inv.DeletedAt = deletedAt
	return &inv, nil
}

// Update persists changes to an existing investment entity.
func (r *InvestmentRepo) Update(ctx context.Context, investment *domain.Investment) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}

	query := `UPDATE investments SET name=$1, ticker=$2, asset_type=$3, quantity=$4, ` +
		`average_price=$5, total_invested=$6, status=$7, broker=$8, updated_at=$9
		 WHERE id=$10 AND deleted_at IS NULL`
	_, err := q.Exec(ctx, query,
		investment.Name, investment.Ticker, string(investment.AssetType),
		investment.Quantity, investment.AveragePrice, investment.TotalInvested,
		string(investment.Status), investment.Broker, investment.UpdatedAt,
		investment.ID,
	)
	if err != nil {
		return fmt.Errorf("update investment: %w", err)
	}
	return nil
}

// Delete soft-deletes an investment by ID and user ID.
func (r *InvestmentRepo) Delete(ctx context.Context, id, userID string) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}

	now := time.Now()
	_, err := q.Exec(ctx,
		`UPDATE investments SET deleted_at=$1, updated_at=$1 WHERE id=$2 AND user_id=$3 AND deleted_at IS NULL`,
		now, id, userID,
	)
	if err != nil {
		return fmt.Errorf("soft-delete investment: %w", err)
	}
	return nil
}

// List returns paginated investments for a user with optional filters.
func (r *InvestmentRepo) List(
	ctx context.Context,
	userID string,
	filter domain.InvestmentFilter,
) ([]*domain.Investment, error) {
	query := `SELECT id, user_id, name, ticker, asset_type, quantity, average_price, ` +
		`total_invested, status, broker, created_at, updated_at
			  FROM investments WHERE user_id=$1 AND deleted_at IS NULL`
	args := []interface{}{userID}
	argIdx := 2

	if filter.TypeFilter != nil {
		query += fmt.Sprintf(" AND asset_type=$%d", argIdx)
		args = append(args, string(*filter.TypeFilter))
		argIdx++
	}
	if filter.StatusFilter != nil {
		query += fmt.Sprintf(" AND status=$%d", argIdx)
		args = append(args, string(*filter.StatusFilter))
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filter.Limit)
		argIdx++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filter.Offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list investments: %w", err)
	}
	defer rows.Close()

	var investments []*domain.Investment
	for rows.Next() {
		var inv domain.Investment
		var assetType, status string
		err := rows.Scan(
			&inv.ID, &inv.UserID, &inv.Name, &inv.Ticker,
			&assetType, &inv.Quantity, &inv.AveragePrice, &inv.TotalInvested,
			&status, &inv.Broker, &inv.CreatedAt, &inv.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan investment: %w", err)
		}
		inv.AssetType = domain.AssetType(assetType)
		inv.Status = domain.InvestmentStatus(status)
		investments = append(investments, &inv)
	}

	return investments, nil
}

// Count returns the total number of investments matching the given filter.
func (r *InvestmentRepo) Count(ctx context.Context, userID string, filter domain.InvestmentFilter) (int, error) {
	query := `SELECT COUNT(*) FROM investments WHERE user_id=$1 AND deleted_at IS NULL`
	args := []interface{}{userID}
	argIdx := 2

	if filter.TypeFilter != nil {
		query += fmt.Sprintf(" AND asset_type=$%d", argIdx)
		args = append(args, string(*filter.TypeFilter))
		argIdx++
	}
	if filter.StatusFilter != nil {
		query += fmt.Sprintf(" AND status=$%d", argIdx)
		args = append(args, string(*filter.StatusFilter))
	}

	var count int
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count investments: %w", err)
	}
	return count, nil
}

// FindByUser retrieves all non-deleted investments for a user.
func (r *InvestmentRepo) FindByUser(ctx context.Context, userID string) ([]*domain.Investment, error) {
	query := `SELECT id, user_id, name, ticker, asset_type, quantity, average_price, ` +
		`total_invested, status, broker, created_at, updated_at
		 FROM investments WHERE user_id=$1 AND deleted_at IS NULL ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("find by user: %w", err)
	}
	defer rows.Close()

	var investments []*domain.Investment
	for rows.Next() {
		var inv domain.Investment
		var assetType, status string
		err := rows.Scan(
			&inv.ID, &inv.UserID, &inv.Name, &inv.Ticker,
			&assetType, &inv.Quantity, &inv.AveragePrice, &inv.TotalInvested,
			&status, &inv.Broker, &inv.CreatedAt, &inv.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan investment: %w", err)
		}
		inv.AssetType = domain.AssetType(assetType)
		inv.Status = domain.InvestmentStatus(status)
		investments = append(investments, &inv)
	}

	return investments, nil
}

// FindActiveByUser retrieves all active investments for a user.
func (r *InvestmentRepo) FindActiveByUser(ctx context.Context, userID string) ([]*domain.Investment, error) {
	query := `SELECT id, user_id, name, ticker, asset_type, quantity, average_price, ` +
		`total_invested, status, broker, created_at, updated_at
		 FROM investments WHERE user_id=$1 AND status='active' AND deleted_at IS NULL ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("find active by user: %w", err)
	}
	defer rows.Close()

	var investments []*domain.Investment
	for rows.Next() {
		var inv domain.Investment
		var assetType, status string
		err := rows.Scan(
			&inv.ID, &inv.UserID, &inv.Name, &inv.Ticker,
			&assetType, &inv.Quantity, &inv.AveragePrice, &inv.TotalInvested,
			&status, &inv.Broker, &inv.CreatedAt, &inv.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan investment: %w", err)
		}
		inv.AssetType = domain.AssetType(assetType)
		inv.Status = domain.InvestmentStatus(status)
		investments = append(investments, &inv)
	}

	return investments, nil
}
