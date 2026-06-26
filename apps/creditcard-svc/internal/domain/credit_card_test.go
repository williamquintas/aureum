package domain_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aureum/creditcard-svc/internal/domain"
)

func TestCardBrand_Valid(t *testing.T) {
	tests := []struct {
		brand domain.CardBrand
		want  bool
	}{
		{domain.CardBrandVisa, true},
		{domain.CardBrandMastercard, true},
		{domain.CardBrandAmex, true},
		{domain.CardBrandElo, true},
		{domain.CardBrandHipercard, true},
		{domain.CardBrandDiners, true},
		{domain.CardBrandOther, true},
		{domain.CardBrand(""), false},
		{domain.CardBrand("invalid"), false},
		{domain.CardBrand("master"), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.brand), func(t *testing.T) {
			assert.Equal(t, tt.want, tt.brand.Valid())
		})
	}
}

func TestCardType_Valid(t *testing.T) {
	tests := []struct {
		cardType domain.CardType
		want     bool
	}{
		{domain.CardTypeCredit, true},
		{domain.CardTypeDebit, true},
		{domain.CardTypeMultiple, true},
		{domain.CardType(""), false},
		{domain.CardType("debit_card"), false},
		{domain.CardType("credito"), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.cardType), func(t *testing.T) {
			assert.Equal(t, tt.want, tt.cardType.Valid())
		})
	}
}

func TestInvoiceStatus_Valid(t *testing.T) {
	tests := []struct {
		status domain.InvoiceStatus
		want   bool
	}{
		{domain.InvoiceStatusOpen, true},
		{domain.InvoiceStatusClosed, true},
		{domain.InvoiceStatusPaid, true},
		{domain.InvoiceStatusOverdue, true},
		{domain.InvoiceStatus(""), false},
		{domain.InvoiceStatus("pending"), false},
		{domain.InvoiceStatus("cancelled"), false},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.Valid())
		})
	}
}

