package customer

import (
	"context"
	"fmt"
	"net/mail"
	"strings"

	"github.com/google/uuid"

	domain "subscription-management-system/internal/domain/customer"
)

// Repository defines the persistence contract required by Service.
//
// Implementations are expected to enforce storage-level guarantees such as
// transactional writes, referential integrity checks, and not-found semantics.
type Repository interface {
	CreateCustomer(ctx context.Context, params CreateCustomerParams) (domain.Aggregate, error)
	GetCustomer(ctx context.Context, customerID string) (domain.Aggregate, error)
	UpdateCustomer(ctx context.Context, customerID string, params UpdateCustomerParams) (domain.Aggregate, error)
	ChangeStatus(ctx context.Context, customerID string, params ChangeStatusParams) (domain.Account, error)
	AddContact(ctx context.Context, customerID string, params AddContactParams) (domain.Contact, error)
	UpdateContact(ctx context.Context, customerID string, contactID string, params UpdateContactParams) (domain.Contact, error)
	InviteUser(ctx context.Context, customerID string, params InviteUserParams) (domain.CustomerUser, error)
}

// Service implements customer-account business rules and authorization checks.
type Service struct {
	repo Repository
}

// NewService builds a Service that delegates persistence to repo.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateCustomerInput is the API payload for creating a customer account.
type CreateCustomerInput struct {
	ExternalRef      string                `json:"external_ref"`
	LegalName        string                `json:"legal_name"`
	DisplayName      string                `json:"display_name"`
	Segment          string                `json:"segment"`
	ParentCustomerID *string               `json:"parent_customer_id"`
	Preference       CreatePreferenceInput `json:"preference"`
	PrimaryContact   CreateContactInput    `json:"primary_contact"`
}

// CreatePreferenceInput captures optional preference values supplied at create time.
type CreatePreferenceInput struct {
	Locale                 string `json:"locale"`
	Timezone               string `json:"timezone"`
	DefaultCurrency        string `json:"default_currency"`
	InvoiceDeliveryChannel string `json:"invoice_delivery_channel"`
}

// UpdateCustomerInput is a partial-update payload for customer account fields.
type UpdateCustomerInput struct {
	DisplayName      *string                `json:"display_name"`
	Segment          *string                `json:"segment"`
	ParentCustomerID *string                `json:"parent_customer_id"`
	Preference       *UpdatePreferenceInput `json:"preference"`
}

// UpdatePreferenceInput is a partial-update payload for customer preferences.
type UpdatePreferenceInput struct {
	Locale                 *string `json:"locale"`
	Timezone               *string `json:"timezone"`
	DefaultCurrency        *string `json:"default_currency"`
	InvoiceDeliveryChannel *string `json:"invoice_delivery_channel"`
}

// CreateContactInput is the API payload for adding a customer contact.
type CreateContactInput struct {
	Name      string  `json:"name"`
	Email     string  `json:"email"`
	Phone     *string `json:"phone"`
	Role      string  `json:"role"`
	IsPrimary bool    `json:"is_primary"`
}

// UpdateContactInput is a partial-update payload for an existing contact.
type UpdateContactInput struct {
	Name      *string `json:"name"`
	Email     *string `json:"email"`
	Phone     *string `json:"phone"`
	Role      *string `json:"role"`
	IsPrimary *bool   `json:"is_primary"`
}

// InviteUserInput is the API payload for inviting a user to a customer account.
type InviteUserInput struct {
	UserID string          `json:"user_id"`
	Role   domain.UserRole `json:"role"`
}

// ChangeStatusInput is the API payload for changing a customer status.
type ChangeStatusInput struct {
	Status domain.Status `json:"status"`
	Reason string        `json:"reason"`
}

// CreateCustomerParams contains normalized values passed to Repository.CreateCustomer.
type CreateCustomerParams struct {
	Actor            domain.Actor
	ExternalRef      string
	LegalName        string
	DisplayName      string
	Segment          string
	ParentCustomerID *string
	Preference       PreferenceParams
	PrimaryContact   ContactParams
}

// PreferenceParams is the repository-facing representation of customer preferences.
type PreferenceParams struct {
	Locale                 string
	Timezone               string
	DefaultCurrency        string
	InvoiceDeliveryChannel string
}

