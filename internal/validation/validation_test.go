package validation

import (
	"testing"
	"time"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/models"
)

func validPayment() models.PostPaymentRequest {
	return models.PostPaymentRequest{
		CardNumber:  "11223344556677",
		ExpiryMonth: 12,
		ExpiryYear:  2030,
		Currency:    "GBP",
		Amount:      1200,
		Cvv:         "123",
	}
}

func Test_ValidatePayment_Valid(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	errs := ValidatePayment(validPayment(), now)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got %+v", errs)
	}
}

func TestValidatePayment_ExpiryMonth(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name  string
		month int
	}{
		{"zero", 0},
		{"negative", -1},
		{"thirteen", 13},
		{"way too high", 99},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := validPayment()
			p.ExpiryMonth = tc.month
			errs := ValidatePayment(p, now)
			if !hasFieldError(errs, "expiry_month") {
				t.Errorf("expected expiry_month error, got %+v", errs)
			}
		})
	}
}

func TestValidatePayment_ExpiryYear_Boundary(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name      string
		month     int
		year      int
		wantError bool
	}{
		{"expires this month - valid", 6, 2026, false},
		{"expires next month - valid", 7, 2026, false},
		{"expired last month - invalid", 5, 2026, true},
		{"expired last year - invalid", 12, 2025, true},
		{"far future - valid", 1, 2030, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := validPayment()
			p.ExpiryMonth = tc.month
			p.ExpiryYear = tc.year
			errs := ValidatePayment(p, now)
			got := hasFieldError(errs, "expiry_year")
			if got != tc.wantError {
				t.Errorf("expiry_year error = %v, want %v (errs: %+v)", got, tc.wantError, errs)
			}
		})
	}
}

func TestValidatePayment_Currency(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name     string
		currency string
	}{
		{"too short", "GB"},
		{"too long", "GBPX"},
		{"unsupported", "JPY"},
		{"lowercase not accepted", "gbp"},
		{"empty", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := validPayment()
			p.Currency = tc.currency
			errs := ValidatePayment(p, now)
			if !hasFieldError(errs, "currency") {
				t.Errorf("expected currency error, got %+v", errs)
			}
		})
	}
}

func TestValidatePayment_Currency_AllSupported(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	for _, cur := range []string{"GBP", "USD", "INR"} {
		t.Run(cur, func(t *testing.T) {
			p := validPayment()
			p.Currency = cur
			errs := ValidatePayment(p, now)
			if hasFieldError(errs, "currency") {
				t.Errorf("expected %s to be accepted, got %+v", cur, errs)
			}
		})
	}
}

func TestValidatePayment_Amount(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name   string
		amount int
	}{
		{"zero", 0},
		{"negative", -100},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := validPayment()
			p.Amount = tc.amount
			errs := ValidatePayment(p, now)
			if !hasFieldError(errs, "amount") {
				t.Errorf("expected amount error, got %+v", errs)
			}
		})
	}
}

func Test_ValidatePayment_InvalidCard(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		card string
	}{
		{"too short", "1234"},
		{"too long", "1234567895432113322223344"},
		{"non numeric", "123ddfs345672d"},
		{"empty", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := validPayment()
			p.CardNumber = tc.card
			errs := ValidatePayment(p, now)
			if !hasFieldError(errs, "card_number") {
				t.Errorf("expected card_number error, got %+v", errs)
			}
		})
	}

}

func Test_ValidatePayment_InvalidCvv(t *testing.T) {
	now := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		cvv  string
	}{
		{"too short", "1"},
		{"too long", "12345"},
		{"non numeric", "asb"},
		{"empty", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := validPayment()
			p.Cvv = tc.cvv
			errs := ValidatePayment(p, now)
			if !hasFieldError(errs, "cvv") {
				t.Errorf("expected cvv error, got %+v", errs)
			}
		})
	}
}

func hasFieldError(errs []models.FieldError, field string) bool {
	for _, e := range errs {
		if e.Field == field {
			return true
		}
	}
	return false
}
