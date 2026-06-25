package domain_test

import (
	"testing"

	"github.com/aureum/debt-svc/internal/domain"
	"github.com/stretchr/testify/assert"
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