// UpdateCustomerParams contains normalized values passed to Repository.UpdateCustomer.
type UpdateCustomerParams struct {
	Actor            domain.Actor
	DisplayName      string
	Segment          string
	ParentCustomerID *string
	Preference       PreferenceParams
}

// ContactParams is the repository-facing representation of a contact.
type ContactParams struct {
	Name      string
	Email     string
	Phone     *string
	Role      string
	IsPrimary bool
}

// AddContactParams contains normalized values passed to Repository.AddContact.
type AddContactParams struct {
	Actor   domain.Actor
	Contact ContactParams
}

// UpdateContactParams contains normalized values passed to Repository.UpdateContact.
type UpdateContactParams struct {
	Actor   domain.Actor
	Contact ContactParams
}

// InviteUserParams contains normalized values passed to Repository.InviteUser.
type InviteUserParams struct {
	Actor  domain.Actor
	UserID string
	Role   domain.UserRole
}

// ChangeStatusParams contains normalized values passed to Repository.ChangeStatus.
type ChangeStatusParams struct {
	Actor  domain.Actor
	Status domain.Status
	Reason string
}

// CreateCustomer validates actor permissions and required fields, applies defaults,
// normalizes values, and creates the customer aggregate.
//
// Business rules:
//   - Only platform_admin, owner, and admin actors can mutate customer data.
//   - external_ref, legal_name, and display_name are required.
//   - parent_customer_id must be a valid UUID when provided.
//   - Primary contact must include name, role, and a valid email.
//   - Segment defaults to "default" and preference defaults to en-US/UTC/USD/email.
func (s *Service) CreateCustomer(ctx context.Context, actor domain.Actor, input CreateCustomerInput) (domain.Aggregate, error) {
	if err := validateMutatingActor(actor); err != nil {
		return domain.Aggregate{}, err
	}

	if strings.TrimSpace(input.ExternalRef) == "" || strings.TrimSpace(input.LegalName) == "" || strings.TrimSpace(input.DisplayName) == "" {
		return domain.Aggregate{}, wrapValidation("external_ref, legal_name and display_name are required")
	}

	if input.ParentCustomerID != nil && strings.TrimSpace(*input.ParentCustomerID) != "" {
		if err := validateUUID(*input.ParentCustomerID, "parent_customer_id"); err != nil {
			return domain.Aggregate{}, err
		}
	}

	if err := validateContactInput(input.PrimaryContact); err != nil {
		return domain.Aggregate{}, err
	}

	params := CreateCustomerParams{
		Actor:            actor,
		ExternalRef:      strings.TrimSpace(input.ExternalRef),
		LegalName:        strings.TrimSpace(input.LegalName),
		DisplayName:      strings.TrimSpace(input.DisplayName),
		Segment:          defaultString(input.Segment, "default"),
		ParentCustomerID: trimOptional(input.ParentCustomerID),
		Preference:       normalizePreference(input.Preference),
		PrimaryContact: ContactParams{
			Name:      strings.TrimSpace(input.PrimaryContact.Name),
			Email:     strings.ToLower(strings.TrimSpace(input.PrimaryContact.Email)),
			Phone:     trimOptional(input.PrimaryContact.Phone),
			Role:      defaultString(input.PrimaryContact.Role, "billing"),
			IsPrimary: true,
		},
	}

	return s.repo.CreateCustomer(ctx, params)
}

// GetCustomer returns a customer aggregate after validating actor identity and customer ID format.
func (s *Service) GetCustomer(ctx context.Context, actor domain.Actor, customerID string) (domain.Aggregate, error) {
	if err := validateReadActor(actor); err != nil {
		return domain.Aggregate{}, err
	}

	if err := validateUUID(customerID, "customer_id"); err != nil {
		return domain.Aggregate{}, err
	}

	return s.repo.GetCustomer(ctx, customerID)
}

