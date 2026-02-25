package api

import "github.com/gofiber/fiber/v2"

// Register mounts versioned customer-management routes on the Fiber application.
func (h *Handler) Register(app *fiber.App) {
	v1 := app.Group("/v1")

	v1.Get("/health", h.health)
	v1.Post("/customers", h.createCustomer)
	v1.Get("/customers/:customerID", h.getCustomer)
	v1.Patch("/customers/:customerID", h.updateCustomer)
	v1.Patch("/customers/:customerID/status", h.changeCustomerStatus)
	v1.Post("/customers/:customerID/contacts", h.addContact)
	v1.Patch("/customers/:customerID/contacts/:contactID", h.updateContact)
	v1.Post("/customers/:customerID/users/invite", h.inviteUser)
}
