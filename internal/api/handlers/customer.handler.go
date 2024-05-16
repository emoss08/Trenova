package handlers

import (
	"time"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/customer"
	"github.com/emoss08/trenova/internal/util"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type CustomerHandler struct {
	Server            *api.Server
	Logger            *zerolog.Logger
	Service           *services.CustomerService
	PermissionService *services.PermissionService
}

func NewCustomerHandler(s *api.Server) *CustomerHandler {
	return &CustomerHandler{
		Server:            s,
		Logger:            s.Logger,
		Service:           services.NewCustomerService(s),
		PermissionService: services.NewPermissionService(s),
	}
}

type CustomerResponse struct {
	ID                  uuid.UUID                       `json:"id,omitempty"`
	BusinessUnitID      uuid.UUID                       `json:"businessUnitId"`
	OrganizationID      uuid.UUID                       `json:"organizationId"`
	CreatedAt           time.Time                       `json:"createdAt" validate:"omitempty"`
	UpdatedAt           time.Time                       `json:"updatedAt" validate:"omitempty"`
	Version             int                             `json:"version" validate:"omitempty"`
	Status              customer.Status                 `json:"status" validate:"required,oneof=A I"`
	Code                string                          `json:"code" validate:"required,max=10"`
	Name                string                          `json:"name" validate:"required,max=150"`
	AddressLine1        string                          `json:"addressLine1" validate:"required,max=150"`
	AddressLine2        string                          `json:"addressLine2" validate:"omitempty,max=150"`
	City                string                          `json:"city" validate:"required,max=150"`
	StateID             uuid.UUID                       `json:"stateId" validate:"omitempty,uuid"`
	PostalCode          string                          `json:"postalCode" validate:"required,max=10"`
	HasCustomerPortal   bool                            `json:"hasCustomerPortal" validate:"omitempty"`
	AutoMarkReadyToBill bool                            `json:"autoMarkReadyToBill" validate:"omitempty"`
	EmailProfile        ent.CustomerEmailProfile        `json:"emailProfile" validate:"omitempty"`
	RuleProfile         ent.CustomerRuleProfile         `json:"ruleProfile" validate:"omitempty"`
	DeliverySlots       []ent.DeliverySlot              `json:"deliverySlots" validate:"omitempty"`
	DetentionPolicies   []ent.CustomerDetentionPolicies `json:"detentionPolicies" validate:"omitempty"`
	Contacts            []ent.CustomerContact           `json:"contacts" validate:"omitempty"`
	Edges               ent.CustomerEdges               `json:"edges" validate:"omitempty"`
}

// RegisterRoutes registers the routes for the CustomerHandler.
func (h *CustomerHandler) RegisterRoutes(r fiber.Router) {
	customersAPI := r.Group("/customers")
	customersAPI.Get("/", h.getCustomers())
	customersAPI.Post("/", h.createCustomer())
	customersAPI.Put("/:customerID", h.updateCustomer())
}

// TODO: This is incomplete we need to add customer email & rule profile, delivery slots,
// contact and detention policy

// getCustomers is a handler that returns a list of customers.
//
// GET /customers
func (h *CustomerHandler) getCustomers() fiber.Handler {
	return func(c *fiber.Ctx) error {
		orgID, ok := c.Locals(util.CTXOrganizationID).(uuid.UUID)
		buID, buOK := c.Locals(util.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !buOK {
			return c.Status(fiber.StatusInternalServerError).JSON(types.ValidationErrorResponse{
				Type: "internalError",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "internalError",
						Detail: "Organization ID or Business Unit ID not found in the request context",
						Attr:   "orgID, buID",
					},
				},
			})
		}

		// Check if the user has the required permission
		err := h.PermissionService.CheckUserPermission(c, "customer.view")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		offset, limit, err := util.PaginationParams(c)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: err.Error(),
						Attr:   "offset, limit",
					},
				},
			})
		}

		entities, count, err := h.Service.GetCustomers(c.UserContext(), limit, offset, orgID, buID)
		if err != nil {
			h.Logger.Error().Err(err).Msg("Error fetching customers")
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		nextURL := util.GetNextPageURL(c, limit, offset, count)
		prevURL := util.GetPrevPageURL(c, limit, offset)

		responses := make([]CustomerResponse, len(entities))
		for i, customer := range entities {
			// Directly assign the delivery slots and contacts
			deliverySlots := make([]ent.DeliverySlot, len(customer.Edges.DeliverySlots))
			for i, d := range customer.Edges.DeliverySlots {
				deliverySlots[i] = *d
			}

			// Directly assign the contacts
			contacts := make([]ent.CustomerContact, len(customer.Edges.Contacts))
			for i, contact := range customer.Edges.Contacts {
				contacts[i] = *contact
			}

			// detentionPolocies := make([]ent.CustomerDetentionPolicies, len(customer.Edges.DetentionPolicies))
			// for i, detentionPolicy := range customer.Edges.DetentionPolicies {
			// 	detentionPolocies[i] = *detentionPolocies
			// }

			responses[i] = CustomerResponse{
				ID:                  customer.ID,
				BusinessUnitID:      customer.BusinessUnitID,
				OrganizationID:      customer.OrganizationID,
				CreatedAt:           customer.CreatedAt,
				UpdatedAt:           customer.UpdatedAt,
				Version:             customer.Version,
				Status:              customer.Status,
				Code:                customer.Code,
				Name:                customer.Name,
				AddressLine1:        customer.AddressLine1,
				AddressLine2:        customer.AddressLine2,
				City:                customer.City,
				StateID:             customer.StateID,
				PostalCode:          customer.PostalCode,
				HasCustomerPortal:   customer.HasCustomerPortal,
				AutoMarkReadyToBill: customer.AutoMarkReadyToBill,
				EmailProfile:        *customer.Edges.EmailProfile,
				RuleProfile:         *customer.Edges.RuleProfile,
				DeliverySlots:       deliverySlots,
				// DetentionPolicies:   detentionPolocies,
				Contacts: contacts,
				Edges:    customer.Edges,
			}
		}

		return c.Status(fiber.StatusOK).JSON(types.HTTPResponse{
			Results:  responses,
			Count:    count,
			Next:     nextURL,
			Previous: prevURL,
		})
	}
}

