package domain_test

import (
	"testing"

	"github.com/aureum/debt-svc/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr(s string) *string { return &s }

func TestDebtType_Valid(t *testing.T) {
	tests := []struct {
		name  string
		dt    domain.DebtType
		valid bool
	}{
		{"personal_loan", domain.DebtTypePersonalLoan, true},
		{"student_loan", domain.DebtTypeStudentLoan, true},
		{"mortgage", domain.DebtTypeMortgage, true},
		{"car_loan", domain.DebtTypeCarLoan, true},
		{"credit_card_debt", domain.DebtTypeCreditCardDebt, true},
		{"medical_debt", domain.DebtTypeMedicalDebt, true},
		{"other", domain.DebtTypeOther, true},
		{"invalid_type", domain.DebtType("invalid_type"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.dt.Valid())
		})
	}
}

func TestDebtStatus_Valid(t *testing.T) {
	tests := []struct {
		name  string
		ds    domain.DebtStatus
		valid bool
	}{
		{"active", domain.DebtStatusActive, true},
		{"paused", domain.DebtStatusPaused, true},
		{"paid_off", domain.DebtStatusPaidOff, true},
		{"defaulted", domain.DebtStatusDefaulted, true},
		{"settled", domain.DebtStatusSettled, true},
		{"invalid_status", domain.DebtStatus("invalid_status"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.ds.Valid())
		})
	}
}

func TestNewDebt(t *testing.T) {
	tests := []struct {
		name    string
		input   domain.CreateDebtInput
		wantErr error
	}{
		{
			name: "valid",
			input: domain.CreateDebtInput{
				UserID:      "user-1",
				Name:        "Car Loan",
				DebtType:    domain.DebtTypeCarLoan,
				TotalAmount: 10000000,
				StartDate:   "2024-01-01",
				Status:      domain.DebtStatusActive,
			},
			wantErr: nil,
		},
		{
			name: "missing user id",
			input: domain.CreateDebtInput{
				UserID:      "",
				Name:        "Car Loan",
				DebtType:    domain.DebtTypeCarLoan,
				TotalAmount: 10000000,
				StartDate:   "2024-01-01",
				Status:      domain.DebtStatusActive,
			},
			wantErr: domain.ErrMissingField,
		},
		{
			name: "missing name",
			input: domain.CreateDebtInput{
				UserID:      "user-1",
				Name:        "",
				DebtType:    domain.DebtTypeCarLoan,
				TotalAmount: 10000000,
				StartDate:   "2024-01-01",
				Status:      domain.DebtStatusActive,
			},
			wantErr: domain.ErrMissingField,
		},
		{
			name: "empty debt type",
			input: domain.CreateDebtInput{
				UserID:      "user-1",
				Name:        "Car Loan",
				DebtType:    "",
				TotalAmount: 10000000,
				StartDate:   "2024-01-01",
				Status:      domain.DebtStatusActive,
			},
			wantErr: domain.ErrInvalidDebtType,
		},
		{
			name: "invalid debt type",
			input: domain.CreateDebtInput{
				UserID:      "user-1",
				Name:        "Car Loan",
				DebtType:    domain.DebtType("fake_type"),
				TotalAmount: 10000000,
				StartDate:   "2024-01-01",
				Status:      domain.DebtStatusActive,
			},
			wantErr: domain.ErrInvalidDebtType,
		},
		{
			name: "total amount zero",
			input: domain.CreateDebtInput{
				UserID:      "user-1",
				Name:        "Car Loan",
				DebtType:    domain.DebtTypeCarLoan,
				TotalAmount: 0,
				StartDate:   "2024-01-01",
				Status:      domain.DebtStatusActive,
			},
			wantErr: domain.ErrNegativeAmount,
		},
		{
			name: "total amount negative",
			input: domain.CreateDebtInput{
				UserID:      "user-1",
				Name:        "Car Loan",
				DebtType:    domain.DebtTypeCarLoan,
				TotalAmount: -100,
				StartDate:   "2024-01-01",
				Status:      domain.DebtStatusActive,
			},
			wantErr: domain.ErrNegativeAmount,
		},
		{
			name: "missing start date",
			input: domain.CreateDebtInput{
				UserID:      "user-1",
				Name:        "Car Loan",
				DebtType:    domain.DebtTypeCarLoan,
				TotalAmount: 10000000,
				StartDate:   "",
				Status:      domain.DebtStatusActive,
			},
			wantErr: domain.ErrMissingField,
		},
		{
			name: "missing status",
			input: domain.CreateDebtInput{
				UserID:      "user-1",
				Name:        "Car Loan",
				DebtType:    domain.DebtTypeCarLoan,
				TotalAmount: 10000000,
				StartDate:   "2024-01-01",
				Status:      "",
			},
			wantErr: domain.ErrMissingField,
		},
		{
			name: "invalid status",
			input: domain.CreateDebtInput{
				UserID:      "user-1",
				Name:        "Car Loan",
				DebtType:    domain.DebtTypeCarLoan,
				TotalAmount: 10000000,
				StartDate:   "2024-01-01",
				Status:      domain.DebtStatus("fake_status"),
			},
			wantErr: domain.ErrInvalidStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			debt, err := domain.NewDebt(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, debt)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, debt)
			assert.Equal(t, tt.input.UserID, debt.UserID)
			assert.Equal(t, tt.input.Name, debt.Name)
			assert.Equal(t, tt.input.DebtType, debt.DebtType)
			assert.Equal(t, tt.input.TotalAmount, debt.TotalAmount)
			assert.Equal(t, tt.input.TotalAmount, debt.RemainingAmount)
			assert.Equal(t, tt.input.StartDate, debt.StartDate)
			assert.Equal(t, tt.input.Status, debt.Status)
			assert.NotZero(t, debt.CreatedAt)
			assert.NotZero(t, debt.UpdatedAt)
		})
	}
}

