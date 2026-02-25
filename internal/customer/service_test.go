package customer

import (
	"context"
	"testing"
	"time"

	domain "subscription-management-system/internal/domain/customer"
)

func TestCreateCustomerSuccess(t *testing.T) {
	repo := &fakeRepository{aggregate: seedAggregate()}
	svc := NewService(repo)

	result, err := svc.CreateCustomer(context.Background(), domain.Actor{ID: "actor-1", Role: domain.RoleAdmin}, CreateCustomerInput{
		ExternalRef: "cust-ext-1",
		LegalName:   "Acme Inc",
		DisplayName: "Acme",
		Segment:     "enterprise",
		Preference: CreatePreferenceInput{
			Locale:                 "en-US",
			Timezone:               "UTC",
			DefaultCurrency:        "USD",
			InvoiceDeliveryChannel: "email",
		},
		PrimaryContact: CreateContactInput{
			Name:      "Jane Doe",
			Email:     "jane@acme.com",
			Role:      "billing",
			IsPrimary: true,
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Customer.ExternalRef != "cust-ext-1" {
		t.Fatalf("expected external_ref to be mapped")
	}

	if !repo.createCalled {
		t.Fatalf("expected repository create to be called")
	}
}

func TestCreateCustomerForbiddenForViewer(t *testing.T) {
	svc := NewService(&fakeRepository{aggregate: seedAggregate()})

	_, err := svc.CreateCustomer(context.Background(), domain.Actor{ID: "actor-1", Role: domain.RoleViewer}, CreateCustomerInput{
		ExternalRef: "cust-ext-1",
		LegalName:   "Acme Inc",
		DisplayName: "Acme",
		PrimaryContact: CreateContactInput{
			Name:  "Jane Doe",
			Email: "jane@acme.com",
			Role:  "billing",
		},
	})

	if err == nil {
		t.Fatalf("expected forbidden error")
	}

	if err != ErrForbidden {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestChangeStatusRejectsInvalidTransition(t *testing.T) {
	agg := seedAggregate()
	agg.Customer.Status = domain.StatusProspect

	svc := NewService(&fakeRepository{aggregate: agg})

	_, err := svc.ChangeStatus(context.Background(), domain.Actor{ID: "actor-1", Role: domain.RoleAdmin}, agg.Customer.ID, ChangeStatusInput{
		Status: domain.StatusArchived,
		Reason: "cleanup",
	})
	if err == nil {
		t.Fatalf("expected error")
	}

	if !containsValidation(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestChangeStatusArchiveBlockedByBalance(t *testing.T) {
	agg := seedAggregate()
	agg.Customer.Status = domain.StatusActive
	agg.Customer.OpenInvoiceBalance = 100

	svc := NewService(&fakeRepository{aggregate: agg})

	_, err := svc.ChangeStatus(context.Background(), domain.Actor{ID: "actor-1", Role: domain.RoleAdmin}, agg.Customer.ID, ChangeStatusInput{
		Status: domain.StatusArchived,
		Reason: "close account",
	})
	if err == nil {
		t.Fatalf("expected conflict error")
	}

	if !containsConflict(err) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestUpdateContactMissingContact(t *testing.T) {
	agg := seedAggregate()
	svc := NewService(&fakeRepository{aggregate: agg})

	name := "new"
	_, err := svc.UpdateContact(context.Background(), domain.Actor{ID: "actor-1", Role: domain.RoleAdmin}, agg.Customer.ID, "33f5f31f-9444-4c64-9a8b-44fe87d5a591", UpdateContactInput{
		Name: &name,
	})
	if err == nil {
		t.Fatalf("expected not found error")
	}

	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateCustomerAppliesDefaultsAndNormalization(t *testing.T) {
	repo := &fakeRepository{aggregate: seedAggregate()}
	svc := NewService(repo)

	parentCustomerID := "  208de514-5854-4a82-91c6-f3c63f90d2f1  "
	phone := "   "

	_, err := svc.CreateCustomer(context.Background(), domain.Actor{ID: "actor-1", Role: domain.RoleOwner}, CreateCustomerInput{
		ExternalRef:      "  cust-ext-2  ",
		LegalName:        "  Acme Holdings  ",
		DisplayName:      "  Acme  ",
		Segment:          "   ",
		ParentCustomerID: &parentCustomerID,
		Preference:       CreatePreferenceInput{},
		PrimaryContact: CreateContactInput{
			Name:      "  Jane Doe  ",
			Email:     "  JANE@ACME.COM  ",
			Phone:     &phone,
			Role:      "  billing  ",
			IsPrimary: false,
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.createParams.ExternalRef != "cust-ext-2" {
		t.Fatalf("expected external_ref to be trimmed, got %q", repo.createParams.ExternalRef)
	}

	if repo.createParams.LegalName != "Acme Holdings" || repo.createParams.DisplayName != "Acme" {
		t.Fatalf("expected names to be trimmed")
	}

	if repo.createParams.Segment != "default" {
		t.Fatalf("expected default segment, got %q", repo.createParams.Segment)
	}

	if repo.createParams.ParentCustomerID == nil || *repo.createParams.ParentCustomerID != "208de514-5854-4a82-91c6-f3c63f90d2f1" {
		t.Fatalf("expected trimmed parent_customer_id")
	}

	if repo.createParams.Preference.Locale != "en-US" || repo.createParams.Preference.Timezone != "UTC" || repo.createParams.Preference.DefaultCurrency != "USD" || repo.createParams.Preference.InvoiceDeliveryChannel != "email" {
		t.Fatalf("expected default preference values to be applied")
	}

	if repo.createParams.PrimaryContact.Email != "jane@acme.com" {
		t.Fatalf("expected lower-cased email, got %q", repo.createParams.PrimaryContact.Email)
	}

	if repo.createParams.PrimaryContact.Phone != nil {
		t.Fatalf("expected blank phone to be normalized to nil")
	}

	if !repo.createParams.PrimaryContact.IsPrimary {
		t.Fatalf("expected primary contact to be forced as primary")
	}
}

func TestCreateCustomerRejectsInvalidParentCustomerID(t *testing.T) {
	repo := &fakeRepository{aggregate: seedAggregate()}
	svc := NewService(repo)

	parentCustomerID := "not-a-uuid"
	_, err := svc.CreateCustomer(context.Background(), domain.Actor{ID: "actor-1", Role: domain.RoleAdmin}, CreateCustomerInput{
		ExternalRef:      "cust-ext-2",
		LegalName:        "Acme Holdings",
		DisplayName:      "Acme",
		ParentCustomerID: &parentCustomerID,
		PrimaryContact: CreateContactInput{
			Name:  "Jane Doe",
			Email: "jane@acme.com",
			Role:  "billing",
		},
	})

	if !containsValidation(err) {
		t.Fatalf("expected validation error, got %v", err)
	}

	if repo.createCalled {
		t.Fatalf("expected repository create not to be called")
	}
}

func TestGetCustomerRejectsUnauthorizedActor(t *testing.T) {
	svc := NewService(&fakeRepository{aggregate: seedAggregate()})

	_, err := svc.GetCustomer(context.Background(), domain.Actor{ID: "", Role: domain.RoleAdmin}, "b10873e4-abd0-4210-bf7e-a2a642f2b97b")
	if err != ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestGetCustomerRejectsInvalidCustomerID(t *testing.T) {
	svc := NewService(&fakeRepository{aggregate: seedAggregate()})

	_, err := svc.GetCustomer(context.Background(), domain.Actor{ID: "actor-1", Role: domain.RoleViewer}, "invalid-id")
	if !containsValidation(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestUpdateCustomerRejectsNoFields(t *testing.T) {
	agg := seedAggregate()
	repo := &fakeRepository{aggregate: agg}
	svc := NewService(repo)

	_, err := svc.UpdateCustomer(context.Background(), domain.Actor{ID: "actor-1", Role: domain.RoleAdmin}, agg.Customer.ID, UpdateCustomerInput{})
	if !containsValidation(err) {
		t.Fatalf("expected validation error, got %v", err)
	}

	if repo.updateCalled {
		t.Fatalf("expected repository update not to be called")
	}
}

func TestUpdateCustomerSuccessNormalizesValues(t *testing.T) {
	agg := seedAggregate()
	repo := &fakeRepository{aggregate: agg}
	svc := NewService(repo)

	displayName := "  New Display Name  "
	segment := "  mid-market  "
	parentCustomerID := "  208de514-5854-4a82-91c6-f3c63f90d2f1  "
	currency := "  eur  "
	timezone := "  Europe/Berlin  "

	result, err := svc.UpdateCustomer(context.Background(), domain.Actor{ID: "actor-1", Role: domain.RoleAdmin}, agg.Customer.ID, UpdateCustomerInput{
		DisplayName:      &displayName,
		Segment:          &segment,
		ParentCustomerID: &parentCustomerID,
		Preference: &UpdatePreferenceInput{
			DefaultCurrency: &currency,
			Timezone:        &timezone,
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !repo.updateCalled {
		t.Fatalf("expected repository update to be called")
	}

	if repo.updateParams.DisplayName != "New Display Name" || repo.updateParams.Segment != "mid-market" {
		t.Fatalf("expected normalized account fields")
	}

	if repo.updateParams.ParentCustomerID == nil || *repo.updateParams.ParentCustomerID != "208de514-5854-4a82-91c6-f3c63f90d2f1" {
		t.Fatalf("expected normalized parent_customer_id")
	}

	if repo.updateParams.Preference.DefaultCurrency != "EUR" {
		t.Fatalf("expected currency to be upper-cased, got %q", repo.updateParams.Preference.DefaultCurrency)
	}

	if repo.updateParams.Preference.Timezone != "Europe/Berlin" {
		t.Fatalf("expected timezone to be trimmed, got %q", repo.updateParams.Preference.Timezone)
	}

	if result.Customer.DisplayName != "New Display Name" || result.Preference.DefaultCurrency != "EUR" {
		t.Fatalf("expected updated aggregate to reflect normalized values")
	}
}

func TestUpdateCustomerClearsParentCustomerID(t *testing.T) {
	agg := seedAggregate()
	agg.Customer.ParentCustomerID = stringPtr("208de514-5854-4a82-91c6-f3c63f90d2f1")

	repo := &fakeRepository{aggregate: agg}
	svc := NewService(repo)

	parentCustomerID := "   "
	_, err := svc.UpdateCustomer(context.Background(), domain.Actor{ID: "actor-1", Role: domain.RoleAdmin}, agg.Customer.ID, UpdateCustomerInput{
		ParentCustomerID: &parentCustomerID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.updateParams.ParentCustomerID != nil {
		t.Fatalf("expected parent_customer_id to be cleared")
	}
}

func TestChangeStatusNoOpSkipsRepositoryWrite(t *testing.T) {
	agg := seedAggregate()
	agg.Customer.Status = domain.StatusActive

	repo := &fakeRepository{aggregate: agg}
	svc := NewService(repo)

	account, err := svc.ChangeStatus(context.Background(), domain.Actor{ID: "actor-1", Role: domain.RoleAdmin}, agg.Customer.ID, ChangeStatusInput{
		Status: domain.Status("  ACTIVE  "),
		Reason: "unused",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.changeCalled {
		t.Fatalf("expected repository status change not to be called for no-op")
	}

	if account.Status != domain.StatusActive {
		t.Fatalf("expected status to remain active")
	}
}

func TestChangeStatusSuccessNormalizesInput(t *testing.T) {
	agg := seedAggregate()
	agg.Customer.Status = domain.StatusActive

	repo := &fakeRepository{aggregate: agg}
	svc := NewService(repo)

	account, err := svc.ChangeStatus(context.Background(), domain.Actor{ID: "actor-1", Role: domain.RoleAdmin}, agg.Customer.ID, ChangeStatusInput{
		Status: domain.Status("  SUSPENDED  "),
		Reason: "  policy violation  ",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !repo.changeCalled {
		t.Fatalf("expected repository status change to be called")
	}

	if repo.changeParams.Status != domain.StatusSuspended {
		t.Fatalf("expected status to be normalized, got %q", repo.changeParams.Status)
	}

	if repo.changeParams.Reason != "policy violation" {
		t.Fatalf("expected reason to be trimmed, got %q", repo.changeParams.Reason)
	}

	if account.Status != domain.StatusSuspended {
		t.Fatalf("expected returned account to be suspended")
	}
}

func TestAddContactSuccessNormalizesValues(t *testing.T) {
	agg := seedAggregate()
	repo := &fakeRepository{aggregate: agg}
	svc := NewService(repo)

	phone := "   "
	contact, err := svc.AddContact(context.Background(), domain.Actor{ID: "actor-1", Role: domain.RoleAdmin}, agg.Customer.ID, CreateContactInput{
		Name:      "  Finance Team  ",
		Email:     "  BILLING@ACME.COM  ",
		Phone:     &phone,
		Role:      "  accounts  ",
		IsPrimary: false,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !repo.addContactCalled {
		t.Fatalf("expected repository add contact to be called")
	}

	if repo.addContactParams.Contact.Name != "Finance Team" {
		t.Fatalf("expected contact name to be trimmed")
	}

	if repo.addContactParams.Contact.Email != "billing@acme.com" {
		t.Fatalf("expected contact email to be normalized")
	}

	if repo.addContactParams.Contact.Phone != nil {
		t.Fatalf("expected blank phone to be nil")
	}

	if repo.addContactParams.Contact.Role != "accounts" {
		t.Fatalf("expected role to be trimmed, got %q", repo.addContactParams.Contact.Role)
	}

	if contact.Email != "billing@acme.com" {
		t.Fatalf("expected returned contact email to be normalized")
	}
}

func TestAddContactRejectsInvalidEmail(t *testing.T) {
	agg := seedAggregate()
	repo := &fakeRepository{aggregate: agg}
	svc := NewService(repo)

	_, err := svc.AddContact(context.Background(), domain.Actor{ID: "actor-1", Role: domain.RoleAdmin}, agg.Customer.ID, CreateContactInput{
		Name:  "Finance Team",
		Email: "not-an-email",
		Role:  "billing",
	})
	if !containsValidation(err) {
		t.Fatalf("expected validation error, got %v", err)
	}

	if repo.addContactCalled {
		t.Fatalf("expected repository add contact not to be called")
	}
}

func TestUpdateContactRejectsNoFields(t *testing.T) {
	agg := seedAggregate()
	repo := &fakeRepository{aggregate: agg}
	svc := NewService(repo)

	_, err := svc.UpdateContact(context.Background(), domain.Actor{ID: "actor-1", Role: domain.RoleAdmin}, agg.Customer.ID, agg.Contacts[0].ID, UpdateContactInput{})
	if !containsValidation(err) {
		t.Fatalf("expected validation error, got %v", err)
	}

	if repo.updateContactCalled {
		t.Fatalf("expected repository update contact not to be called")
	}
}

func TestUpdateContactDefaultsEmptyRoleToBilling(t *testing.T) {
	agg := seedAggregate()
	repo := &fakeRepository{aggregate: agg}
	svc := NewService(repo)

	role := "   "
	contact, err := svc.UpdateContact(context.Background(), domain.Actor{ID: "actor-1", Role: domain.RoleAdmin}, agg.Customer.ID, agg.Contacts[0].ID, UpdateContactInput{
		Role: &role,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !repo.updateContactCalled {
		t.Fatalf("expected repository update contact to be called")
	}

	if repo.updateContactParams.Contact.Role != "billing" {
		t.Fatalf("expected empty role to default to billing, got %q", repo.updateContactParams.Contact.Role)
	}

	if contact.Role != "billing" {
		t.Fatalf("expected returned contact role to be billing")
	}
}

func TestUpdateContactClearsPhone(t *testing.T) {
	agg := seedAggregate()
	agg.Contacts[0].Phone = stringPtr("+14155552671")

	repo := &fakeRepository{aggregate: agg}
	svc := NewService(repo)

	phone := "   "
	contact, err := svc.UpdateContact(context.Background(), domain.Actor{ID: "actor-1", Role: domain.RoleAdmin}, agg.Customer.ID, agg.Contacts[0].ID, UpdateContactInput{
		Phone: &phone,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.updateContactParams.Contact.Phone != nil {
		t.Fatalf("expected blank phone to clear existing value")
	}

	if contact.Phone != nil {
		t.Fatalf("expected returned contact phone to be nil")
	}
}

func TestInviteUserRejectsInvalidRole(t *testing.T) {
	agg := seedAggregate()
	repo := &fakeRepository{aggregate: agg}
	svc := NewService(repo)

	_, err := svc.InviteUser(context.Background(), domain.Actor{ID: "actor-1", Role: domain.RoleAdmin}, agg.Customer.ID, InviteUserInput{
		UserID: "user-1",
		Role:   domain.RolePlatformAdmin,
	})
	if !containsValidation(err) {
		t.Fatalf("expected validation error, got %v", err)
	}

	if repo.inviteUserCalled {
		t.Fatalf("expected repository invite user not to be called")
	}
}

func TestInviteUserSuccessNormalizesValues(t *testing.T) {
	agg := seedAggregate()
	repo := &fakeRepository{aggregate: agg}
	svc := NewService(repo)

	user, err := svc.InviteUser(context.Background(), domain.Actor{ID: "actor-1", Role: domain.RoleAdmin}, agg.Customer.ID, InviteUserInput{
		UserID: "  user-42  ",
		Role:   domain.UserRole("  ADMIN  "),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !repo.inviteUserCalled {
		t.Fatalf("expected repository invite user to be called")
	}

	if repo.inviteUserParams.UserID != "user-42" {
		t.Fatalf("expected user_id to be trimmed, got %q", repo.inviteUserParams.UserID)
	}

	if repo.inviteUserParams.Role != domain.RoleAdmin {
		t.Fatalf("expected role to be normalized, got %q", repo.inviteUserParams.Role)
	}

	if user.UserID != "user-42" || user.Role != domain.RoleAdmin {
		t.Fatalf("expected returned user to contain normalized values")
	}
}

func TestValidateActorRules(t *testing.T) {
	if err := validateMutatingActor(domain.Actor{ID: "", Role: domain.RoleAdmin}); err != ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized for missing actor id, got %v", err)
	}

	if err := validateMutatingActor(domain.Actor{ID: "actor-1", Role: domain.RoleViewer}); err != ErrForbidden {
		t.Fatalf("expected ErrForbidden for viewer mutation, got %v", err)
	}

	if err := validateReadActor(domain.Actor{ID: "actor-1", Role: domain.RoleViewer}); err != nil {
		t.Fatalf("expected viewer to be allowed for read, got %v", err)
	}

	if err := validateReadActor(domain.Actor{ID: "actor-1", Role: domain.UserRole("unknown")}); err != ErrUnauthorized {
		t.Fatalf("expected ErrUnauthorized for unknown role, got %v", err)
	}
}

func TestHelperFunctions(t *testing.T) {
	preference := normalizePreference(CreatePreferenceInput{DefaultCurrency: "eur"})
	if preference.Locale != "en-US" || preference.Timezone != "UTC" || preference.DefaultCurrency != "EUR" || preference.InvoiceDeliveryChannel != "email" {
		t.Fatalf("expected normalizePreference defaults and normalization")
	}

	if got := defaultString("   ", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback for blank value")
	}

	if got := defaultString("  value  ", "fallback"); got != "value" {
		t.Fatalf("expected trimmed non-empty value")
	}

	if trimOptional(nil) != nil {
		t.Fatalf("expected nil optional to remain nil")
	}

	if got := trimOptional(stringPtr("   ")); got != nil {
		t.Fatalf("expected blank optional string to normalize to nil")
	}

	trimmed := trimOptional(stringPtr("  hello  "))
	if trimmed == nil || *trimmed != "hello" {
		t.Fatalf("expected optional string to be trimmed")
	}

	if normalizeStatus(domain.Status("  ACTIVE  ")) != domain.StatusActive {
		t.Fatalf("expected status normalization to lowercase and trim")
	}

	if !isValidStatus(domain.StatusSuspended) || isValidStatus(domain.Status("invalid")) {
		t.Fatalf("expected status validation to match known statuses")
	}

	if !isTransitionAllowed(domain.StatusActive, domain.StatusSuspended) || isTransitionAllowed(domain.StatusProspect, domain.StatusArchived) {
		t.Fatalf("expected transition matrix to be enforced")
	}

	if normalizeRole(domain.UserRole("  ADMIN  ")) != domain.RoleAdmin {
		t.Fatalf("expected role normalization to lowercase and trim")
	}

	if !isInvitableRole(domain.RoleViewer) || isInvitableRole(domain.RolePlatformAdmin) {
		t.Fatalf("expected invitable role checks to be enforced")
	}

	if !containsValidation(wrapValidation("invalid field")) {
		t.Fatalf("expected wrapped validation error")
	}
}

func containsValidation(err error) bool {
	return err != nil && (err == ErrValidation || contains(err.Error(), ErrValidation.Error()))
}

func containsConflict(err error) bool {
	return err != nil && (err == ErrConflict || contains(err.Error(), ErrConflict.Error()))
}

func contains(input string, target string) bool {
	return len(input) >= len(target) && (input == target || (len(target) > 0 && stringContains(input, target)))
}

func stringContains(s string, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

type fakeRepository struct {
	aggregate          domain.Aggregate
	createCalled       bool
	createParams       CreateCustomerParams
	updateCalled       bool
	updateParams       UpdateCustomerParams
	changeCalled       bool
	changeParams       ChangeStatusParams
	addContactCalled   bool
	addContactParams   AddContactParams
	updateContactCalled bool
	updateContactParams UpdateContactParams
	inviteUserCalled   bool
	inviteUserParams   InviteUserParams
}

func (f *fakeRepository) CreateCustomer(_ context.Context, params CreateCustomerParams) (domain.Aggregate, error) {
	f.createCalled = true
	f.createParams = params
	f.aggregate.Customer.ExternalRef = params.ExternalRef
	f.aggregate.Customer.LegalName = params.LegalName
	f.aggregate.Customer.DisplayName = params.DisplayName
	f.aggregate.Customer.Segment = params.Segment
	f.aggregate.Preference = domain.Preference{
		CustomerID:             f.aggregate.Customer.ID,
		Locale:                 params.Preference.Locale,
		Timezone:               params.Preference.Timezone,
		DefaultCurrency:        params.Preference.DefaultCurrency,
		InvoiceDeliveryChannel: params.Preference.InvoiceDeliveryChannel,
	}
	return f.aggregate, nil
}

func (f *fakeRepository) GetCustomer(_ context.Context, _ string) (domain.Aggregate, error) {
	if f.aggregate.Customer.ID == "" {
		return domain.Aggregate{}, ErrNotFound
	}
	return f.aggregate, nil
}

func (f *fakeRepository) UpdateCustomer(_ context.Context, customerID string, params UpdateCustomerParams) (domain.Aggregate, error) {
	f.updateCalled = true
	f.updateParams = params

	if f.aggregate.Customer.ID != customerID {
		return domain.Aggregate{}, ErrNotFound
	}
	f.aggregate.Customer.DisplayName = params.DisplayName
	f.aggregate.Customer.Segment = params.Segment
	f.aggregate.Customer.ParentCustomerID = params.ParentCustomerID
	f.aggregate.Preference.Locale = params.Preference.Locale
	f.aggregate.Preference.Timezone = params.Preference.Timezone
	f.aggregate.Preference.DefaultCurrency = params.Preference.DefaultCurrency
	f.aggregate.Preference.InvoiceDeliveryChannel = params.Preference.InvoiceDeliveryChannel
	return f.aggregate, nil
}

func (f *fakeRepository) ChangeStatus(_ context.Context, customerID string, params ChangeStatusParams) (domain.Account, error) {
	f.changeCalled = true
	f.changeParams = params

	if f.aggregate.Customer.ID != customerID {
		return domain.Account{}, ErrNotFound
	}
	f.aggregate.Customer.Status = params.Status
	return f.aggregate.Customer, nil
}

func (f *fakeRepository) AddContact(_ context.Context, customerID string, params AddContactParams) (domain.Contact, error) {
	f.addContactCalled = true
	f.addContactParams = params

	if f.aggregate.Customer.ID != customerID {
		return domain.Contact{}, ErrNotFound
	}
	contact := domain.Contact{
		ID:         "8a657467-ec6d-4e84-b4f7-3f41fd1553da",
		CustomerID: customerID,
		Name:       params.Contact.Name,
		Email:      params.Contact.Email,
		Phone:      params.Contact.Phone,
		Role:       params.Contact.Role,
		IsPrimary:  params.Contact.IsPrimary,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	f.aggregate.Contacts = append(f.aggregate.Contacts, contact)
	return contact, nil
}

func (f *fakeRepository) UpdateContact(_ context.Context, customerID string, contactID string, params UpdateContactParams) (domain.Contact, error) {
	f.updateContactCalled = true
	f.updateContactParams = params

	if f.aggregate.Customer.ID != customerID {
		return domain.Contact{}, ErrNotFound
	}
	for i := range f.aggregate.Contacts {
		if f.aggregate.Contacts[i].ID == contactID {
			f.aggregate.Contacts[i].Name = params.Contact.Name
			f.aggregate.Contacts[i].Email = params.Contact.Email
			f.aggregate.Contacts[i].Phone = params.Contact.Phone
			f.aggregate.Contacts[i].Role = params.Contact.Role
			f.aggregate.Contacts[i].IsPrimary = params.Contact.IsPrimary
			f.aggregate.Contacts[i].UpdatedAt = time.Now().UTC()
			return f.aggregate.Contacts[i], nil
		}
	}
	return domain.Contact{}, ErrNotFound
}

func (f *fakeRepository) InviteUser(_ context.Context, customerID string, params InviteUserParams) (domain.CustomerUser, error) {
	f.inviteUserCalled = true
	f.inviteUserParams = params

	if f.aggregate.Customer.ID != customerID {
		return domain.CustomerUser{}, ErrNotFound
	}
	user := domain.CustomerUser{
		ID:         "ac9428f9-b8bb-43ea-a05f-37e9a21fc5c0",
		CustomerID: customerID,
		UserID:     params.UserID,
		Role:       params.Role,
		Status:     "invited",
		InvitedAt:  time.Now().UTC(),
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	f.aggregate.Users = append(f.aggregate.Users, user)
	return user, nil
}

func seedAggregate() domain.Aggregate {
	now := time.Now().UTC()
	customerID := "b10873e4-abd0-4210-bf7e-a2a642f2b97b"
	contactID := "9fb8ecfa-6e2d-4655-91c6-cf1547af129c"

	return domain.Aggregate{
		Customer: domain.Account{
			ID:                     customerID,
			ExternalRef:            "seed-ext",
			LegalName:              "Seed Legal Name",
			DisplayName:            "Seed",
			Segment:                "enterprise",
			Status:                 domain.StatusActive,
			OpenInvoiceBalance:     0,
			HasActiveSubscriptions: false,
			CreatedAt:              now,
			UpdatedAt:              now,
		},
		Preference: domain.Preference{
			CustomerID:             customerID,
			Locale:                 "en-US",
			Timezone:               "UTC",
			DefaultCurrency:        "USD",
			InvoiceDeliveryChannel: "email",
			CreatedAt:              now,
			UpdatedAt:              now,
		},
		Contacts: []domain.Contact{
			{
				ID:         contactID,
				CustomerID: customerID,
				Name:       "Seed Contact",
				Email:      "seed@example.com",
				Role:       "billing",
				IsPrimary:  true,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		},
		Users: []domain.CustomerUser{},
	}
}

func stringPtr(value string) *string {
	return &value
}
