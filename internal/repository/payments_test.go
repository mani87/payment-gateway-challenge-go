package repository

import (
	"fmt"
	"sync"
	"testing"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestPaymentsRepository_AddAndGet(t *testing.T) {
	repo := NewPaymentsRepository()

	payment := models.PostPaymentResponse{
		Id:            "test-id-1",
		PaymentStatus: "Authorized",
	}
	repo.AddPayment(payment)

	got := repo.GetPayment("test-id-1")
	assert.NotNil(t, got)
	assert.Equal(t, "Authorized", got.PaymentStatus)
}

func TestPaymentsRepository_GetPayment_NotFound(t *testing.T) {
	repo := NewPaymentsRepository()

	got := repo.GetPayment("does-not-exist")
	assert.Nil(t, got)
}

func TestPaymentsRepository_AddPayment_Overwrite(t *testing.T) {
	repo := NewPaymentsRepository()

	repo.AddPayment(models.PostPaymentResponse{Id: "dup-id", PaymentStatus: "Declined"})
	repo.AddPayment(models.PostPaymentResponse{Id: "dup-id", PaymentStatus: "Authorized"})

	got := repo.GetPayment("dup-id")
	assert.Equal(t, "Authorized", got.PaymentStatus)
}

func TestPaymentsRepository_All_Empty(t *testing.T) {
	repo := NewPaymentsRepository()

	assert.Empty(t, repo.All())
}

func TestPaymentsRepository_All_ReturnsEverythingStored(t *testing.T) {
	repo := NewPaymentsRepository()

	repo.AddPayment(models.PostPaymentResponse{Id: "id-1", PaymentStatus: "Authorized"})
	repo.AddPayment(models.PostPaymentResponse{Id: "id-2", PaymentStatus: "Declined"})

	assert.Len(t, repo.All(), 2)
}

// TestPaymentsRepository_ConcurrentAccess proves the map+mutex is
// safe under concurrent load — every HTTP request runs on its own
// goroutine, so this is the scenario that would data-race with the
// original slice-based implementation.
func TestPaymentsRepository_ConcurrentAccess(t *testing.T) {
	repo := NewPaymentsRepository()
	var wg sync.WaitGroup

	const n = 100
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("payment-%d", i)
			repo.AddPayment(models.PostPaymentResponse{Id: id, PaymentStatus: "Authorized"})
			repo.GetPayment(id)
		}(i)
	}
	wg.Wait()

	assert.Len(t, repo.All(), n)
}

// TestPaymentsRepository_ConcurrentReadWriteSameID hammers a single
// key from many goroutines at once — reads and writes racing on the
// exact same map entry, the sharpest test of the mutex.
func TestPaymentsRepository_ConcurrentReadWriteSameID(t *testing.T) {
	repo := NewPaymentsRepository()
	repo.AddPayment(models.PostPaymentResponse{Id: "shared-id", PaymentStatus: "Authorized"})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			repo.GetPayment("shared-id")
		}()
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			repo.AddPayment(models.PostPaymentResponse{Id: "shared-id", PaymentStatus: fmt.Sprintf("status-%d", i)})
		}(i)
	}
	wg.Wait()

	got := repo.GetPayment("shared-id")
	assert.NotNil(t, got)
}
