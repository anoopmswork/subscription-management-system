package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"subscription-management-system/internal/customer"
	domain "subscription-management-system/internal/domain/customer"
)

func TestCreateCustomerUnauthorizedWithoutActorHeaders(t *testing.T) {
	app := setupTestApp()

	body := `{"external_ref":"cust-1","legal_name":"Acme Inc","display_name":"Acme","primary_contact":{"name":"Jane","email":"jane@acme.com","role":"billing"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/customers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app test failed: %v", err)
	}

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestCreateCustomerSuccess(t *testing.T) {
	app := setupTestApp()

	body := `{"external_ref":"cust-1","legal_name":"Acme Inc","display_name":"Acme","segment":"enterprise","primary_contact":{"name":"Jane","email":"jane@acme.com","role":"billing"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/customers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Actor-ID", "actor-1")
	req.Header.Set("X-Actor-Role", "admin")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app test failed: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	var payload domain.Aggregate
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if payload.Customer.ExternalRef != "cust-1" {
		t.Fatalf("expected customer external_ref to be cust-1, got %s", payload.Customer.ExternalRef)
	}
}

func setupTestApp() *fiber.App {
	repo := &stubRepository{}
	svc := customer.NewService(repo)
	h := NewHandler(svc)

	app := fiber.New()
	h.Register(app)
	return app
}

type stubRepository struct{}

func (s *stubRepository) CreateCustomer(_ context.Context, params customer.CreateCustomerParams) (domain.Aggregate, error) {
	now := time.Now().UTC()
	customerID := "fbc3e6f7-5d85-4a78-aad6-5c420f925367"

	return domain.Aggregate{
		Customer: domain.Account{
			ID:                     customerID,
			ExternalRef:            params.ExternalRef,
			LegalName:              params.LegalName,
			DisplayName:            params.DisplayName,
			Segment:                params.Segment,
			Status:                 domain.StatusProspect,
			OpenInvoiceBalance:     0,
			HasActiveSubscriptions: false,
			CreatedAt:              now,
			UpdatedAt:              now,
		},
		Preference: domain.Preference{
			CustomerID:             customerID,
			Locale:                 params.Preference.Locale,
			Timezone:               params.Preference.Timezone,
			DefaultCurrency:        params.Preference.DefaultCurrency,
			InvoiceDeliveryChannel: params.Preference.InvoiceDeliveryChannel,
			CreatedAt:              now,
			UpdatedAt:              now,
		},
		Contacts: []domain.Contact{
			{
				ID:         "4710f201-2997-4501-aec3-f4a465db6a43",
				CustomerID: customerID,
				Name:       params.PrimaryContact.Name,
				Email:      params.PrimaryContact.Email,
				Role:       params.PrimaryContact.Role,
				IsPrimary:  true,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		},
		Users: []domain.CustomerUser{},
	}, nil
}

func (s *stubRepository) GetCustomer(_ context.Context, _ string) (domain.Aggregate, error) {
	return domain.Aggregate{}, customer.ErrNotFound
}

func (s *stubRepository) UpdateCustomer(_ context.Context, _ string, _ customer.UpdateCustomerParams) (domain.Aggregate, error) {
	return domain.Aggregate{}, nil
}

func (s *stubRepository) ChangeStatus(_ context.Context, _ string, _ customer.ChangeStatusParams) (domain.Account, error) {
	return domain.Account{}, nil
}

func (s *stubRepository) AddContact(_ context.Context, _ string, _ customer.AddContactParams) (domain.Contact, error) {
	return domain.Contact{}, nil
}

func (s *stubRepository) UpdateContact(_ context.Context, _ string, _ string, _ customer.UpdateContactParams) (domain.Contact, error) {
	return domain.Contact{}, nil
}

func (s *stubRepository) InviteUser(_ context.Context, _ string, _ customer.InviteUserParams) (domain.CustomerUser, error) {
	return domain.CustomerUser{}, nil
}
