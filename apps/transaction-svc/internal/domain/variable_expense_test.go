//nolint:goconst // test file - repeated strings acceptable
package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func validCreateVariableExpenseInput() CreateVariableExpenseInput {
	return CreateVariableExpenseInput{
		UserID:        "user-1",
		Description:   "Dinner out",
		Destination:   "Restaurant X",
		Category:      "Food",
		ExpenseType:   ExpenseTypeDiscretionary,
		PaymentMethod: PaymentMethodDebitCard,
		PaymentDate:   "2026-05-15",
		PaidAmount:    15000,
		Status:        StatusPending,
	}
}

func TestNewVariableExpense_Success(t *testing.T) {
	input := validCreateVariableExpenseInput()
	ve, err := NewVariableExpense(input)
	require.NoError(t, err)
	require.NotNil(t, ve)
	require.Equal(t, input.UserID, ve.UserID)
	require.Equal(t, input.Description, ve.Description)
	require.Equal(t, input.Destination, ve.Destination)
	require.Equal(t, input.Category, ve.Category)
	require.Equal(t, input.ExpenseType, ve.ExpenseType)
	require.Equal(t, input.PaymentMethod, ve.PaymentMethod)
	require.Equal(t, input.PaymentDate, ve.PaymentDate)
	require.Equal(t, input.PaidAmount, ve.PaidAmount)
	require.Equal(t, input.Status, ve.Status)
	require.False(t, ve.CreatedAt.IsZero())
}

func TestNewVariableExpense_ValidationErrors(t *testing.T) {
	tests := []struct {
		name  string
		tweak func(input *CreateVariableExpenseInput)
		err   error
	}{
		{
			name:  "empty user_id",
			tweak: func(input *CreateVariableExpenseInput) { input.UserID = "" },
			err:   ErrMissingField,
		},
		{
			name:  "empty description",
			tweak: func(input *CreateVariableExpenseInput) { input.Description = "" },
			err:   ErrMissingField,
		},
		{
			name:  "empty destination",
			tweak: func(input *CreateVariableExpenseInput) { input.Destination = "" },
			err:   ErrMissingField,
		},
		{
			name:  "empty category",
			tweak: func(input *CreateVariableExpenseInput) { input.Category = "" },
			err:   ErrMissingField,
		},
		{
			name:  "empty expense_type",
			tweak: func(input *CreateVariableExpenseInput) { input.ExpenseType = "" },
			err:   ErrMissingField,
		},
		{
			name:  "invalid expense_type",
			tweak: func(input *CreateVariableExpenseInput) { input.ExpenseType = "luxury" },
			err:   ErrInvalidEnum,
		},
		{
			name:  "empty payment method",
			tweak: func(input *CreateVariableExpenseInput) { input.PaymentMethod = "" },
			err:   ErrMissingField,
		},
		{
			name:  "invalid payment method",
			tweak: func(input *CreateVariableExpenseInput) { input.PaymentMethod = "eth" },
			err:   ErrInvalidEnum,
		},
		{
			name:  "empty payment_date",
			tweak: func(input *CreateVariableExpenseInput) { input.PaymentDate = "" },
			err:   ErrMissingField,
		},
		{
			name:  "zero paid_amount",
			tweak: func(input *CreateVariableExpenseInput) { input.PaidAmount = 0 },
			err:   ErrNegativeAmount,
		},
		{
			name:  "negative paid_amount",
			tweak: func(input *CreateVariableExpenseInput) { input.PaidAmount = -1 },
			err:   ErrNegativeAmount,
		},
		{
			name:  "empty status",
			tweak: func(input *CreateVariableExpenseInput) { input.Status = "" },
			err:   ErrMissingField,
		},
		{
			name:  "invalid status",
			tweak: func(input *CreateVariableExpenseInput) { input.Status = "archived" },
			err:   ErrInvalidStatus,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validCreateVariableExpenseInput()
			tt.tweak(&input)
			_, err := NewVariableExpense(input)
			require.ErrorIs(t, err, tt.err)
		})
	}
}

