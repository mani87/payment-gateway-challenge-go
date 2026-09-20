package repository

import (
	"sync"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/models"
)

type PaymentsRepository struct {
	mu       sync.RWMutex
	payments map[string]models.PostPaymentResponse

	// This can be a col in real DB and indexed as well!
	idempotency map[string]models.PostPaymentResponse
}

func NewPaymentsRepository() *PaymentsRepository {
	return &PaymentsRepository{
		payments:    make(map[string]models.PostPaymentResponse),
		idempotency: make(map[string]models.PostPaymentResponse),
	}
}

// nil if key is empty or unseen
func (ps *PaymentsRepository) GetByIdempotencyKey(key string) *models.PostPaymentResponse {
	if key == "" {
		return nil
	}

	ps.mu.RLock()
	defer ps.mu.RUnlock()

	p, ok := ps.idempotency[key]
	if !ok {
		return nil
	}
	return &p
}

// save record to corresponding idempotent key
func (ps *PaymentsRepository) AddIdempotencyKey(key string, payment models.PostPaymentResponse) {
	if key == "" {
		return
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.idempotency[key] = payment
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