// createCustomer is a handler that creates a new customer.
//
// POST /customers
func (h *CustomerHandler) createCustomer() fiber.Handler {
	return func(c *fiber.Ctx) error {
		newEntity := new(services.CustomerRequest)

		orgID, ok := c.Locals(util.CTXOrganizationID).(uuid.UUID)
		buID, buOK := c.Locals(util.CTXBusinessUnitID).(uuid.UUID)

		if !ok || !buOK {
			return c.Status(fiber.StatusInternalServerError).JSON(types.ValidationErrorResponse{
				Type: "internalError",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "internalError",
						Detail: "Organization ID or Business Unit ID not found in the request context",
						Attr:   "orgID, buID",
					},
				},
			})
		}

		// Check if the user has the required permission
		err := h.PermissionService.CheckUserPermission(c, "customer.add")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		newEntity.BusinessUnitID = buID
		newEntity.OrganizationID = orgID

		if err = util.ParseBodyAndValidate(c, newEntity); err != nil {
			return err
		}

		entity, err := h.Service.CreateCustomer(c.UserContext(), newEntity)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusCreated).JSON(entity)
	}
}

// updateCustomer is a handler that updates a customer.
//
// PUT /customers/:customerID
func (h *CustomerHandler) updateCustomer() fiber.Handler {
	return func(c *fiber.Ctx) error {
		customerID := c.Params("customerID")
		if customerID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: "Comment Type ID is required",
						Attr:   "customerID",
					},
				},
			})
		}

		// Check if the user has the required permission
		err := h.PermissionService.CheckUserPermission(c, "customer.edit")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		updatedEntity := new(ent.Customer)

		if err = util.ParseBodyAndValidate(c, updatedEntity); err != nil {
			return err
		}

		updatedEntity.ID = uuid.MustParse(customerID)

		entity, err := h.Service.UpdateCustomer(c.UserContext(), updatedEntity)
		if err != nil {
			h.Logger.Error().Err(err).Msg("Error updating customer")
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}