func TestNewCreditCard(t *testing.T) {
	validInput := func() domain.CreateCreditCardInput {
		return domain.CreateCreditCardInput{
			UserID:         "user-1",
			Name:           "My Card",
			Brand:          domain.CardBrandVisa,
			CardType:       domain.CardTypeCredit,
			LastFourDigits: "1234",
			ClosingDay:     15,
			DueDay:         10,
			CreditLimit:    500000,
		}
	}

	t.Run("success", func(t *testing.T) {
		input := validInput()
		card, err := domain.NewCreditCard(input)
		require.NoError(t, err)
		require.NotNil(t, card)
		assert.Equal(t, input.UserID, card.UserID)
		assert.Equal(t, input.Name, card.Name)
		assert.Equal(t, input.Brand, card.Brand)
		assert.Equal(t, input.CardType, card.CardType)
		assert.Equal(t, input.LastFourDigits, card.LastFourDigits)
		assert.Equal(t, input.ClosingDay, card.ClosingDay)
		assert.Equal(t, input.DueDay, card.DueDay)
		assert.Equal(t, input.CreditLimit, card.CreditLimit)
		assert.Equal(t, input.CreditLimit, card.AvailableCredit)
		assert.True(t, card.Active)
		assert.False(t, card.CreatedAt.IsZero())
		assert.False(t, card.UpdatedAt.IsZero())
	})

	t.Run("missing user id", func(t *testing.T) {
		input := validInput()
		input.UserID = ""
		card, err := domain.NewCreditCard(input)
		assert.Nil(t, card)
		assert.ErrorIs(t, err, domain.ErrMissingField)
	})

	t.Run("missing name", func(t *testing.T) {
		input := validInput()
		input.Name = ""
		card, err := domain.NewCreditCard(input)
		assert.Nil(t, card)
		assert.ErrorIs(t, err, domain.ErrMissingField)
	})

	t.Run("missing brand", func(t *testing.T) {
		input := validInput()
		input.Brand = ""
		card, err := domain.NewCreditCard(input)
		assert.Nil(t, card)
		assert.ErrorIs(t, err, domain.ErrMissingField)
	})

	t.Run("invalid brand", func(t *testing.T) {
		input := validInput()
		input.Brand = "bitcoin"
		card, err := domain.NewCreditCard(input)
		assert.Nil(t, card)
		assert.ErrorIs(t, err, domain.ErrInvalidCardBrand)
	})

	t.Run("missing card type", func(t *testing.T) {
		input := validInput()
		input.CardType = ""
		card, err := domain.NewCreditCard(input)
		assert.Nil(t, card)
		assert.ErrorIs(t, err, domain.ErrMissingField)
	})

	t.Run("invalid card type", func(t *testing.T) {
		input := validInput()
		input.CardType = "prepaid"
		card, err := domain.NewCreditCard(input)
		assert.Nil(t, card)
		assert.ErrorIs(t, err, domain.ErrInvalidCardType)
	})

	t.Run("missing last four digits", func(t *testing.T) {
		input := validInput()
		input.LastFourDigits = ""
		card, err := domain.NewCreditCard(input)
		assert.Nil(t, card)
		assert.ErrorIs(t, err, domain.ErrMissingField)
	})

	t.Run("closing day 0", func(t *testing.T) {
		input := validInput()
		input.ClosingDay = 0
		card, err := domain.NewCreditCard(input)
		assert.Nil(t, card)
		assert.ErrorIs(t, err, domain.ErrInvalidDay)
	})

	t.Run("closing day 32", func(t *testing.T) {
		input := validInput()
		input.ClosingDay = 32
		card, err := domain.NewCreditCard(input)
		assert.Nil(t, card)
		assert.ErrorIs(t, err, domain.ErrInvalidDay)
	})

	t.Run("due day 0", func(t *testing.T) {
		input := validInput()
		input.DueDay = 0
		card, err := domain.NewCreditCard(input)
		assert.Nil(t, card)
		assert.ErrorIs(t, err, domain.ErrInvalidDay)
	})

	t.Run("due day 32", func(t *testing.T) {
		input := validInput()
		input.DueDay = 32
		card, err := domain.NewCreditCard(input)
		assert.Nil(t, card)
		assert.ErrorIs(t, err, domain.ErrInvalidDay)
	})

	t.Run("negative credit limit", func(t *testing.T) {
		input := validInput()
		input.CreditLimit = -100
		card, err := domain.NewCreditCard(input)
		assert.Nil(t, card)
		assert.ErrorIs(t, err, domain.ErrNegativeAmount)
		assert.True(t, strings.Contains(err.Error(), "cannot be negative"))
	})
}

