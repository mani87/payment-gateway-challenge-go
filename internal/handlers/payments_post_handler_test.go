package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/bank"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/models"
	"github.com/cko-recruitment/payment-gateway-challenge-go/internal/repository"
	"github.com/stretchr/testify/assert"
)

type fakeBank struct {
	response *bank.BankPaymentResponse
	err      error
	calls    int
}

func (fb *fakeBank) ProcessPayment(context context.Context, req models.PostPaymentRequest) (*bank.BankPaymentResponse, error) {
	fb.calls++
	return fb.response, fb.err
}

func validRequestBody() models.PostPaymentRequest {
	return models.PostPaymentRequest{
		CardNumber:  "2222405343248877",
		ExpiryMonth: 6,
		ExpiryYear:  2030,
		Currency:    "GBP",
		Amount:      100,
		Cvv:         "123",
	}
}

func doPost(t *testing.T, h *PaymentsHandler, body models.PostPaymentRequest) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	assert.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/payments", bytes.NewReader(b))
	rr := httptest.NewRecorder()
	h.PostHandler()(rr, req)
	return rr
}

func TestPostHandler_Authorized(t *testing.T) {
	fb := &fakeBank{response: &bank.BankPaymentResponse{Authorized: true}}
	repo := repository.NewPaymentsRepository()
	h := NewPaymentsHandler(repo, fb)

	rr := doPost(t, h, validRequestBody())

	assert.Equal(t, rr.Code, 201)

	var resp models.PostPaymentResponse
	assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, resp.PaymentStatus, "Authorized")
	assert.Equal(t, resp.CardNumberLastFour, "8877")

	assert.NotNil(t, repo.GetPayment(resp.Id), "payment is persisted")
}

func TestPostHandler_Declined(t *testing.T) {
	fb := &fakeBank{response: &bank.BankPaymentResponse{Authorized: false}}
	repo := repository.NewPaymentsRepository()
	h := NewPaymentsHandler(repo, fb)

	rr := doPost(t, h, validRequestBody())

	assert.Equal(t, rr.Code, 201)

	var resp models.PostPaymentResponse
	assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, resp.PaymentStatus, "Declined")
	assert.Equal(t, resp.CardNumberLastFour, "8877")

	assert.NotNil(t, repo.GetPayment(resp.Id), "payment is persisted")
}

func TestPostHandler_BankUnavailable(t *testing.T) {
	fb := &fakeBank{err: bank.ErrBankUnavailable}
	repo := repository.NewPaymentsRepository()
	h := NewPaymentsHandler(repo, fb)

	rr := doPost(t, h, validRequestBody())

	assert.Equal(t, 502, rr.Code)
}

func TestPostHandler_BadBankRequest(t *testing.T) {
	fb := &fakeBank{err: bank.ErrBadBankRequest}
	repo := repository.NewPaymentsRepository()
	h := NewPaymentsHandler(repo, fb)

	rr := doPost(t, h, validRequestBody())

	assert.Equal(t, 502, rr.Code)
}

func TestPostHandler_InvalidRequest_NeverCallsBank(t *testing.T) {
	fb := &fakeBank{response: &bank.BankPaymentResponse{Authorized: true}}
	repo := repository.NewPaymentsRepository()
	h := NewPaymentsHandler(repo, fb)

	bad := validRequestBody()
	bad.Cvv = "12" // too short

	rr := doPost(t, h, bad)

	assert.Equal(t, 400, rr.Code)

	var resp models.RejectedResponse
	assert.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "Rejected", resp.PaymentStatus)
	assert.NotEmpty(t, resp.Errors)

	assert.Zero(t, fb.calls, "bank must not be called for an invalid request")
}

func TestPostHandler_ResponseNeverLeaksFullCardOrCvv(t *testing.T) {
	fb := &fakeBank{response: &bank.BankPaymentResponse{Authorized: true}}
	repo := repository.NewPaymentsRepository()
	h := NewPaymentsHandler(repo, fb)

	body := validRequestBody()
	rr := doPost(t, h, body)

	assert.NotContains(t, rr.Body.String(), body.CardNumber)
	assert.NotContains(t, rr.Body.String(), body.Cvv)
}

func TestPostHandler_IdempotencyKey_ReplaysOriginalResponse(t *testing.T) {
	fb := &fakeBank{response: &bank.BankPaymentResponse{Authorized: true}}
	repo := repository.NewPaymentsRepository()
	h := NewPaymentsHandler(repo, fb)

	body, err := json.Marshal(validRequestBody())
	assert.NoError(t, err)

	doWithKey := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/api/payments", bytes.NewReader(body))
		req.Header.Set("Idempotency-Key", "test-key-123")
		rr := httptest.NewRecorder()
		h.PostHandler()(rr, req)
		return rr
	}

	first := doWithKey()
	assert.Equal(t, 201, first.Code)

	var firstResp models.PostPaymentResponse
	assert.NoError(t, json.Unmarshal(first.Body.Bytes(), &firstResp))

	second := doWithKey()
	assert.Equal(t, 200, second.Code, "replay should not report a new creation")

	var secondResp models.PostPaymentResponse
	assert.NoError(t, json.Unmarshal(second.Body.Bytes(), &secondResp))

	assert.Equal(t, firstResp.Id, secondResp.Id, "same key must return the same payment, not a new one")
	assert.Equal(t, 1, fb.calls, "bank must only be charged once for a replayed key")
	assert.Len(t, repo.All(), 1, "only one payment should ever be persisted for this key")
}