func TestApplyUpdate(t *testing.T) {
	debt := &domain.Debt{
		UserID:          "user-1",
		Name:            "Original Name",
		DebtType:        domain.DebtTypeCarLoan,
		TotalAmount:     10000000,
		RemainingAmount: 10000000,
		Status:          domain.DebtStatusActive,
	}

	t.Run("access denied", func(t *testing.T) {
		err := debt.ApplyUpdate(domain.UpdateDebtInput{UserID: "other-user"})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrAccessDenied)
	})

	t.Run("update name", func(t *testing.T) {
		d := *debt
		err := d.ApplyUpdate(domain.UpdateDebtInput{UserID: "", Name: ptr("New Name")})
		require.NoError(t, err)
		assert.Equal(t, "New Name", d.Name)
	})

	t.Run("update name to empty", func(t *testing.T) {
		err := debt.ApplyUpdate(domain.UpdateDebtInput{Name: ptr("")})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrMissingField)
	})

	t.Run("update debt type", func(t *testing.T) {
		d := *debt
		mtg := domain.DebtTypeMortgage
		err := d.ApplyUpdate(domain.UpdateDebtInput{DebtType: &mtg})
		require.NoError(t, err)
		assert.Equal(t, domain.DebtTypeMortgage, d.DebtType)
	})

	t.Run("update debt type invalid", func(t *testing.T) {
		invalid := domain.DebtType("fake")
		err := debt.ApplyUpdate(domain.UpdateDebtInput{DebtType: &invalid})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidDebtType)
	})

	t.Run("update total amount", func(t *testing.T) {
		d := *debt
		amount := int64(20000000)
		err := d.ApplyUpdate(domain.UpdateDebtInput{TotalAmount: &amount})
		require.NoError(t, err)
		assert.Equal(t, int64(20000000), d.TotalAmount)
	})

	t.Run("update total amount negative", func(t *testing.T) {
		neg := int64(-100)
		err := debt.ApplyUpdate(domain.UpdateDebtInput{TotalAmount: &neg})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNegativeAmount)
	})

	t.Run("update status valid transition", func(t *testing.T) {
		d := *debt
		paused := domain.DebtStatusPaused
		err := d.ApplyUpdate(domain.UpdateDebtInput{Status: &paused})
		require.NoError(t, err)
		assert.Equal(t, domain.DebtStatusPaused, d.Status)
	})

	t.Run("update status invalid transition", func(t *testing.T) {
		d := &domain.Debt{Status: domain.DebtStatusPaidOff}
		active := domain.DebtStatusActive
		err := d.ApplyUpdate(domain.UpdateDebtInput{Status: &active})
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrStatusTransition)
	})
}

func TestTransitionStatus(t *testing.T) {
	tests := []struct {
		name    string
		from    domain.DebtStatus
		to      domain.DebtStatus
		wantErr error
	}{
		{"active to paused", domain.DebtStatusActive, domain.DebtStatusPaused, nil},
		{"active to paid_off", domain.DebtStatusActive, domain.DebtStatusPaidOff, nil},
		{"active to defaulted", domain.DebtStatusActive, domain.DebtStatusDefaulted, nil},
		{"active to settled", domain.DebtStatusActive, domain.DebtStatusSettled, nil},
		{"paused to active", domain.DebtStatusPaused, domain.DebtStatusActive, nil},
		{"paused to paid_off", domain.DebtStatusPaused, domain.DebtStatusPaidOff, nil},
		{"paused to defaulted", domain.DebtStatusPaused, domain.DebtStatusDefaulted, nil},
		{"paused to settled", domain.DebtStatusPaused, domain.DebtStatusSettled, nil},
		{"defaulted to settled", domain.DebtStatusDefaulted, domain.DebtStatusSettled, nil},
		{"paid_off to active", domain.DebtStatusPaidOff, domain.DebtStatusActive, domain.ErrStatusTransition},
		{"paid_off to paused", domain.DebtStatusPaidOff, domain.DebtStatusPaused, domain.ErrStatusTransition},
		{"paid_off to defaulted", domain.DebtStatusPaidOff, domain.DebtStatusDefaulted, domain.ErrStatusTransition},
		{"paid_off to settled", domain.DebtStatusPaidOff, domain.DebtStatusSettled, domain.ErrStatusTransition},
		{"settled to active", domain.DebtStatusSettled, domain.DebtStatusActive, domain.ErrStatusTransition},
		{"settled to paused", domain.DebtStatusSettled, domain.DebtStatusPaused, domain.ErrStatusTransition},
		{"settled to paid_off", domain.DebtStatusSettled, domain.DebtStatusPaidOff, domain.ErrStatusTransition},
		{"settled to defaulted", domain.DebtStatusSettled, domain.DebtStatusDefaulted, domain.ErrStatusTransition},
		{"invalid target status", domain.DebtStatusActive, domain.DebtStatus("invalid"), domain.ErrInvalidStatus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &domain.Debt{Status: tt.from}
			err := d.TransitionStatus(tt.to)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.to, d.Status)
		})
	}
}

