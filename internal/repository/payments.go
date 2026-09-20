package repository

import (
	"sync"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/models"
)

type PaymentsRepository struct {
	mu       sync.RWMutex
	payments map[string]models.PostPaymentResponse
}

func NewPaymentsRepository() *PaymentsRepository {
	return &PaymentsRepository{
		payments: make(map[string]models.PostPaymentResponse),
	}
}

func (ps *PaymentsRepository) GetPayment(id string) *models.PostPaymentResponse {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	p, ok := ps.payments[id]
	if !ok {
		return nil
	}
	return &p
}

func (ps *PaymentsRepository) AddPayment(payment models.PostPaymentResponse) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.payments[payment.Id] = payment
}

// All returns every stored payment. Used by tests to assert on the
// total count — not needed by the handler itself.
func (ps *PaymentsRepository) All() []models.PostPaymentResponse {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	out := make([]models.PostPaymentResponse, 0, len(ps.payments))
	for _, p := range ps.payments {
		out = append(out, p)
	}
	return out
}
