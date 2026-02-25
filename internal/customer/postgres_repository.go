package customer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "subscription-management-system/internal/domain/customer"
)

// PostgresRepository persists customer aggregates in PostgreSQL.
//
// Write operations are executed in transactions and append corresponding
// domain events to the outbox_events table.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a PostgreSQL-backed Repository implementation.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// CreateCustomer inserts account, preference, and primary-contact records, then
// emits outbox events for account creation and contact creation.
func (r *PostgresRepository) CreateCustomer(ctx context.Context, params CreateCustomerParams) (domain.Aggregate, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Aggregate{}, err
	}
	defer tx.Rollback(ctx)

	now := time.Now().UTC()
	customerID := uuid.NewString()
	contactID := uuid.NewString()

	_, err = tx.Exec(ctx, `
		INSERT INTO customer_accounts (
			id, external_ref, legal_name, display_name, segment, status, parent_customer_id,
			open_invoice_balance_minor, has_active_subscriptions, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 0, false, $8, $8)
	`,
		customerID,
		params.ExternalRef,
		params.LegalName,
		params.DisplayName,
		params.Segment,
		domain.StatusProspect,
		params.ParentCustomerID,
		now,
	)
	if err != nil {
		return domain.Aggregate{}, mapDBError(err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO customer_preferences (
			customer_id, locale, timezone, default_currency, invoice_delivery_channel, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
	`, customerID, params.Preference.Locale, params.Preference.Timezone, params.Preference.DefaultCurrency, params.Preference.InvoiceDeliveryChannel, now)
	if err != nil {
		return domain.Aggregate{}, mapDBError(err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO customer_contacts (
			id, customer_id, name, email, phone, role, is_primary, verified_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, true, NULL, $7, $7)
	`, contactID, customerID, params.PrimaryContact.Name, params.PrimaryContact.Email, params.PrimaryContact.Phone, params.PrimaryContact.Role, now)
	if err != nil {
		return domain.Aggregate{}, mapDBError(err)
	}

	if err := r.insertOutboxEvent(ctx, tx, "customer", customerID, "customer.account.created", map[string]any{
		"customer_id": customerID,
		"actor_id":    params.Actor.ID,
		"status":      domain.StatusProspect,
	}); err != nil {
		return domain.Aggregate{}, err
	}

	if err := r.insertOutboxEvent(ctx, tx, "customer", customerID, "customer.contact.added", map[string]any{
		"customer_id": customerID,
		"contact_id":  contactID,
		"actor_id":    params.Actor.ID,
	}); err != nil {
		return domain.Aggregate{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Aggregate{}, err
	}

	return r.GetCustomer(ctx, customerID)
}

// GetCustomer loads the account, preference, contacts, and users that compose a customer aggregate.
func (r *PostgresRepository) GetCustomer(ctx context.Context, customerID string) (domain.Aggregate, error) {
	account, err := r.fetchAccount(ctx, r.pool, customerID)
	if err != nil {
		return domain.Aggregate{}, err
	}

	pref, err := r.fetchPreference(ctx, r.pool, customerID)
	if err != nil {
		return domain.Aggregate{}, err
	}

	contacts, err := r.fetchContacts(ctx, r.pool, customerID)
	if err != nil {
		return domain.Aggregate{}, err
	}

	users, err := r.fetchUsers(ctx, r.pool, customerID)
	if err != nil {
		return domain.Aggregate{}, err
	}

	return domain.Aggregate{
		Customer:   account,
		Preference: pref,
		Contacts:   contacts,
		Users:      users,
	}, nil
}

// UpdateCustomer updates account and preference data in one transaction and
// emits a customer.account.updated outbox event.
func (r *PostgresRepository) UpdateCustomer(ctx context.Context, customerID string, params UpdateCustomerParams) (domain.Aggregate, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Aggregate{}, err
	}
	defer tx.Rollback(ctx)

	if err := r.ensureCustomerExists(ctx, tx, customerID); err != nil {
		return domain.Aggregate{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE customer_accounts
		SET display_name = $2, segment = $3, parent_customer_id = $4, updated_at = NOW()
		WHERE id = $1
	`, customerID, params.DisplayName, params.Segment, params.ParentCustomerID)
	if err != nil {
		return domain.Aggregate{}, mapDBError(err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO customer_preferences (customer_id, locale, timezone, default_currency, invoice_delivery_channel, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (customer_id) DO UPDATE
		SET locale = EXCLUDED.locale,
			timezone = EXCLUDED.timezone,
			default_currency = EXCLUDED.default_currency,
			invoice_delivery_channel = EXCLUDED.invoice_delivery_channel,
			updated_at = NOW()
	`, customerID, params.Preference.Locale, params.Preference.Timezone, params.Preference.DefaultCurrency, params.Preference.InvoiceDeliveryChannel)
	if err != nil {
		return domain.Aggregate{}, mapDBError(err)
	}

	if err := r.insertOutboxEvent(ctx, tx, "customer", customerID, "customer.account.updated", map[string]any{
		"customer_id": customerID,
		"actor_id":    params.Actor.ID,
	}); err != nil {
		return domain.Aggregate{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Aggregate{}, err
	}

	return r.GetCustomer(ctx, customerID)
}

// ChangeStatus updates a customer account status and records a status-changed outbox event.
func (r *PostgresRepository) ChangeStatus(ctx context.Context, customerID string, params ChangeStatusParams) (domain.Account, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Account{}, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		UPDATE customer_accounts
		SET status = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, external_ref, legal_name, display_name, segment, status,
			parent_customer_id, open_invoice_balance_minor, has_active_subscriptions, created_at, updated_at
	`, customerID, params.Status)

	account, err := scanAccount(row)
	if err != nil {
		return domain.Account{}, mapDBError(err)
	}

	if err := r.insertOutboxEvent(ctx, tx, "customer", customerID, "customer.account.status_changed", map[string]any{
		"customer_id": customerID,
		"actor_id":    params.Actor.ID,
		"new_status":  params.Status,
		"reason":      params.Reason,
	}); err != nil {
		return domain.Account{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Account{}, err
	}

	return account, nil
}

// AddContact inserts a new contact for a customer.
//
// When params.Contact.IsPrimary is true, any existing primary contact for the
// same customer is demoted before insertion.
func (r *PostgresRepository) AddContact(ctx context.Context, customerID string, params AddContactParams) (domain.Contact, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Contact{}, err
	}
	defer tx.Rollback(ctx)

	if err := r.ensureCustomerExists(ctx, tx, customerID); err != nil {
		return domain.Contact{}, err
	}

	if params.Contact.IsPrimary {
		_, err = tx.Exec(ctx, `UPDATE customer_contacts SET is_primary = false, updated_at = NOW() WHERE customer_id = $1 AND is_primary = true`, customerID)
		if err != nil {
			return domain.Contact{}, mapDBError(err)
		}
	}

	contactID := uuid.NewString()
	row := tx.QueryRow(ctx, `
		INSERT INTO customer_contacts (id, customer_id, name, email, phone, role, is_primary, verified_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULL, NOW(), NOW())
		RETURNING id, customer_id, name, email, phone, role, is_primary, verified_at, created_at, updated_at
	`, contactID, customerID, params.Contact.Name, params.Contact.Email, params.Contact.Phone, params.Contact.Role, params.Contact.IsPrimary)

	contact, err := scanContact(row)
	if err != nil {
		return domain.Contact{}, mapDBError(err)
	}

	if err := r.insertOutboxEvent(ctx, tx, "customer", customerID, "customer.contact.added", map[string]any{
		"customer_id": customerID,
		"contact_id":  contact.ID,
		"actor_id":    params.Actor.ID,
	}); err != nil {
		return domain.Contact{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Contact{}, err
	}

	return contact, nil
}

// UpdateContact updates an existing contact and records a customer.contact.updated outbox event.
//
// When params.Contact.IsPrimary is true, other primary contacts for the same
// customer are demoted in the same transaction.
func (r *PostgresRepository) UpdateContact(ctx context.Context, customerID string, contactID string, params UpdateContactParams) (domain.Contact, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Contact{}, err
	}
	defer tx.Rollback(ctx)

	if err := r.ensureCustomerExists(ctx, tx, customerID); err != nil {
		return domain.Contact{}, err
	}

	if params.Contact.IsPrimary {
		_, err = tx.Exec(ctx, `
			UPDATE customer_contacts
			SET is_primary = false, updated_at = NOW()
			WHERE customer_id = $1 AND id <> $2 AND is_primary = true
		`, customerID, contactID)
		if err != nil {
			return domain.Contact{}, mapDBError(err)
		}
	}

	row := tx.QueryRow(ctx, `
		UPDATE customer_contacts
		SET name = $3,
			email = $4,
			phone = $5,
			role = $6,
			is_primary = $7,
			updated_at = NOW()
		WHERE customer_id = $1 AND id = $2
		RETURNING id, customer_id, name, email, phone, role, is_primary, verified_at, created_at, updated_at
	`, customerID, contactID, params.Contact.Name, params.Contact.Email, params.Contact.Phone, params.Contact.Role, params.Contact.IsPrimary)

	contact, err := scanContact(row)
	if err != nil {
		return domain.Contact{}, mapDBError(err)
	}

	if err := r.insertOutboxEvent(ctx, tx, "customer", customerID, "customer.contact.updated", map[string]any{
		"customer_id": customerID,
		"contact_id":  contactID,
		"actor_id":    params.Actor.ID,
	}); err != nil {
		return domain.Contact{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Contact{}, err
	}

	return contact, nil
}

// InviteUser creates a customer_users row in invited state and emits a
// customer.user.invited outbox event.
func (r *PostgresRepository) InviteUser(ctx context.Context, customerID string, params InviteUserParams) (domain.CustomerUser, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.CustomerUser{}, err
	}
	defer tx.Rollback(ctx)

	if err := r.ensureCustomerExists(ctx, tx, customerID); err != nil {
		return domain.CustomerUser{}, err
	}

	row := tx.QueryRow(ctx, `
		INSERT INTO customer_users (id, customer_id, user_id, role, status, invited_at, accepted_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'invited', NOW(), NULL, NOW(), NOW())
		RETURNING id, customer_id, user_id, role, status, invited_at, accepted_at, created_at, updated_at
	`, uuid.NewString(), customerID, params.UserID, params.Role)

	user, err := scanUser(row)
	if err != nil {
		return domain.CustomerUser{}, mapDBError(err)
	}

	if err := r.insertOutboxEvent(ctx, tx, "customer", customerID, "customer.user.invited", map[string]any{
		"customer_id": customerID,
		"user_id":     params.UserID,
		"role":        params.Role,
		"actor_id":    params.Actor.ID,
	}); err != nil {
		return domain.CustomerUser{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.CustomerUser{}, err
	}

	return user, nil
}

type queryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (r *PostgresRepository) fetchAccount(ctx context.Context, q queryer, customerID string) (domain.Account, error) {
	row := q.QueryRow(ctx, `
		SELECT id, external_ref, legal_name, display_name, segment, status,
			parent_customer_id, open_invoice_balance_minor, has_active_subscriptions, created_at, updated_at
		FROM customer_accounts
		WHERE id = $1
	`, customerID)

	account, err := scanAccount(row)
	if err != nil {
		return domain.Account{}, mapDBError(err)
	}

	return account, nil
}

func (r *PostgresRepository) fetchPreference(ctx context.Context, q queryer, customerID string) (domain.Preference, error) {
	row := q.QueryRow(ctx, `
		SELECT customer_id, locale, timezone, default_currency, invoice_delivery_channel, created_at, updated_at
		FROM customer_preferences
		WHERE customer_id = $1
	`, customerID)

	pref, err := scanPreference(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Preference{
				CustomerID:             customerID,
				Locale:                 "en-US",
				Timezone:               "UTC",
				DefaultCurrency:        "USD",
				InvoiceDeliveryChannel: "email",
			}, nil
		}
		return domain.Preference{}, mapDBError(err)
	}

	return pref, nil
}

func (r *PostgresRepository) fetchContacts(ctx context.Context, q queryer, customerID string) ([]domain.Contact, error) {
	rows, err := q.Query(ctx, `
		SELECT id, customer_id, name, email, phone, role, is_primary, verified_at, created_at, updated_at
		FROM customer_contacts
		WHERE customer_id = $1
		ORDER BY is_primary DESC, created_at ASC
	`, customerID)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()

	contacts := make([]domain.Contact, 0)
	for rows.Next() {
		contact, scanErr := scanContact(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		contacts = append(contacts, contact)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return contacts, nil
}

func (r *PostgresRepository) fetchUsers(ctx context.Context, q queryer, customerID string) ([]domain.CustomerUser, error) {
	rows, err := q.Query(ctx, `
		SELECT id, customer_id, user_id, role, status, invited_at, accepted_at, created_at, updated_at
		FROM customer_users
		WHERE customer_id = $1
		ORDER BY created_at ASC
	`, customerID)
	if err != nil {
		return nil, mapDBError(err)
	}
	defer rows.Close()

	users := make([]domain.CustomerUser, 0)
	for rows.Next() {
		user, scanErr := scanUser(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		users = append(users, user)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return users, nil
}

func (r *PostgresRepository) ensureCustomerExists(ctx context.Context, tx pgx.Tx, customerID string) error {
	row := tx.QueryRow(ctx, `SELECT id FROM customer_accounts WHERE id = $1`, customerID)
	var id string
	if err := row.Scan(&id); err != nil {
		return mapDBError(err)
	}
	return nil
}

func (r *PostgresRepository) insertOutboxEvent(ctx context.Context, tx pgx.Tx, aggregateType string, aggregateID string, eventName string, payload any) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events (id, aggregate_type, aggregate_id, event_name, payload, created_at, processed_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NULL)
	`, uuid.NewString(), aggregateType, aggregateID, eventName, payloadJSON)
	if err != nil {
		return mapDBError(err)
	}

	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAccount(row scanner) (domain.Account, error) {
	var account domain.Account
	var status string
	var parent sql.NullString

	err := row.Scan(
		&account.ID,
		&account.ExternalRef,
		&account.LegalName,
		&account.DisplayName,
		&account.Segment,
		&status,
		&parent,
		&account.OpenInvoiceBalance,
		&account.HasActiveSubscriptions,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		return domain.Account{}, err
	}

	account.Status = domain.Status(status)
	if parent.Valid {
		parentID := parent.String
		account.ParentCustomerID = &parentID
	}

	return account, nil
}

func scanPreference(row scanner) (domain.Preference, error) {
	var pref domain.Preference
	err := row.Scan(
		&pref.CustomerID,
		&pref.Locale,
		&pref.Timezone,
		&pref.DefaultCurrency,
		&pref.InvoiceDeliveryChannel,
		&pref.CreatedAt,
		&pref.UpdatedAt,
	)
	if err != nil {
		return domain.Preference{}, err
	}
	return pref, nil
}

func scanContact(row scanner) (domain.Contact, error) {
	var contact domain.Contact
	var phone sql.NullString
	var verifiedAt sql.NullTime

	err := row.Scan(
		&contact.ID,
		&contact.CustomerID,
		&contact.Name,
		&contact.Email,
		&phone,
		&contact.Role,
		&contact.IsPrimary,
		&verifiedAt,
		&contact.CreatedAt,
		&contact.UpdatedAt,
	)
	if err != nil {
		return domain.Contact{}, err
	}

	if phone.Valid {
		phoneValue := phone.String
		contact.Phone = &phoneValue
	}

	if verifiedAt.Valid {
		verified := verifiedAt.Time
		contact.VerifiedAt = &verified
	}

	return contact, nil
}

func scanUser(row scanner) (domain.CustomerUser, error) {
	var user domain.CustomerUser
	var role string
	var acceptedAt sql.NullTime

	err := row.Scan(
		&user.ID,
		&user.CustomerID,
		&user.UserID,
		&role,
		&user.Status,
		&user.InvitedAt,
		&acceptedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return domain.CustomerUser{}, err
	}

	user.Role = domain.UserRole(role)
	if acceptedAt.Valid {
		accepted := acceptedAt.Time
		user.AcceptedAt = &accepted
	}

	return user, nil
}

func mapDBError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return fmt.Errorf("unique constraint violation: %w", ErrConflict)
		case "23503", "23514", "22P02":
			return fmt.Errorf("invalid database input: %w", ErrValidation)
		}
	}

	return err
}