func TestApplyPayment(t *testing.T) {
	t.Run("valid payment reduces remaining", func(t *testing.T) {
		d := &domain.Debt{RemainingAmount: 10000000, Status: domain.DebtStatusActive}
		err := d.ApplyPayment(3000000)
		require.NoError(t, err)
		assert.Equal(t, int64(7000000), d.RemainingAmount)
		assert.Equal(t, domain.DebtStatusActive, d.Status)
		assert.NotZero(t, d.UpdatedAt)
	})

	t.Run("full payment marks paid off", func(t *testing.T) {
		d := &domain.Debt{RemainingAmount: 10000000, Status: domain.DebtStatusActive}
		err := d.ApplyPayment(10000000)
		require.NoError(t, err)
		assert.Equal(t, int64(0), d.RemainingAmount)
		assert.Equal(t, domain.DebtStatusPaidOff, d.Status)
	})

	t.Run("already paid returns error", func(t *testing.T) {
		d := &domain.Debt{RemainingAmount: 0, Status: domain.DebtStatusPaidOff}
		err := d.ApplyPayment(1000)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrDebtAlreadyPaid)
	})

	t.Run("payment exceeds remaining balance", func(t *testing.T) {
		d := &domain.Debt{RemainingAmount: 500000, Status: domain.DebtStatusActive}
		err := d.ApplyPayment(1000000)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrPaymentExceedsBalance)
	})

	t.Run("zero amount returns error", func(t *testing.T) {
		d := &domain.Debt{RemainingAmount: 10000000, Status: domain.DebtStatusActive}
		err := d.ApplyPayment(0)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNegativeAmount)
	})

	t.Run("negative amount returns error", func(t *testing.T) {
		d := &domain.Debt{RemainingAmount: 10000000, Status: domain.DebtStatusActive}
		err := d.ApplyPayment(-500)
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNegativeAmount)
	})
}

func TestNewPayment(t *testing.T) {
	tests := []struct {
		name    string
		input   domain.RegisterPaymentInput
		wantErr error
	}{
		{
			name: "valid",
			input: domain.RegisterPaymentInput{
				DebtID:      "debt-1",
				UserID:      "user-1",
				Amount:      50000,
				PaymentDate: "2024-01-15",
				Notes:       "test payment",
			},
			wantErr: nil,
		},
		{
			name: "missing debt id",
			input: domain.RegisterPaymentInput{
				DebtID:      "",
				UserID:      "user-1",
				Amount:      50000,
				PaymentDate: "2024-01-15",
			},
			wantErr: domain.ErrMissingField,
		},
		{
			name: "missing user id",
			input: domain.RegisterPaymentInput{
				DebtID:      "debt-1",
				UserID:      "",
				Amount:      50000,
				PaymentDate: "2024-01-15",
			},
			wantErr: domain.ErrMissingField,
		},
		{
			name: "zero amount",
			input: domain.RegisterPaymentInput{
				DebtID:      "debt-1",
				UserID:      "user-1",
				Amount:      0,
				PaymentDate: "2024-01-15",
			},
			wantErr: domain.ErrNegativeAmount,
		},
		{
			name: "negative amount",
			input: domain.RegisterPaymentInput{
				DebtID:      "debt-1",
				UserID:      "user-1",
				Amount:      -100,
				PaymentDate: "2024-01-15",
			},
			wantErr: domain.ErrNegativeAmount,
		},
		{
			name: "missing payment date",
			input: domain.RegisterPaymentInput{
				DebtID:      "debt-1",
				UserID:      "user-1",
				Amount:      50000,
				PaymentDate: "",
			},
			wantErr: domain.ErrMissingField,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment, err := domain.NewPayment(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, payment)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, payment)
			assert.Equal(t, tt.input.DebtID, payment.DebtID)
			assert.Equal(t, tt.input.UserID, payment.UserID)
			assert.Equal(t, tt.input.Amount, payment.Amount)
			assert.Equal(t, tt.input.PaymentDate, payment.PaymentDate)
			assert.Equal(t, tt.input.Notes, payment.Notes)
			assert.NotZero(t, payment.CreatedAt)
		})
	}
}
