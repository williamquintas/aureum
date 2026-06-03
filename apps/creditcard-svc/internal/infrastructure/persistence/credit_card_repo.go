package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/aureum/creditcard-svc/internal/domain"
)

type CreditCardRepo struct {
	pool *pgxpool.Pool
}

func NewCreditCardRepo(pool *pgxpool.Pool) *CreditCardRepo {
	return &CreditCardRepo{pool: pool}
}

func (r *CreditCardRepo) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return withTx(r.pool, ctx, fn)
}

func (r *CreditCardRepo) Save(ctx context.Context, card *domain.CreditCard) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}
	_, err := q.Exec(ctx,
		`INSERT INTO credit_cards (id, user_id, name, brand, card_type, last_four_digits, closing_day, due_day, credit_limit, available_credit, active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		card.ID, card.UserID, card.Name, string(card.Brand), string(card.CardType),
		card.LastFourDigits, card.ClosingDay, card.DueDay, card.CreditLimit,
		card.AvailableCredit, card.Active, card.CreatedAt, card.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert credit card: %w", err)
	}
	return nil
}

func (r *CreditCardRepo) FindByID(ctx context.Context, id, userID string) (*domain.CreditCard, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, brand, card_type, last_four_digits, closing_day, due_day, credit_limit, available_credit, active, created_at, updated_at, deleted_at
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
		return nil, fmt.Errorf("find credit card by id: %w", err)
	}

	card.Brand = domain.CardBrand(brand)
	card.CardType = domain.CardType(cardType)
	card.DeletedAt = deletedAt
	return &card, nil
}

func (r *CreditCardRepo) Update(ctx context.Context, card *domain.CreditCard) error {
	q := getQuerier(ctx)
	if q == nil {
		return fmt.Errorf("no transaction in context")
	}
	_, err := q.Exec(ctx,
		`UPDATE credit_cards SET name=$1, brand=$2, card_type=$3, last_four_digits=$4, closing_day=$5, due_day=$6, credit_limit=$7, available_credit=$8, active=$9, updated_at=$10
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

func (r *CreditCardRepo) FindByUser(ctx context.Context, userID string) ([]*domain.CreditCard, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, name, brand, card_type, last_four_digits, closing_day, due_day, credit_limit, available_credit, active, created_at, updated_at
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

func (r *CreditCardRepo) List(ctx context.Context, userID string, filter domain.CreditCardFilter) ([]*domain.CreditCard, error) {
	query := `SELECT id, user_id, name, brand, card_type, last_four_digits, closing_day, due_day, credit_limit, available_credit, active, created_at, updated_at
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

func (r *CreditCardRepo) Count(ctx context.Context, userID string, filter domain.CreditCardFilter) (int, error) {
	query := `SELECT COUNT(*) FROM credit_cards WHERE user_id=$1 AND deleted_at IS NULL`
	args := []interface{}{userID}
	argIdx := 2

	if filter.ActiveFilter != nil {
		query += fmt.Sprintf(" AND active=$%d", argIdx)
		args = append(args, *filter.ActiveFilter)
		argIdx++
	}

	var count int
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count credit cards: %w", err)
	}
	return count, nil
}
