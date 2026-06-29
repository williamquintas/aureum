package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aureum/debt-svc/internal/domain"
)

func TestCalculateAmortization(t *testing.T) {
	t.Run("standard amortization with zero interest", func(t *testing.T) {
		s := domain.CalculateAmortization(10000000, 0, 1000000, 12)

		assert.Equal(t, int64(10000000), s.TotalAmount)
		assert.Equal(t, int64(1000000), s.MonthlyPayment)
		assert.Equal(t, int64(0), s.InterestRate)
		assert.Equal(t, int64(0), s.TotalInterest)
		assert.Equal(t, int64(10000000), s.TotalPaid)
		assert.LessOrEqual(t, int(s.RemainingMonths), 12)
		assert.NotEmpty(t, s.Entries)

		last := s.Entries[len(s.Entries)-1]
		assert.Equal(t, int64(0), last.Balance)

		sumPrincipal := int64(0)
		for _, e := range s.Entries {
			sumPrincipal += e.Principal
			assert.Equal(t, int64(0), e.Interest)
		}
		assert.Equal(t, int64(10000000), sumPrincipal)
	})

	t.Run("amortization with interest rate", func(t *testing.T) {
		s := domain.CalculateAmortization(10000000, 500, 1000000, 12)

		assert.Equal(t, int64(10000000), s.TotalAmount)
		assert.Equal(t, int64(500), s.InterestRate)
		assert.Greater(t, s.TotalInterest, int64(0))
		assert.Greater(t, s.TotalPaid, s.TotalAmount)
		assert.Equal(t, s.TotalAmount+s.TotalInterest, s.TotalPaid)

		last := s.Entries[len(s.Entries)-1]
		assert.Equal(t, int64(0), last.Balance)

		sumPrincipal := int64(0)
		for _, e := range s.Entries {
			sumPrincipal += e.Principal
			assert.Equal(t, e.Principal+e.Interest, e.TotalPayment)
		}
		assert.Equal(t, int64(10000000), sumPrincipal)
	})

	t.Run("payment too small to cover interest", func(t *testing.T) {
		s := domain.CalculateAmortization(100000, 120000, 100, 12)

		assert.Equal(t, int32(1), s.RemainingMonths)
		assert.Len(t, s.Entries, 1)
		assert.Equal(t, int64(0), s.Entries[0].Principal)
		assert.Equal(t, int64(100000), s.Entries[0].Interest)
		assert.Equal(t, int64(100000), s.Entries[0].TotalPayment)
		assert.Equal(t, int64(0), s.Entries[0].Balance)
		assert.Equal(t, int64(100000), s.TotalInterest)
		assert.Equal(t, int64(200000), s.TotalPaid)
	})

	t.Run("single month remaining", func(t *testing.T) {
		s := domain.CalculateAmortization(500000, 1000, 1000000, 1)

		assert.Equal(t, int32(1), s.RemainingMonths)
		assert.Len(t, s.Entries, 1)
		assert.Equal(t, int64(500000), s.Entries[0].Principal)
		assert.Greater(t, s.Entries[0].Interest, int64(0))
		assert.Equal(t, int64(0), s.Entries[0].Balance)
	})

	t.Run("entry count matches months when not paid early", func(t *testing.T) {
		s := domain.CalculateAmortization(1000000000, 0, 100, 12)

		assert.Len(t, s.Entries, 12)
	})

	t.Run("total interest plus total amount equals total paid", func(t *testing.T) {
		s := domain.CalculateAmortization(10000000, 750, 200380, 60)

		assert.Equal(t, s.TotalAmount+s.TotalInterest, s.TotalPaid)

		sumPrincipal := int64(0)
		sumInterest := int64(0)
		for _, e := range s.Entries {
			sumPrincipal += e.Principal
			sumInterest += e.Interest
		}
		assert.Equal(t, s.TotalAmount, sumPrincipal)
	})

	t.Run("verify rounding", func(t *testing.T) {
		s := domain.CalculateAmortization(10000000, 750, 200380, 60)

		for _, e := range s.Entries {
			assert.Equal(t, e.Principal+e.Interest, e.TotalPayment,
				"month %d: Principal(%d) + Interest(%d) != TotalPayment(%d)",
				e.Month, e.Principal, e.Interest, e.TotalPayment)
		}

		sumPrincipal := int64(0)
		sumInterest := int64(0)
		for _, e := range s.Entries {
			sumPrincipal += e.Principal
			sumInterest += e.Interest
		}
		assert.Equal(t, s.TotalAmount, sumPrincipal,
			"sum of principal payments (%d) != total amount (%d)", sumPrincipal, s.TotalAmount)
		assert.Equal(t, s.TotalInterest, sumInterest,
			"sum of interest payments (%d) != total interest (%d)", sumInterest, s.TotalInterest)
	})

	t.Run("final payment balance", func(t *testing.T) {
		testCases := []struct {
			name         string
			totalAmount  int64
			interestRate int64
			monthlyPay   int64
			months       int
		}{
			{"5yr @ 7.5%", 10000000, 750, 200380, 60},
			{"zero interest, 12mo", 10000000, 0, 1000000, 12},
			{"high interest, 12mo", 10000000, 500, 1000000, 12},
			{"single month", 500000, 1000, 1000000, 1},
			{"small amount", 100000, 120000, 100, 12},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				s := domain.CalculateAmortization(tc.totalAmount, tc.interestRate, tc.monthlyPay, tc.months)

				require := assert.New(t)
				last := s.Entries[len(s.Entries)-1]
				require.Equal(int64(0), last.Balance,
					"month %d: final balance should be 0, got %d", last.Month, last.Balance)
			})
		}
	})
}

func TestComputeMonthlyPayment(t *testing.T) {
	t.Run("zero interest", func(t *testing.T) {
		payment := domain.ComputeMonthlyPayment(12000000, 0, 12)
		assert.Equal(t, int64(1000000), payment)
	})

	t.Run("with interest", func(t *testing.T) {
		payment := domain.ComputeMonthlyPayment(10000000, 750, 12)
		assert.Greater(t, payment, int64(833334))
		assert.Less(t, payment, int64(900000))
	})

	t.Run("single month", func(t *testing.T) {
		payment := domain.ComputeMonthlyPayment(500000, 1000, 1)
		assert.Greater(t, payment, int64(500000))
		assert.Less(t, payment, int64(510000))
	})

	t.Run("zero months returns zero", func(t *testing.T) {
		payment := domain.ComputeMonthlyPayment(100000, 500, 0)
		assert.Equal(t, int64(0), payment)
	})
}

func TestMonthsBetween(t *testing.T) {
	t.Run("same month", func(t *testing.T) {
		n, err := domain.MonthsBetween("2024-01-01", "2024-01-31")
		require.NoError(t, err)
		assert.Equal(t, 1, n)
	})

	t.Run("exactly one year", func(t *testing.T) {
		n, err := domain.MonthsBetween("2024-01-01", "2025-01-01")
		require.NoError(t, err)
		assert.Equal(t, 12, n)
	})

	t.Run("multiple years", func(t *testing.T) {
		n, err := domain.MonthsBetween("2024-01-01", "2029-01-01")
		require.NoError(t, err)
		assert.Equal(t, 60, n)
	})

	t.Run("invalid start date", func(t *testing.T) {
		_, err := domain.MonthsBetween("not-a-date", "2024-02-01")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidDate)
	})

	t.Run("invalid end date", func(t *testing.T) {
		_, err := domain.MonthsBetween("2024-01-01", "not-a-date")
		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidDate)
	})
}
