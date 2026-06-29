// Package persistence provides PostgreSQL-based repository implementations for the credit card service.
package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/creditcard-svc/internal/domain"
)

// CreditCardRepo implements domain.CreditCardRepository using PostgreSQL (pgx).
type CreditCardRepo struct {
	pool *pgxpool.Pool
}

// NewCreditCardRepo creates a new CreditCardRepo.
func NewCreditCardRepo(pool *pgxpool.Pool) *CreditCardRepo {
	return &CreditCardRepo{pool: pool}
}

// WithTx executes a function within a database transaction.
func (r *CreditCardRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return withTx(ctx, r.pool, fn)
}

// Save inserts a new credit card record.
func (r *CreditCardRepo) Save(ctx context.Context, card *domain.CreditCard) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}
	_, err := q.Exec(ctx,
		`INSERT INTO credit_cards (id, user_id, name, brand, card_type, `+
			`last_four_digits, closing_day, due_day, credit_limit, `+
			`available_credit, active, created_at, updated_at) `+
			`VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		card.ID, card.UserID, card.Name, string(card.Brand), string(card.CardType),
		card.LastFourDigits, card.ClosingDay, card.DueDay, card.CreditLimit,
		card.AvailableCredit, card.Active, card.CreatedAt, card.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert credit card: %w", err)
	}
	return nil
}

// FindByID retrieves a credit card by ID and user ID, excluding soft-deleted records.
func (r *CreditCardRepo) FindByID(ctx context.Context, id, userID string) (*domain.CreditCard, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, brand, card_type, last_four_digits, `+
			`closing_day, due_day, credit_limit, available_credit, active, `+
			`created_at, updated_at, deleted_at
		 FROM credit_cards WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`,
		id, userID,
	)

	var card domain.CreditCard
	var brand, cardType string
	var deletedAt *time.Time
	err := row.Scan(
		&card.ID, &card.UserID, &card.Name, &brand, &cardType,
		&card.LastFourDigits, &card.ClosingDay, &card.DueDay,
		&card.CreditLimit, &card.AvailableCredit, &card.Active,
		&card.CreatedAt, &card.UpdatedAt, &deletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("find credit card by id: %w", err)
	}

	card.Brand = domain.CardBrand(brand)
	card.CardType = domain.CardType(cardType)
	card.DeletedAt = deletedAt
	return &card, nil
}

// Update applies changes to an existing credit card.
func (r *CreditCardRepo) Update(ctx context.Context, card *domain.CreditCard) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}
	_, err := q.Exec(ctx,
		`UPDATE credit_cards SET name=$1, brand=$2, card_type=$3, `+
			`last_four_digits=$4, closing_day=$5, due_day=$6, `+
			`credit_limit=$7, available_credit=$8, active=$9, updated_at=$10
		 WHERE id=$11 AND deleted_at IS NULL`,
		card.Name, string(card.Brand), string(card.CardType), card.LastFourDigits,
		card.ClosingDay, card.DueDay, card.CreditLimit, card.AvailableCredit,
		card.Active, card.UpdatedAt, card.ID,
	)
	if err != nil {
		return fmt.Errorf("update credit card: %w", err)
	}
	return nil
}

// Delete performs a soft-delete on a credit card.
func (r *CreditCardRepo) Delete(ctx context.Context, id, userID string) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}
	_, err := q.Exec(ctx,
		`UPDATE credit_cards SET deleted_at=$1, updated_at=$1 WHERE id=$2 AND user_id=$3 AND deleted_at IS NULL`,
		time.Now(), id, userID,
	)
	if err != nil {
		return fmt.Errorf("soft-delete credit card: %w", err)
	}
	return nil
}

// FindByUser retrieves all active credit cards belonging to a user.
func (r *CreditCardRepo) FindByUser(ctx context.Context, userID string) ([]*domain.CreditCard, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, name, brand, card_type, last_four_digits, `+
			`closing_day, due_day, credit_limit, available_credit, active, `+
			`created_at, updated_at
		 FROM credit_cards WHERE user_id=$1 AND deleted_at IS NULL ORDER BY name ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("find cards by user: %w", err)
	}
	defer rows.Close()

	var cards []*domain.CreditCard
	for rows.Next() {
		var card domain.CreditCard
		var brand, cardType string
		err := rows.Scan(
			&card.ID, &card.UserID, &card.Name, &brand, &cardType,
			&card.LastFourDigits, &card.ClosingDay, &card.DueDay,
			&card.CreditLimit, &card.AvailableCredit, &card.Active,
			&card.CreatedAt, &card.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan credit card: %w", err)
		}
		card.Brand = domain.CardBrand(brand)
		card.CardType = domain.CardType(cardType)
		cards = append(cards, &card)
	}
	return cards, nil
}

// List returns credit cards filtered by user ID with optional filters, ordered by name ASC.
func (r *CreditCardRepo) List(
	ctx context.Context, userID string, filter domain.CreditCardFilter,
) ([]*domain.CreditCard, error) {
	query := `SELECT id, user_id, name, brand, card_type, last_four_digits, ` +
		`closing_day, due_day, credit_limit, available_credit, active, created_at, updated_at
			  FROM credit_cards WHERE user_id=$1 AND deleted_at IS NULL`
	args := []interface{}{userID}
	argIdx := 2

	if filter.ActiveFilter != nil {
		query += fmt.Sprintf(" AND active=$%d", argIdx)
		args = append(args, *filter.ActiveFilter)
		argIdx++
	}

	query += " ORDER BY name ASC"

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
		return nil, fmt.Errorf("list credit cards: %w", err)
	}
	defer rows.Close()

	var cards []*domain.CreditCard
	for rows.Next() {
		var card domain.CreditCard
		var brand, cardType string
		err := rows.Scan(
			&card.ID, &card.UserID, &card.Name, &brand, &cardType,
			&card.LastFourDigits, &card.ClosingDay, &card.DueDay,
			&card.CreditLimit, &card.AvailableCredit, &card.Active,
			&card.CreatedAt, &card.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan credit card: %w", err)
		}
		card.Brand = domain.CardBrand(brand)
		card.CardType = domain.CardType(cardType)
		cards = append(cards, &card)
	}
	return cards, nil
}

// Count returns the total number of credit cards matching the filter.
func (r *CreditCardRepo) Count(ctx context.Context, userID string, filter domain.CreditCardFilter) (int, error) {
	query := `SELECT COUNT(*) FROM credit_cards WHERE user_id=$1 AND deleted_at IS NULL`
	args := []interface{}{userID}
	argIdx := 2

	if filter.ActiveFilter != nil {
		query += fmt.Sprintf(" AND active=$%d", argIdx)
		args = append(args, *filter.ActiveFilter)
	}

	var count int
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count credit cards: %w", err)
	}
	return count, nil
}