// UpdateCustomer applies a partial update to account and preference fields.
//
// Business rules:
//   - Mutating permissions and customer_id UUID are required.
//   - At least one updatable field must be present.
//   - display_name and segment cannot become empty.
//   - All preference fields must remain non-empty after merge.
func (s *Service) UpdateCustomer(ctx context.Context, actor domain.Actor, customerID string, input UpdateCustomerInput) (domain.Aggregate, error) {
	if err := validateMutatingActor(actor); err != nil {
		return domain.Aggregate{}, err
	}

	if err := validateUUID(customerID, "customer_id"); err != nil {
		return domain.Aggregate{}, err
	}

	current, err := s.repo.GetCustomer(ctx, customerID)
	if err != nil {
		return domain.Aggregate{}, err
	}

	displayName := current.Customer.DisplayName
	segment := current.Customer.Segment
	parentCustomerID := current.Customer.ParentCustomerID
	preference := PreferenceParams{
		Locale:                 current.Preference.Locale,
		Timezone:               current.Preference.Timezone,
		DefaultCurrency:        current.Preference.DefaultCurrency,
		InvoiceDeliveryChannel: current.Preference.InvoiceDeliveryChannel,
	}

	changed := false

	if input.DisplayName != nil {
		displayName = strings.TrimSpace(*input.DisplayName)
		changed = true
	}

	if input.Segment != nil {
		segment = strings.TrimSpace(*input.Segment)
		changed = true
	}

	if input.ParentCustomerID != nil {
		candidate := strings.TrimSpace(*input.ParentCustomerID)
		if candidate == "" {
			parentCustomerID = nil
		} else {
			if err := validateUUID(candidate, "parent_customer_id"); err != nil {
				return domain.Aggregate{}, err
			}
			parentCustomerID = &candidate
		}
		changed = true
	}

	if input.Preference != nil {
		if input.Preference.Locale != nil {
			preference.Locale = strings.TrimSpace(*input.Preference.Locale)
			changed = true
		}
		if input.Preference.Timezone != nil {
			preference.Timezone = strings.TrimSpace(*input.Preference.Timezone)
			changed = true
		}
		if input.Preference.DefaultCurrency != nil {
			preference.DefaultCurrency = strings.ToUpper(strings.TrimSpace(*input.Preference.DefaultCurrency))
			changed = true
		}
		if input.Preference.InvoiceDeliveryChannel != nil {
			preference.InvoiceDeliveryChannel = strings.TrimSpace(*input.Preference.InvoiceDeliveryChannel)
			changed = true
		}
	}

	if !changed {
		return domain.Aggregate{}, wrapValidation("at least one field must be provided")
	}

	if displayName == "" {
		return domain.Aggregate{}, wrapValidation("display_name cannot be empty")
	}

	if segment == "" {
		return domain.Aggregate{}, wrapValidation("segment cannot be empty")
	}

	if preference.Locale == "" || preference.Timezone == "" || preference.DefaultCurrency == "" || preference.InvoiceDeliveryChannel == "" {
		return domain.Aggregate{}, wrapValidation("preference fields cannot be empty")
	}

	return s.repo.UpdateCustomer(ctx, customerID, UpdateCustomerParams{
		Actor:            actor,
		DisplayName:      displayName,
		Segment:          segment,
		ParentCustomerID: parentCustomerID,
		Preference:       preference,
	})
}

// ChangeStatus transitions a customer between allowed lifecycle statuses.
//
// Business rules:
//   - Mutating permissions and customer_id UUID are required.
//   - Allowed transitions: prospect->active, active->suspended|archived,
//     suspended->active|archived.
//   - Archiving is blocked when open_invoice_balance_minor > 0 or when active
//     subscriptions exist.
//   - No-op transitions return the current account without writing.
func (s *Service) ChangeStatus(ctx context.Context, actor domain.Actor, customerID string, input ChangeStatusInput) (domain.Account, error) {
	if err := validateMutatingActor(actor); err != nil {
		return domain.Account{}, err
	}

	if err := validateUUID(customerID, "customer_id"); err != nil {
		return domain.Account{}, err
	}

	target := normalizeStatus(input.Status)
	if !isValidStatus(target) {
		return domain.Account{}, wrapValidation("invalid status")
	}

	current, err := s.repo.GetCustomer(ctx, customerID)
	if err != nil {
		return domain.Account{}, err
	}

	if current.Customer.Status == target {
		return current.Customer, nil
	}

	if !isTransitionAllowed(current.Customer.Status, target) {
		return domain.Account{}, wrapValidation("invalid status transition")
	}

	if target == domain.StatusArchived {
		if current.Customer.OpenInvoiceBalance > 0 {
			return domain.Account{}, fmt.Errorf("cannot archive customer with open invoice balance: %w", ErrConflict)
		}
		if current.Customer.HasActiveSubscriptions {
			return domain.Account{}, fmt.Errorf("cannot archive customer with active subscriptions: %w", ErrConflict)
		}
	}

	return s.repo.ChangeStatus(ctx, customerID, ChangeStatusParams{
		Actor:  actor,
		Status: target,
		Reason: strings.TrimSpace(input.Reason),
	})
}

