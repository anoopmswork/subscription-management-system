package api

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"subscription-management-system/internal/customer"
	domain "subscription-management-system/internal/domain/customer"
)

// Handler wires HTTP routes to customer service operations.
type Handler struct {
	service *customer.Service
}

// NewHandler constructs an API handler bound to the provided customer service.
func NewHandler(service *customer.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
		"time":   time.Now().UTC(),
	})
}

func (h *Handler) createCustomer(c *fiber.Ctx) error {
	var request customer.CreateCustomerInput
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	result, err := h.service.CreateCustomer(c.Context(), actorFromContext(c), request)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

func (h *Handler) getCustomer(c *fiber.Ctx) error {
	result, err := h.service.GetCustomer(c.Context(), actorFromContext(c), c.Params("customerID"))
	if err != nil {
		return respondError(c, err)
	}

	return c.JSON(result)
}

func (h *Handler) updateCustomer(c *fiber.Ctx) error {
	var request customer.UpdateCustomerInput
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	result, err := h.service.UpdateCustomer(c.Context(), actorFromContext(c), c.Params("customerID"), request)
	if err != nil {
		return respondError(c, err)
	}

	return c.JSON(result)
}

func (h *Handler) changeCustomerStatus(c *fiber.Ctx) error {
	var request customer.ChangeStatusInput
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	result, err := h.service.ChangeStatus(c.Context(), actorFromContext(c), c.Params("customerID"), request)
	if err != nil {
		return respondError(c, err)
	}

	return c.JSON(fiber.Map{"customer": result})
}

func (h *Handler) addContact(c *fiber.Ctx) error {
	var request customer.CreateContactInput
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	result, err := h.service.AddContact(c.Context(), actorFromContext(c), c.Params("customerID"), request)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"contact": result})
}

func (h *Handler) updateContact(c *fiber.Ctx) error {
	var request customer.UpdateContactInput
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	result, err := h.service.UpdateContact(c.Context(), actorFromContext(c), c.Params("customerID"), c.Params("contactID"), request)
	if err != nil {
		return respondError(c, err)
	}

	return c.JSON(fiber.Map{"contact": result})
}

func (h *Handler) inviteUser(c *fiber.Ctx) error {
	var request customer.InviteUserInput
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	result, err := h.service.InviteUser(c.Context(), actorFromContext(c), c.Params("customerID"), request)
	if err != nil {
		return respondError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"user": result})
}

func respondError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, customer.ErrUnauthorized):
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	case errors.Is(err, customer.ErrForbidden):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	case errors.Is(err, customer.ErrValidation):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, customer.ErrConflict):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, customer.ErrNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
}

func actorFromContext(c *fiber.Ctx) domain.Actor {
	return domain.Actor{
		ID:   strings.TrimSpace(c.Get("X-Actor-ID")),
		Role: domain.UserRole(strings.ToLower(strings.TrimSpace(c.Get("X-Actor-Role")))),
	}
}
