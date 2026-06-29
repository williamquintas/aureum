//nolint:goconst // test file - repeated strings acceptable
package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func validCreateIncomeInput() CreateIncomeInput {
	return CreateIncomeInput{
		UserID:         "user-1",
		Description:    "Freelance project",
		Source:         "Upwork",
		IncomeType:     IncomeTypeFreelance,
		ReceivedDate:   "2026-05-01",
		ReceivedAmount: 500000,
		Status:         StatusPending,
	}
}

func TestNewIncome_Success(t *testing.T) {
	input := validCreateIncomeInput()
	income, err := NewIncome(input)
	require.NoError(t, err)
	require.NotNil(t, income)
	require.Equal(t, input.UserID, income.UserID)
	require.Equal(t, input.Description, income.Description)
	require.Equal(t, input.Source, income.Source)
	require.Equal(t, input.IncomeType, income.IncomeType)
	require.Equal(t, input.ReceivedDate, income.ReceivedDate)
	require.Equal(t, input.ReceivedAmount, income.ReceivedAmount)
	require.Equal(t, input.Status, income.Status)
	require.False(t, income.CreatedAt.IsZero())
	require.False(t, income.UpdatedAt.IsZero())
	require.Nil(t, income.DeletedAt)
}

func TestNewIncome_ValidationErrors(t *testing.T) {
	tests := []struct {
		name  string
		tweak func(input *CreateIncomeInput)
		err   error
	}{
		{
			name:  "empty user_id",
			tweak: func(input *CreateIncomeInput) { input.UserID = "" },
			err:   ErrMissingField,
		},
		{
			name:  "empty description",
			tweak: func(input *CreateIncomeInput) { input.Description = "" },
			err:   ErrMissingField,
		},
		{
			name:  "empty source",
			tweak: func(input *CreateIncomeInput) { input.Source = "" },
			err:   ErrMissingField,
		},
		{
			name:  "empty income_type",
			tweak: func(input *CreateIncomeInput) { input.IncomeType = "" },
			err:   ErrMissingField,
		},
		{
			name:  "invalid income_type",
			tweak: func(input *CreateIncomeInput) { input.IncomeType = "crypto" },
			err:   ErrInvalidEnum,
		},
		{
			name:  "empty received_date",
			tweak: func(input *CreateIncomeInput) { input.ReceivedDate = "" },
			err:   ErrMissingField,
		},
		{
			name:  "zero amount",
			tweak: func(input *CreateIncomeInput) { input.ReceivedAmount = 0 },
			err:   ErrNegativeAmount,
		},
		{
			name:  "negative amount",
			tweak: func(input *CreateIncomeInput) { input.ReceivedAmount = -100 },
			err:   ErrNegativeAmount,
		},
		{
			name:  "empty status",
			tweak: func(input *CreateIncomeInput) { input.Status = "" },
			err:   ErrMissingField,
		},
		{
			name:  "invalid status",
			tweak: func(input *CreateIncomeInput) { input.Status = "archived" },
			err:   ErrInvalidStatus,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validCreateIncomeInput()
			tt.tweak(&input)
			_, err := NewIncome(input)
			require.ErrorIs(t, err, tt.err)
		})
	}
}

func TestIncome_TransitionStatus(t *testing.T) {
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
			name:    "completed to cancelled",
			from:    StatusCompleted,
			to:      StatusCancelled,
			wantErr: ErrStatusTransition,
		},
		{
			name:    "cancelled to pending",
			from:    StatusCancelled,
			to:      StatusPending,
			wantErr: ErrStatusTransition,
		},
		{
			name:    "pending to invalid",
			from:    StatusPending,
			to:      "bogus",
			wantErr: ErrInvalidStatus,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validCreateIncomeInput()
			input.Status = tt.from
			income, err := NewIncome(input)
			require.NoError(t, err)

			err = income.TransitionStatus(tt.to)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, tt.from, income.Status)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.to, income.Status)
			}
		})
	}
}

func TestIncome_ApplyUpdate_Success(t *testing.T) {
	income, err := NewIncome(validCreateIncomeInput())
	require.NoError(t, err)

	newDesc := "Updated project"
	newAmount := int64(600000)
	completed := StatusCompleted

	err = income.ApplyUpdate(UpdateIncomeInput{
		UserID:         income.UserID,
		Description:    &newDesc,
		ReceivedAmount: &newAmount,
		Status:         &completed,
	})

	require.NoError(t, err)
	require.Equal(t, newDesc, income.Description)
	require.Equal(t, newAmount, income.ReceivedAmount)
	require.Equal(t, StatusCompleted, income.Status)
}

func TestIncome_ApplyUpdate_AccessDenied(t *testing.T) {
	income, err := NewIncome(validCreateIncomeInput())
	require.NoError(t, err)

	err = income.ApplyUpdate(UpdateIncomeInput{
		UserID: "different-user",
	})
	require.ErrorIs(t, err, ErrAccessDenied)
}

func TestIncome_ApplyUpdate_ValidationErrors(t *testing.T) {
	income, err := NewIncome(validCreateIncomeInput())
	require.NoError(t, err)

	tests := []struct {
		name  string
		tweak func(input *UpdateIncomeInput)
		err   error
	}{
		{
			name:  "empty description",
			tweak: func(input *UpdateIncomeInput) { desc := ""; input.Description = &desc },
			err:   ErrMissingField,
		},
		{
			name:  "empty source",
			tweak: func(input *UpdateIncomeInput) { src := ""; input.Source = &src },
			err:   ErrMissingField,
		},
		{
			name:  "invalid income_type",
			tweak: func(input *UpdateIncomeInput) { it := IncomeType("crypto"); input.IncomeType = &it },
			err:   ErrInvalidEnum,
		},
		{
			name:  "empty date",
			tweak: func(input *UpdateIncomeInput) { d := ""; input.ReceivedDate = &d },
			err:   ErrMissingField,
		},
		{
			name:  "negative amount",
			tweak: func(input *UpdateIncomeInput) { a := int64(-1); input.ReceivedAmount = &a },
			err:   ErrNegativeAmount,
		},
		{
			name:  "invalid status",
			tweak: func(input *UpdateIncomeInput) { s := TransactionStatus("bogus"); input.Status = &s },
			err:   ErrInvalidStatus,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modified := UpdateIncomeInput{UserID: income.UserID}
			tt.tweak(&modified)
			cloned := *income
			err := cloned.ApplyUpdate(modified)
			require.ErrorIs(t, err, tt.err)
		})
	}
}