// AddContact validates and normalizes contact details, then creates the contact for a customer.
//
// Business rules:
//   - Mutating permissions and customer_id UUID are required.
//   - Contact name, role, and a syntactically valid email are required.
//   - Empty role defaults to "billing" before persistence.
func (s *Service) AddContact(ctx context.Context, actor domain.Actor, customerID string, input CreateContactInput) (domain.Contact, error) {
	if err := validateMutatingActor(actor); err != nil {
		return domain.Contact{}, err
	}

	if err := validateUUID(customerID, "customer_id"); err != nil {
		return domain.Contact{}, err
	}

	if err := validateContactInput(input); err != nil {
		return domain.Contact{}, err
	}

	return s.repo.AddContact(ctx, customerID, AddContactParams{
		Actor: actor,
		Contact: ContactParams{
			Name:      strings.TrimSpace(input.Name),
			Email:     strings.ToLower(strings.TrimSpace(input.Email)),
			Phone:     trimOptional(input.Phone),
			Role:      defaultString(input.Role, "billing"),
			IsPrimary: input.IsPrimary,
		},
	})
}

// UpdateContact applies a partial update to an existing contact.
//
// Business rules:
//   - Mutating permissions plus customer_id/contact_id UUID validation are required.
//   - The target contact must exist on the customer aggregate.
//   - At least one field must be updated.
//   - Email must remain valid and name cannot be empty after merge.
//   - Empty role is normalized to "billing".
func (s *Service) UpdateContact(ctx context.Context, actor domain.Actor, customerID string, contactID string, input UpdateContactInput) (domain.Contact, error) {
	if err := validateMutatingActor(actor); err != nil {
		return domain.Contact{}, err
	}

	if err := validateUUID(customerID, "customer_id"); err != nil {
		return domain.Contact{}, err
	}

	if err := validateUUID(contactID, "contact_id"); err != nil {
		return domain.Contact{}, err
	}

	current, err := s.repo.GetCustomer(ctx, customerID)
	if err != nil {
		return domain.Contact{}, err
	}

	var existing *domain.Contact
	for i := range current.Contacts {
		if current.Contacts[i].ID == contactID {
			existing = &current.Contacts[i]
			break
		}
	}

	if existing == nil {
		return domain.Contact{}, ErrNotFound
	}

	updated := ContactParams{
		Name:      existing.Name,
		Email:     existing.Email,
		Phone:     existing.Phone,
		Role:      existing.Role,
		IsPrimary: existing.IsPrimary,
	}

	changed := false

	if input.Name != nil {
		updated.Name = strings.TrimSpace(*input.Name)
		changed = true
	}

	if input.Email != nil {
		updated.Email = strings.ToLower(strings.TrimSpace(*input.Email))
		changed = true
	}

	if input.Phone != nil {
		phone := strings.TrimSpace(*input.Phone)
		if phone == "" {
			updated.Phone = nil
		} else {
			updated.Phone = &phone
		}
		changed = true
	}

	if input.Role != nil {
		updated.Role = strings.TrimSpace(*input.Role)
		changed = true
	}

	if input.IsPrimary != nil {
		updated.IsPrimary = *input.IsPrimary
		changed = true
	}

	if !changed {
		return domain.Contact{}, wrapValidation("at least one contact field must be provided")
	}

	if err := validateEmail(updated.Email); err != nil {
		return domain.Contact{}, err
	}

	if strings.TrimSpace(updated.Name) == "" {
		return domain.Contact{}, wrapValidation("contact name is required")
	}

	if strings.TrimSpace(updated.Role) == "" {
		updated.Role = "billing"
	}

	return s.repo.UpdateContact(ctx, customerID, contactID, UpdateContactParams{Actor: actor, Contact: updated})
}