func TestVariableExpense_TransitionStatus(t *testing.T) {
	tests := []struct {
		name    string
		from    TransactionStatus
		to      TransactionStatus
		wantErr error
	}{
		{
			name: "pending to completed",
			from: StatusPending,
			to:   StatusCompleted,
		},
		{
			name: "pending to cancelled",
			from: StatusPending,
			to:   StatusCancelled,
		},
		{
			name:    "completed to pending",
			from:    StatusCompleted,
			to:      StatusPending,
			wantErr: ErrStatusTransition,
		},
		{
			name:    "cancelled to completed",
			from:    StatusCancelled,
			to:      StatusCompleted,
			wantErr: ErrStatusTransition,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validCreateVariableExpenseInput()
			input.Status = tt.from
			ve, err := NewVariableExpense(input)
			require.NoError(t, err)

			err = ve.TransitionStatus(tt.to)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, tt.from, ve.Status)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.to, ve.Status)
			}
		})
	}
}

func TestVariableExpense_ApplyUpdate_Success(t *testing.T) {
	ve, err := NewVariableExpense(validCreateVariableExpenseInput())
	require.NoError(t, err)

	newDesc := "Lunch meeting"
	newAmount := int64(20000)
	completed := StatusCompleted

	err = ve.ApplyUpdate(UpdateVariableExpenseInput{
		UserID:      ve.UserID,
		Description: &newDesc,
		PaidAmount:  &newAmount,
		Status:      &completed,
	})

	require.NoError(t, err)
	require.Equal(t, newDesc, ve.Description)
	require.Equal(t, newAmount, ve.PaidAmount)
	require.Equal(t, StatusCompleted, ve.Status)
}

func TestVariableExpense_ApplyUpdate_AccessDenied(t *testing.T) {
	ve, err := NewVariableExpense(validCreateVariableExpenseInput())
	require.NoError(t, err)

	err = ve.ApplyUpdate(UpdateVariableExpenseInput{
		UserID: "different-user",
	})
	require.ErrorIs(t, err, ErrAccessDenied)
}

func TestVariableExpense_ApplyUpdate_ValidationErrors(t *testing.T) {
	ve, err := NewVariableExpense(validCreateVariableExpenseInput())
	require.NoError(t, err)

	tests := []struct {
		name  string
		tweak func(input *UpdateVariableExpenseInput)
		err   error
	}{
		{
			name:  "empty description",
			tweak: func(input *UpdateVariableExpenseInput) { d := ""; input.Description = &d },
			err:   ErrMissingField,
		},
		{
			name:  "empty destination",
			tweak: func(input *UpdateVariableExpenseInput) { d := ""; input.Destination = &d },
			err:   ErrMissingField,
		},
		{
			name:  "empty category",
			tweak: func(input *UpdateVariableExpenseInput) { c := ""; input.Category = &c },
			err:   ErrMissingField,
		},
		{
			name:  "invalid expense_type",
			tweak: func(input *UpdateVariableExpenseInput) { e := ExpenseType("luxury"); input.ExpenseType = &e },
			err:   ErrInvalidEnum,
		},
		{
			name:  "invalid payment method",
			tweak: func(input *UpdateVariableExpenseInput) { p := PaymentMethod("eth"); input.PaymentMethod = &p },
			err:   ErrInvalidEnum,
		},
		{
			name:  "empty payment_date",
			tweak: func(input *UpdateVariableExpenseInput) { d := ""; input.PaymentDate = &d },
			err:   ErrMissingField,
		},
		{
			name:  "negative paid_amount",
			tweak: func(input *UpdateVariableExpenseInput) { a := int64(-1); input.PaidAmount = &a },
			err:   ErrNegativeAmount,
		},
		{
			name:  "invalid status",
			tweak: func(input *UpdateVariableExpenseInput) { s := TransactionStatus("bogus"); input.Status = &s },
			err:   ErrInvalidStatus,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modified := UpdateVariableExpenseInput{UserID: ve.UserID}
			tt.tweak(&modified)
			cloned := *ve
			err := cloned.ApplyUpdate(modified)
			require.ErrorIs(t, err, tt.err)
		})
	}
}
