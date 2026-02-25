package customer

import "time"

// Status describes the lifecycle state of a customer account.
type Status string

const (
	// StatusProspect is the pre-activation state for newly created customers.
	StatusProspect Status = "prospect"
	// StatusActive indicates the account is fully operational.
	StatusActive Status = "active"
	// StatusSuspended indicates the account is temporarily disabled.
	StatusSuspended Status = "suspended"
	// StatusArchived indicates the account is permanently retired.
	StatusArchived Status = "archived"
)

// UserRole identifies the authorization level used by customer operations.
type UserRole string

const (
	// RolePlatformAdmin grants platform-wide access across customer accounts.
	RolePlatformAdmin UserRole = "platform_admin"
	// RoleOwner is the highest customer-scoped role.
	RoleOwner UserRole = "owner"
	// RoleAdmin grants administrative customer-scoped access.
	RoleAdmin UserRole = "admin"
	// RoleBilling is intended for billing and finance operations.
	RoleBilling UserRole = "billing"
	// RoleViewer grants read-only access.
	RoleViewer UserRole = "viewer"
)

// Actor captures authenticated caller context for authorization checks.
type Actor struct {
	ID   string
	Role UserRole
}

// Account stores the canonical top-level customer account state.
type Account struct {
	ID          string `json:"id"`
	ExternalRef string `json:"external_ref"`

	LegalName        string  `json:"legal_name"`
	DisplayName      string  `json:"display_name"`
	Segment          string  `json:"segment"`
	ParentCustomerID *string `json:"parent_customer_id,omitempty"`

	Status                 Status `json:"status"`
	OpenInvoiceBalance     int64  `json:"open_invoice_balance_minor"`
	HasActiveSubscriptions bool   `json:"has_active_subscriptions"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Preference stores locale, currency, and invoice-delivery defaults for a customer.
type Preference struct {
	CustomerID             string    `json:"customer_id"`
	Locale                 string    `json:"locale"`
	Timezone               string    `json:"timezone"`
	DefaultCurrency        string    `json:"default_currency"`
	InvoiceDeliveryChannel string    `json:"invoice_delivery_channel"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// Contact stores customer-facing communication and billing contact details.
type Contact struct {
	ID         string     `json:"id"`
	CustomerID string     `json:"customer_id"`
	Name       string     `json:"name"`
	Email      string     `json:"email"`
	Phone      *string    `json:"phone,omitempty"`
	Role       string     `json:"role"`
	IsPrimary  bool       `json:"is_primary"`
	VerifiedAt *time.Time `json:"verified_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// CustomerUser stores user membership details linked to a customer account.
type CustomerUser struct {
	ID         string     `json:"id"`
	CustomerID string     `json:"customer_id"`
	UserID     string     `json:"user_id"`
	Role       UserRole   `json:"role"`
	Status     string     `json:"status"`
	InvitedAt  time.Time  `json:"invited_at"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// Aggregate is the composed customer read model returned by service and API layers.
type Aggregate struct {
	Customer   Account        `json:"customer"`
	Preference Preference     `json:"preference"`
	Contacts   []Contact      `json:"contacts"`
	Users      []CustomerUser `json:"users"`
}
