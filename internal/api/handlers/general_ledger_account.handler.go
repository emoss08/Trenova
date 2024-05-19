package handlers

import (
	"time"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/services"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/generalledgeraccount"
	"github.com/emoss08/trenova/internal/util"
	"github.com/emoss08/trenova/internal/util/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type GeneralLedgerAccountHandler struct {
	Service           *services.GeneralLedgerAccountService
	PermissionService *services.PermissionService
}

func NewGeneralLedgerAccountHandler(s *api.Server) *GeneralLedgerAccountHandler {
	return &GeneralLedgerAccountHandler{
		Service:           services.NewGeneralLedgerAccountService(s),
		PermissionService: services.NewPermissionService(s),
	}
}

// RegisterRoutes registers the routes for the GeneralLedgerAccountHandler.
func (h *GeneralLedgerAccountHandler) RegisterRoutes(r fiber.Router) {
	glAccountAPI := r.Group("/general-ledger-accounts")
	glAccountAPI.Get("/", h.GetGeneralLedgerAccounts())
	glAccountAPI.Post("/", h.CreateGeneralLedgerAccount())
	glAccountAPI.Put("/:glAccountID", h.UpdateGeneralLedgerAccount())
}

type GeneralLedgerAccountResponse struct {
	ID             uuid.UUID                        `json:"id,omitempty"`
	BusinessUnitID uuid.UUID                        `json:"businessUnitId"`
	OrganizationID uuid.UUID                        `json:"organizationId"`
	Status         generalledgeraccount.Status      `json:"status" validate:"required,oneof=A I"`
	AccountNumber  string                           `json:"accountNumber" validate:"required,max=7"`
	AccountType    generalledgeraccount.AccountType `json:"accountType" validate:"required"`
	CashFlowType   string                           `json:"cashFlowType" validate:"omitempty"`
	AccountSubType string                           `json:"accountSubType" validate:"omitempty"`
	AccountClass   string                           `json:"accountClass" validate:"omitempty"`
	Balance        float64                          `json:"balance" validate:"omitempty"`
	InterestRate   float64                          `json:"interestRate" validate:"omitempty"`
	DateOpened     *pgtype.Date                     `json:"dateOpened" validate:"omitempty"`
	DateClosed     *pgtype.Date                     `json:"dateClosed" validate:"omitempty"`
	Notes          string                           `json:"notes,omitempty"`
	IsTaxRelevant  bool                             `json:"isTaxRelevant" validate:"omitempty"`
	IsReconciled   bool                             `json:"isReconciled" validate:"omitempty"`
	Version        int                              `json:"version" validate:"omitempty"`
	TagIDs         []uuid.UUID                      `json:"tagIds,omitempty"`
	CreatedAt      time.Time                        `json:"createdAt"`
	UpdatedAt      time.Time                        `json:"updatedAt"`
	Edges          ent.GeneralLedgerAccountEdges    `json:"edges"`
}

// GetGeneralLedgerAccounts is a handler that returns a list of general ledger accounts.
//
// GET /general-ledger-accounts
func (h *GeneralLedgerAccountHandler) GetGeneralLedgerAccounts() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// TODO(Wolfred): This needs to take in a query parameter for the status of the GL accounts
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
		err := h.PermissionService.CheckUserPermission(c, "generalledgeraccount.view")
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

		entities, count, err := h.Service.GetGeneralLedgerAccounts(c.UserContext(), limit, offset, orgID, buID)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		nextURL := util.GetNextPageURL(c, limit, offset, count)
		prevURL := util.GetPrevPageURL(c, limit, offset)

		responses := make([]GeneralLedgerAccountResponse, len(entities))
		for i, account := range entities {
			tagIDs := make([]uuid.UUID, len(account.Edges.Tags))
			for j, tag := range account.Edges.Tags {
				tagIDs[j] = tag.ID
			}

			responses[i] = GeneralLedgerAccountResponse{
				ID:             account.ID,
				BusinessUnitID: account.BusinessUnitID,
				OrganizationID: account.OrganizationID,
				Status:         account.Status,
				AccountNumber:  account.AccountNumber,
				AccountType:    account.AccountType,
				CashFlowType:   account.CashFlowType,
				AccountSubType: account.AccountSubType,
				AccountClass:   account.AccountClass,
				Balance:        account.Balance,
				InterestRate:   account.InterestRate,
				DateClosed:     account.DateClosed,
				Notes:          account.Notes,
				IsTaxRelevant:  account.IsTaxRelevant,
				IsReconciled:   account.IsReconciled,
				CreatedAt:      account.CreatedAt,
				UpdatedAt:      account.UpdatedAt,
				Version:        account.Version,
				TagIDs:         tagIDs,
				Edges:          account.Edges,
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

// CreateGeneralLedgerAccount is a handler that creates a new general ledger account.
//
// POST /general-ledger-accounts
func (h *GeneralLedgerAccountHandler) CreateGeneralLedgerAccount() fiber.Handler {
	return func(c *fiber.Ctx) error {
		newEntity := new(services.GeneralLedgerAccountRequest)

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
		err := h.PermissionService.CheckUserPermission(c, "generalledgeraccount.add")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		newEntity.BusinessUnitID = buID
		newEntity.OrganizationID = orgID

		if err := util.ParseBodyAndValidate(c, newEntity); err != nil {
			return err
		}

		entity, err := h.Service.CreateGeneralLedgerAccount(c.UserContext(), newEntity)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusCreated).JSON(entity)
	}
}

// UpdateGeneralLedgerAccount is a handler that updates a general ledger account.
//
// PUT /general-ledger-accounts/:glAccountID
func (h *GeneralLedgerAccountHandler) UpdateGeneralLedgerAccount() fiber.Handler {
	return func(c *fiber.Ctx) error {
		glAccountID := c.Params("glAccountID")
		if glAccountID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(types.ValidationErrorResponse{
				Type: "invalidRequest",
				Errors: []types.ValidationErrorDetail{
					{
						Code:   "invalidRequest",
						Detail: "General Ledger Account ID is required",
						Attr:   "glAccountID",
					},
				},
			})
		}

		// Check if the user has the required permission
		err := h.PermissionService.CheckUserPermission(c, "generalledgeraccount.edit")
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "Unauthorized",
				"message": "You do not have the required permission to access this resource",
			})
		}

		updatedEntity := new(services.GeneralLedgerAccountUpdateRequest)

		if err := util.ParseBodyAndValidate(c, updatedEntity); err != nil {
			return err
		}

		updatedEntity.ID = uuid.MustParse(glAccountID)

		entity, err := h.Service.UpdateGeneralLedgerAccount(c.UserContext(), updatedEntity)
		if err != nil {
			errorResponse := util.CreateDBErrorResponse(err)
			return c.Status(fiber.StatusInternalServerError).JSON(errorResponse)
		}

		return c.Status(fiber.StatusOK).JSON(entity)
	}
}