// InviteUser validates invitation inputs and delegates creation of an invited customer user.
//
// Business rules:
//   - Mutating permissions and customer_id UUID are required.
//   - user_id is required.
//   - Only owner, admin, billing, and viewer roles are invitable.
func (s *Service) InviteUser(ctx context.Context, actor domain.Actor, customerID string, input InviteUserInput) (domain.CustomerUser, error) {
	if err := validateMutatingActor(actor); err != nil {
		return domain.CustomerUser{}, err
	}

	if err := validateUUID(customerID, "customer_id"); err != nil {
		return domain.CustomerUser{}, err
	}

	if strings.TrimSpace(input.UserID) == "" {
		return domain.CustomerUser{}, wrapValidation("user_id is required")
	}

	role := normalizeRole(input.Role)
	if !isInvitableRole(role) {
		return domain.CustomerUser{}, wrapValidation("invalid user role")
	}

	return s.repo.InviteUser(ctx, customerID, InviteUserParams{
		Actor:  actor,
		UserID: strings.TrimSpace(input.UserID),
		Role:   role,
	})
}

func validateMutatingActor(actor domain.Actor) error {
	if err := validateActor(actor); err != nil {
		return err
	}

	if actor.Role != domain.RolePlatformAdmin && actor.Role != domain.RoleOwner && actor.Role != domain.RoleAdmin {
		return ErrForbidden
	}

	return nil
}

func validateReadActor(actor domain.Actor) error {
	return validateActor(actor)
}

func validateActor(actor domain.Actor) error {
	if strings.TrimSpace(actor.ID) == "" {
		return ErrUnauthorized
	}

	role := normalizeRole(actor.Role)
	if role == "" {
		return ErrUnauthorized
	}

	switch role {
	case domain.RolePlatformAdmin, domain.RoleOwner, domain.RoleAdmin, domain.RoleBilling, domain.RoleViewer:
		return nil
	default:
		return ErrUnauthorized
	}
}

func validateContactInput(input CreateContactInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return wrapValidation("contact name is required")
	}
	if err := validateEmail(input.Email); err != nil {
		return err
	}
	if strings.TrimSpace(input.Role) == "" {
		return wrapValidation("contact role is required")
	}
	return nil
}

func validateEmail(email string) error {
	if strings.TrimSpace(email) == "" {
		return wrapValidation("email is required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return wrapValidation("invalid email")
	}
	return nil
}

func validateUUID(raw string, field string) error {
	if _, err := uuid.Parse(strings.TrimSpace(raw)); err != nil {
		return wrapValidation(fmt.Sprintf("invalid %s", field))
	}
	return nil
}

func normalizePreference(input CreatePreferenceInput) PreferenceParams {
	locale := defaultString(input.Locale, "en-US")
	timezone := defaultString(input.Timezone, "UTC")
	currency := strings.ToUpper(defaultString(input.DefaultCurrency, "USD"))
	channel := defaultString(input.InvoiceDeliveryChannel, "email")

	return PreferenceParams{
		Locale:                 locale,
		Timezone:               timezone,
		DefaultCurrency:        currency,
		InvoiceDeliveryChannel: channel,
	}
}

func defaultString(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func trimOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeStatus(status domain.Status) domain.Status {
	return domain.Status(strings.ToLower(strings.TrimSpace(string(status))))
}

func isValidStatus(status domain.Status) bool {
	switch status {
	case domain.StatusProspect, domain.StatusActive, domain.StatusSuspended, domain.StatusArchived:
		return true
	default:
		return false
	}
}

func isTransitionAllowed(from domain.Status, to domain.Status) bool {
	allowed := map[domain.Status]map[domain.Status]struct{}{
		domain.StatusProspect: {
			domain.StatusActive: {},
		},
		domain.StatusActive: {
			domain.StatusSuspended: {},
			domain.StatusArchived:  {},
		},
		domain.StatusSuspended: {
			domain.StatusActive:   {},
			domain.StatusArchived: {},
		},
		domain.StatusArchived: {},
	}

	_, ok := allowed[from][to]
	return ok
}

func normalizeRole(role domain.UserRole) domain.UserRole {
	return domain.UserRole(strings.ToLower(strings.TrimSpace(string(role))))
}

func isInvitableRole(role domain.UserRole) bool {
	switch role {
	case domain.RoleOwner, domain.RoleAdmin, domain.RoleBilling, domain.RoleViewer:
		return true
	default:
		return false
	}
}

func wrapValidation(message string) error {
	return fmt.Errorf("%s: %w", message, ErrValidation)
}
