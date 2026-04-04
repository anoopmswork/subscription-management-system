package api

import "github.com/gofiber/fiber/v2"

// Register mounts versioned customer-management routes on the Fiber application.
func (h *Handler) Register(app *fiber.App) {
	v1 := app.Group("/v1")

	// GET /v1/health returns service health information.
	v1.Get("/health", h.health)
	// POST /v1/customers creates a new customer account.
	v1.Post("/customers", h.createCustomer)
	// GET /v1/customers/:customerID fetches a customer aggregate by ID.
	v1.Get("/customers/:customerID", h.getCustomer)
	// PATCH /v1/customers/:customerID partially updates customer details.
	v1.Patch("/customers/:customerID", h.updateCustomer)
	// PATCH /v1/customers/:customerID/status transitions customer lifecycle status.
	v1.Patch("/customers/:customerID/status", h.changeCustomerStatus)
	// POST /v1/customers/:customerID/contacts creates a customer contact.
	v1.Post("/customers/:customerID/contacts", h.addContact)
	// PATCH /v1/customers/:customerID/contacts/:contactID updates an existing contact.
	v1.Patch("/customers/:customerID/contacts/:contactID", h.updateContact)
	// POST /v1/customers/:customerID/users/invite invites a user to the customer account.
	v1.Post("/customers/:customerID/users/invite", h.inviteUser)
}
