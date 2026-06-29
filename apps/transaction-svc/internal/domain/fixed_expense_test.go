//nolint:goconst // test file - repeated strings acceptable
package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func validCreateFixedExpenseInput() CreateFixedExpenseInput {
	return CreateFixedExpenseInput{
		UserID:        "user-1",
		Description:   "Netflix",
		Category:      "Entertainment",
		DayOfMonth:    15,
		PaymentMethod: PaymentMethodCreditCard,
		Status:        StatusPending,
	}
}

func TestNewFixedExpense_Success(t *testing.T) {
	input := validCreateFixedExpenseInput()
	fe, err := NewFixedExpense(input)
	require.NoError(t, err)
	require.NotNil(t, fe)
	require.Equal(t, input.UserID, fe.UserID)
	require.Equal(t, input.Description, fe.Description)
	require.Equal(t, input.Category, fe.Category)
	require.Equal(t, input.DayOfMonth, fe.DayOfMonth)
	require.Equal(t, input.PaymentMethod, fe.PaymentMethod)
	require.Equal(t, input.Status, fe.Status)
	require.False(t, fe.CreatedAt.IsZero())
}

func TestNewFixedExpense_ValidationErrors(t *testing.T) {
	tests := []struct {
		name  string
		tweak func(input *CreateFixedExpenseInput)
		err   error
	}{
		{
			name:  "empty user_id",
			tweak: func(input *CreateFixedExpenseInput) { input.UserID = "" },
			err:   ErrMissingField,
		},
		{
			name:  "empty description",
			tweak: func(input *CreateFixedExpenseInput) { input.Description = "" },
			err:   ErrMissingField,
		},
		{
			name:  "empty category",
			tweak: func(input *CreateFixedExpenseInput) { input.Category = "" },
			err:   ErrMissingField,
		},
		{
			name:  "day_of_month too low",
			tweak: func(input *CreateFixedExpenseInput) { input.DayOfMonth = 0 },
			err:   ErrInvalidDay,
		},
		{
			name:  "day_of_month too high",
			tweak: func(input *CreateFixedExpenseInput) { input.DayOfMonth = 32 },
			err:   ErrInvalidDay,
		},
		{
			name:  "empty payment method",
			tweak: func(input *CreateFixedExpenseInput) { input.PaymentMethod = "" },
			err:   ErrMissingField,
		},
		{
			name:  "invalid payment method",
			tweak: func(input *CreateFixedExpenseInput) { input.PaymentMethod = "bitcoin" },
			err:   ErrInvalidEnum,
		},
		{
			name:  "empty status",
			tweak: func(input *CreateFixedExpenseInput) { input.Status = "" },
			err:   ErrMissingField,
		},
		{
			name:  "invalid status",
			tweak: func(input *CreateFixedExpenseInput) { input.Status = "archived" },
			err:   ErrInvalidStatus,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validCreateFixedExpenseInput()
			tt.tweak(&input)
			_, err := NewFixedExpense(input)
			require.ErrorIs(t, err, tt.err)
		})
	}
}

func TestFixedExpense_TransitionStatus(t *testing.T) {
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
			name:    "cancelled to pending",
			from:    StatusCancelled,
			to:      StatusPending,
			wantErr: ErrStatusTransition,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validCreateFixedExpenseInput()
			input.Status = tt.from
			fe, err := NewFixedExpense(input)
			require.NoError(t, err)

			err = fe.TransitionStatus(tt.to)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, tt.from, fe.Status)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.to, fe.Status)
			}
		})
	}
}

func TestFixedExpense_ApplyUpdate_Success(t *testing.T) {
	fe, err := NewFixedExpense(validCreateFixedExpenseInput())
	require.NoError(t, err)

	newDesc := "Spotify"
	newDay := 10
	pix := PaymentMethodPix

	err = fe.ApplyUpdate(UpdateFixedExpenseInput{
		UserID:        fe.UserID,
		Description:   &newDesc,
		DayOfMonth:    &newDay,
		PaymentMethod: &pix,
	})

	require.NoError(t, err)
	require.Equal(t, newDesc, fe.Description)
	require.Equal(t, newDay, fe.DayOfMonth)
	require.Equal(t, PaymentMethodPix, fe.PaymentMethod)
}

func TestFixedExpense_ApplyUpdate_AccessDenied(t *testing.T) {
	fe, err := NewFixedExpense(validCreateFixedExpenseInput())
	require.NoError(t, err)

	err = fe.ApplyUpdate(UpdateFixedExpenseInput{
		UserID: "different-user",
	})
	require.ErrorIs(t, err, ErrAccessDenied)
}

func TestFixedExpense_ApplyUpdate_ValidationErrors(t *testing.T) {
	fe, err := NewFixedExpense(validCreateFixedExpenseInput())
	require.NoError(t, err)

	tests := []struct {
		name  string
		tweak func(input *UpdateFixedExpenseInput)
		err   error
	}{
		{
			name:  "empty description",
			tweak: func(input *UpdateFixedExpenseInput) { d := ""; input.Description = &d },
			err:   ErrMissingField,
		},
		{
			name:  "empty category",
			tweak: func(input *UpdateFixedExpenseInput) { c := ""; input.Category = &c },
			err:   ErrMissingField,
		},
		{
			name:  "invalid day_of_month",
			tweak: func(input *UpdateFixedExpenseInput) { d := 0; input.DayOfMonth = &d },
			err:   ErrInvalidDay,
		},
		{
			name:  "invalid payment method",
			tweak: func(input *UpdateFixedExpenseInput) { p := PaymentMethod("eth"); input.PaymentMethod = &p },
			err:   ErrInvalidEnum,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modified := UpdateFixedExpenseInput{UserID: fe.UserID}
			tt.tweak(&modified)
			cloned := *fe
			err := cloned.ApplyUpdate(modified)
			require.ErrorIs(t, err, tt.err)
		})
	}
}