func TestApplyUpdate(t *testing.T) {
	makeCard := func() *domain.CreditCard {
		return &domain.CreditCard{
			ID:              "card-1",
			UserID:          "user-1",
			Name:            "My Card",
			Brand:           domain.CardBrandVisa,
			CardType:        domain.CardTypeCredit,
			LastFourDigits:  "1234",
			ClosingDay:      15,
			DueDay:          10,
			CreditLimit:     500000,
			AvailableCredit: 500000,
			Active:          true,
		}
	}

	t.Run("access denied", func(t *testing.T) {
		card := makeCard()
		err := card.ApplyUpdate(domain.UpdateCreditCardInput{
			UserID: "other-user",
		})
		assert.ErrorIs(t, err, domain.ErrAccessDenied)
	})

	t.Run("update name", func(t *testing.T) {
		card := makeCard()
		newName := "Updated Card"
		err := card.ApplyUpdate(domain.UpdateCreditCardInput{
			UserID: "user-1",
			Name:   &newName,
		})
		require.NoError(t, err)
		assert.Equal(t, newName, card.Name)
	})

	t.Run("update name to empty", func(t *testing.T) {
		card := makeCard()
		empty := ""
		err := card.ApplyUpdate(domain.UpdateCreditCardInput{
			UserID: "user-1",
			Name:   &empty,
		})
		assert.ErrorIs(t, err, domain.ErrMissingField)
	})

	t.Run("update closing day", func(t *testing.T) {
		card := makeCard()
		newDay := 20
		err := card.ApplyUpdate(domain.UpdateCreditCardInput{
			UserID:     "user-1",
			ClosingDay: &newDay,
		})
		require.NoError(t, err)
		assert.Equal(t, 20, card.ClosingDay)
	})

	t.Run("update closing day invalid", func(t *testing.T) {
		card := makeCard()
		badDay := 0
		err := card.ApplyUpdate(domain.UpdateCreditCardInput{
			UserID:     "user-1",
			ClosingDay: &badDay,
		})
		assert.ErrorIs(t, err, domain.ErrInvalidDay)

		badDay2 := 32
		err = card.ApplyUpdate(domain.UpdateCreditCardInput{
			UserID:     "user-1",
			ClosingDay: &badDay2,
		})
		assert.ErrorIs(t, err, domain.ErrInvalidDay)
	})

	t.Run("update due day", func(t *testing.T) {
		card := makeCard()
		newDay := 5
		err := card.ApplyUpdate(domain.UpdateCreditCardInput{
			UserID: "user-1",
			DueDay: &newDay,
		})
		require.NoError(t, err)
		assert.Equal(t, 5, card.DueDay)
	})

	t.Run("update due day invalid", func(t *testing.T) {
		card := makeCard()
		badDay := 0
		err := card.ApplyUpdate(domain.UpdateCreditCardInput{
			UserID: "user-1",
			DueDay: &badDay,
		})
		assert.ErrorIs(t, err, domain.ErrInvalidDay)
	})

	t.Run("update credit limit higher", func(t *testing.T) {
		card := makeCard()
		newLimit := int64(1000000)
		err := card.ApplyUpdate(domain.UpdateCreditCardInput{
			UserID:      "user-1",
			CreditLimit: &newLimit,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1000000), card.CreditLimit)
		assert.Equal(t, int64(1000000), card.AvailableCredit)
	})

	t.Run("update credit limit lower", func(t *testing.T) {
		card := makeCard()
		card.AvailableCredit = 300000
		newLimit := int64(400000)
		err := card.ApplyUpdate(domain.UpdateCreditCardInput{
			UserID:      "user-1",
			CreditLimit: &newLimit,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(400000), card.CreditLimit)
		assert.Equal(t, int64(200000), card.AvailableCredit)
	})

	t.Run("update credit limit negative", func(t *testing.T) {
		card := makeCard()
		neg := int64(-1)
		err := card.ApplyUpdate(domain.UpdateCreditCardInput{
			UserID:      "user-1",
			CreditLimit: &neg,
		})
		assert.ErrorIs(t, err, domain.ErrNegativeAmount)
	})

	t.Run("update active", func(t *testing.T) {
		card := makeCard()
		active := false
		err := card.ApplyUpdate(domain.UpdateCreditCardInput{
			UserID: "user-1",
			Active: &active,
		})
		require.NoError(t, err)
		assert.False(t, card.Active)
	})

	t.Run("updated at changes", func(t *testing.T) {
		card := makeCard()
		original := card.UpdatedAt
		time.Sleep(time.Millisecond)
		newName := "New Name"
		_ = card.ApplyUpdate(domain.UpdateCreditCardInput{
			UserID: "user-1",
			Name:   &newName,
		})
		assert.True(t, card.UpdatedAt.After(original))
	})
}

func TestNewInvoice(t *testing.T) {
	validInput := func() domain.CreateInvoiceInput {
		return domain.CreateInvoiceInput{
			CreditCardID:   "card-1",
			UserID:         "user-1",
			ReferenceMonth: "2026-06",
			ClosingDate:    "2026-06-20",
			DueDate:        "2026-07-10",
		}
	}

	t.Run("success", func(t *testing.T) {
		input := validInput()
		inv, err := domain.NewInvoice(input)
		require.NoError(t, err)
		require.NotNil(t, inv)
		assert.Equal(t, input.CreditCardID, inv.CreditCardID)
		assert.Equal(t, input.UserID, inv.UserID)
		assert.Equal(t, input.ReferenceMonth, inv.ReferenceMonth)
		assert.Equal(t, input.ClosingDate, inv.ClosingDate)
		assert.Equal(t, input.DueDate, inv.DueDate)
		assert.Equal(t, int64(0), inv.TotalAmount)
		assert.Equal(t, int64(0), inv.PaidAmount)
		assert.Equal(t, domain.InvoiceStatusOpen, inv.Status)
		assert.False(t, inv.CreatedAt.IsZero())
		assert.False(t, inv.UpdatedAt.IsZero())
	})

	t.Run("missing credit card id", func(t *testing.T) {
		input := validInput()
		input.CreditCardID = ""
		inv, err := domain.NewInvoice(input)
		assert.Nil(t, inv)
		assert.ErrorIs(t, err, domain.ErrMissingField)
	})

	t.Run("missing user id", func(t *testing.T) {
		input := validInput()
		input.UserID = ""
		inv, err := domain.NewInvoice(input)
		assert.Nil(t, inv)
		assert.ErrorIs(t, err, domain.ErrMissingField)
	})

	t.Run("missing reference month", func(t *testing.T) {
		input := validInput()
		input.ReferenceMonth = ""
		inv, err := domain.NewInvoice(input)
		assert.Nil(t, inv)
		assert.ErrorIs(t, err, domain.ErrMissingField)
	})

	t.Run("invalid reference month format", func(t *testing.T) {
		input := validInput()
		input.ReferenceMonth = "2026-13"
		inv, err := domain.NewInvoice(input)
		assert.Nil(t, inv)
		assert.ErrorIs(t, err, domain.ErrInvalidMonth)
	})

	t.Run("missing closing date", func(t *testing.T) {
		input := validInput()
		input.ClosingDate = ""
		inv, err := domain.NewInvoice(input)
		assert.Nil(t, inv)
		assert.ErrorIs(t, err, domain.ErrMissingField)
	})

	t.Run("missing due date", func(t *testing.T) {
		input := validInput()
		input.DueDate = ""
		inv, err := domain.NewInvoice(input)
		assert.Nil(t, inv)
		assert.ErrorIs(t, err, domain.ErrMissingField)
	})
}

func TestIsValidMonth(t *testing.T) {
	tests := []struct {
		month string
		want  bool
	}{
		{"2026-01", true},
		{"2026-12", true},
		{"2000-01", true},
		{"2100-12", true},
		{"", false},
		{"2026-1", false},
		{"2026-001", false},
		{"2026/01", false},
		{"2026-00", false},
		{"2026-13", false},
		{"1999-12", false},
		{"2101-01", false},
		{"abcd-01", false},
		{"2026-1-", false},
	}
	for _, tt := range tests {
		t.Run(tt.month, func(t *testing.T) {
			input := domain.CreateInvoiceInput{
				CreditCardID:   "card-1",
				UserID:         "user-1",
				ReferenceMonth: tt.month,
				ClosingDate:    "2026-06-20",
				DueDate:        "2026-07-10",
			}
			_, err := domain.NewInvoice(input)
			if tt.want {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestInvoice_AddTransactionAmount(t *testing.T) {
	t.Run("valid amount", func(t *testing.T) {
		inv := &domain.Invoice{Status: domain.InvoiceStatusOpen, TotalAmount: 0}
		err := inv.AddTransactionAmount(1000)
		require.NoError(t, err)
		assert.Equal(t, int64(1000), inv.TotalAmount)
	})

	t.Run("closed invoice returns error", func(t *testing.T) {
		inv := &domain.Invoice{Status: domain.InvoiceStatusClosed}
		err := inv.AddTransactionAmount(1000)
		assert.ErrorIs(t, err, domain.ErrInvoiceNotOpen)
	})

	t.Run("paid invoice returns error", func(t *testing.T) {
		inv := &domain.Invoice{Status: domain.InvoiceStatusPaid}
		err := inv.AddTransactionAmount(1000)
		assert.ErrorIs(t, err, domain.ErrInvoiceNotOpen)
	})

	t.Run("zero amount", func(t *testing.T) {
		inv := &domain.Invoice{Status: domain.InvoiceStatusOpen}
		err := inv.AddTransactionAmount(0)
		assert.ErrorIs(t, err, domain.ErrNegativeAmount)
	})

	t.Run("negative amount", func(t *testing.T) {
		inv := &domain.Invoice{Status: domain.InvoiceStatusOpen}
		err := inv.AddTransactionAmount(-500)
		assert.ErrorIs(t, err, domain.ErrNegativeAmount)
	})

	t.Run("accumulates multiple transactions", func(t *testing.T) {
		inv := &domain.Invoice{Status: domain.InvoiceStatusOpen, TotalAmount: 0}
		_ = inv.AddTransactionAmount(1000)
		_ = inv.AddTransactionAmount(2000)
		_ = inv.AddTransactionAmount(500)
		assert.Equal(t, int64(3500), inv.TotalAmount)
	})
}

func TestInvoice_Pay(t *testing.T) {
	t.Run("partial payment", func(t *testing.T) {
		inv := &domain.Invoice{
			Status:      domain.InvoiceStatusOpen,
			TotalAmount: 10000,
			PaidAmount:  0,
		}
		err := inv.Pay(3000)
		require.NoError(t, err)
		assert.Equal(t, int64(3000), inv.PaidAmount)
		assert.Equal(t, domain.InvoiceStatusOpen, inv.Status)
	})

	t.Run("full payment marks paid", func(t *testing.T) {
		inv := &domain.Invoice{
			Status:      domain.InvoiceStatusOpen,
			TotalAmount: 10000,
			PaidAmount:  0,
		}
		err := inv.Pay(10000)
		require.NoError(t, err)
		assert.Equal(t, int64(10000), inv.PaidAmount)
		assert.Equal(t, domain.InvoiceStatusPaid, inv.Status)
	})

	t.Run("already paid returns error", func(t *testing.T) {
		inv := &domain.Invoice{
			Status:      domain.InvoiceStatusPaid,
			TotalAmount: 10000,
			PaidAmount:  10000,
		}
		err := inv.Pay(1000)
		assert.ErrorIs(t, err, domain.ErrInvoiceAlreadyPaid)
	})

	t.Run("payment exceeds remaining amount", func(t *testing.T) {
		inv := &domain.Invoice{
			Status:      domain.InvoiceStatusOpen,
			TotalAmount: 10000,
			PaidAmount:  3000,
		}
		err := inv.Pay(8000)
		assert.ErrorIs(t, err, domain.ErrPaymentExceedsAmount)
	})

	t.Run("zero amount", func(t *testing.T) {
		inv := &domain.Invoice{
			Status:      domain.InvoiceStatusOpen,
			TotalAmount: 10000,
		}
		err := inv.Pay(0)
		assert.ErrorIs(t, err, domain.ErrNegativeAmount)
	})

	t.Run("negative amount", func(t *testing.T) {
		inv := &domain.Invoice{
			Status:      domain.InvoiceStatusOpen,
			TotalAmount: 10000,
		}
		err := inv.Pay(-100)
		assert.ErrorIs(t, err, domain.ErrNegativeAmount)
	})
}

func TestInvoice_TransitionStatus(t *testing.T) {
	t.Run("open to closed", func(t *testing.T) {
		inv := &domain.Invoice{Status: domain.InvoiceStatusOpen}
		err := inv.TransitionStatus(domain.InvoiceStatusClosed)
		require.NoError(t, err)
		assert.Equal(t, domain.InvoiceStatusClosed, inv.Status)
	})

	t.Run("open to overdue", func(t *testing.T) {
		inv := &domain.Invoice{Status: domain.InvoiceStatusOpen}
		err := inv.TransitionStatus(domain.InvoiceStatusOverdue)
		require.NoError(t, err)
		assert.Equal(t, domain.InvoiceStatusOverdue, inv.Status)
	})

	t.Run("closed to overdue", func(t *testing.T) {
		inv := &domain.Invoice{Status: domain.InvoiceStatusClosed}
		err := inv.TransitionStatus(domain.InvoiceStatusOverdue)
		require.NoError(t, err)
		assert.Equal(t, domain.InvoiceStatusOverdue, inv.Status)
	})

	t.Run("closed to paid", func(t *testing.T) {
		inv := &domain.Invoice{Status: domain.InvoiceStatusClosed}
		err := inv.TransitionStatus(domain.InvoiceStatusPaid)
		require.NoError(t, err)
		assert.Equal(t, domain.InvoiceStatusPaid, inv.Status)
	})

	t.Run("overdue to closed", func(t *testing.T) {
		inv := &domain.Invoice{Status: domain.InvoiceStatusOverdue}
		err := inv.TransitionStatus(domain.InvoiceStatusClosed)
		require.NoError(t, err)
		assert.Equal(t, domain.InvoiceStatusClosed, inv.Status)
	})

	t.Run("overdue to paid", func(t *testing.T) {
		inv := &domain.Invoice{Status: domain.InvoiceStatusOverdue}
		err := inv.TransitionStatus(domain.InvoiceStatusPaid)
		require.NoError(t, err)
		assert.Equal(t, domain.InvoiceStatusPaid, inv.Status)
	})

	t.Run("paid to any", func(t *testing.T) {
		inv := &domain.Invoice{Status: domain.InvoiceStatusPaid}
		err := inv.TransitionStatus(domain.InvoiceStatusOpen)
		assert.ErrorIs(t, err, domain.ErrStatusTransition)
	})

	t.Run("open to paid (invalid)", func(t *testing.T) {
		inv := &domain.Invoice{Status: domain.InvoiceStatusOpen}
		err := inv.TransitionStatus(domain.InvoiceStatusPaid)
		assert.ErrorIs(t, err, domain.ErrStatusTransition)
	})

	t.Run("open to open (invalid)", func(t *testing.T) {
		inv := &domain.Invoice{Status: domain.InvoiceStatusOpen}
		err := inv.TransitionStatus(domain.InvoiceStatusOpen)
		assert.ErrorIs(t, err, domain.ErrStatusTransition)
	})

	t.Run("invalid status value", func(t *testing.T) {
		inv := &domain.Invoice{Status: domain.InvoiceStatusOpen}
		err := inv.TransitionStatus("unknown")
		assert.ErrorIs(t, err, domain.ErrInvalidStatus)
	})
}

func TestNewInvoiceTransaction(t *testing.T) {
	validInput := func() domain.CreateTransactionInput {
		return domain.CreateTransactionInput{
			InvoiceID:       "inv-1",
			UserID:          "user-1",
			Description:     "Restaurant",
			Amount:          5000,
			Category:        "food",
			TransactionDate: "2026-06-15",
			Installments:    1,
		}
	}

	t.Run("success", func(t *testing.T) {
		input := validInput()
		tx, err := domain.NewInvoiceTransaction(input)
		require.NoError(t, err)
		require.NotNil(t, tx)
		assert.Equal(t, input.InvoiceID, tx.InvoiceID)
		assert.Equal(t, input.UserID, tx.UserID)
		assert.Equal(t, input.Description, tx.Description)
		assert.Equal(t, input.Amount, tx.Amount)
		assert.Equal(t, input.Category, tx.Category)
		assert.Equal(t, input.TransactionDate, tx.TransactionDate)
		assert.Equal(t, int32(1), tx.Installments)
		assert.False(t, tx.CreatedAt.IsZero())
	})

	t.Run("missing invoice id", func(t *testing.T) {
		input := validInput()
		input.InvoiceID = ""
		tx, err := domain.NewInvoiceTransaction(input)
		assert.Nil(t, tx)
		assert.ErrorIs(t, err, domain.ErrMissingField)
	})

	t.Run("missing user id", func(t *testing.T) {
		input := validInput()
		input.UserID = ""
		tx, err := domain.NewInvoiceTransaction(input)
		assert.Nil(t, tx)
		assert.ErrorIs(t, err, domain.ErrMissingField)
	})

	t.Run("missing description", func(t *testing.T) {
		input := validInput()
		input.Description = ""
		tx, err := domain.NewInvoiceTransaction(input)
		assert.Nil(t, tx)
		assert.ErrorIs(t, err, domain.ErrMissingField)
	})

	t.Run("zero amount", func(t *testing.T) {
		input := validInput()
		input.Amount = 0
		tx, err := domain.NewInvoiceTransaction(input)
		assert.Nil(t, tx)
		assert.ErrorIs(t, err, domain.ErrNegativeAmount)
	})

	t.Run("negative amount", func(t *testing.T) {
		input := validInput()
		input.Amount = -100
		tx, err := domain.NewInvoiceTransaction(input)
		assert.Nil(t, tx)
		assert.ErrorIs(t, err, domain.ErrNegativeAmount)
	})

	t.Run("missing transaction date", func(t *testing.T) {
		input := validInput()
		input.TransactionDate = ""
		tx, err := domain.NewInvoiceTransaction(input)
		assert.Nil(t, tx)
		assert.ErrorIs(t, err, domain.ErrMissingField)
	})

	t.Run("default category when empty", func(t *testing.T) {
		input := validInput()
		input.Category = ""
		tx, err := domain.NewInvoiceTransaction(input)
		require.NoError(t, err)
		assert.Equal(t, "other", tx.Category)
	})

	t.Run("default installments when zero", func(t *testing.T) {
		input := validInput()
		input.Installments = 0
		tx, err := domain.NewInvoiceTransaction(input)
		require.NoError(t, err)
		assert.Equal(t, int32(1), tx.Installments)
	})

	t.Run("default installments when negative", func(t *testing.T) {
		input := validInput()
		input.Installments = -1
		tx, err := domain.NewInvoiceTransaction(input)
		require.NoError(t, err)
		assert.Equal(t, int32(1), tx.Installments)
	})

	t.Run("table driven fields", func(t *testing.T) {
		tests := []struct {
			name   string
			modify func(*domain.CreateTransactionInput)
			check  func(*testing.T, *domain.CreateTransactionInput, *domain.InvoiceTransaction)
		}{
			{
				name: "custom category",
				modify: func(input *domain.CreateTransactionInput) {
					input.Category = "transport"
				},
				check: func(t *testing.T, input *domain.CreateTransactionInput, tx *domain.InvoiceTransaction) {
					assert.Equal(t, "transport", tx.Category)
				},
			},
			{
				name: "multiple installments",
				modify: func(input *domain.CreateTransactionInput) {
					input.Installments = 12
				},
				check: func(t *testing.T, input *domain.CreateTransactionInput, tx *domain.InvoiceTransaction) {
					assert.Equal(t, int32(12), tx.Installments)
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				input := validInput()
				tt.modify(&input)
				tx, err := domain.NewInvoiceTransaction(input)
				require.NoError(t, err)
				tt.check(t, &input, tx)
			})
		}
	})
}

func Example_cardBrands() {
	fmt.Println("Valid card brands:", len(domain.ValidCardBrands()))
	// Output: Valid card brands: 7
}

func Example_cardTypes() {
	fmt.Println("Valid card types:", len(domain.ValidCardTypes()))
	// Output: Valid card types: 3
}
