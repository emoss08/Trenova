package controllers

import (
	"net/http"
	"time"

	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/generalledgeraccount"
	"github.com/emoss08/trenova/middleware"
	"github.com/emoss08/trenova/services"
	"github.com/emoss08/trenova/tools"
	"github.com/emoss08/trenova/tools/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

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
	TagIDs         []uuid.UUID                      `json:"tagIds,omitempty"`
	CreatedAt      time.Time                        `json:"createdAt"`
	UpdatedAt      time.Time                        `json:"updatedAt"`
	Edges          ent.GeneralLedgerAccountEdges    `json:"edges"`
}

// GetGeneralLedgerAccounts gets the general ledger accounts for an organization.
func GetGeneralLedgerAccounts(w http.ResponseWriter, r *http.Request) {
	// TODO(Wolfred): This needs to take in a query parameter for the status of the GL accounts

	offset, limit, err := tools.PaginationParams(r)
	if err != nil {
		tools.ResponseWithError(w, http.StatusBadRequest, types.ValidationErrorResponse{
			Type: "invalidRequest",
			Errors: []types.ValidationErrorDetail{
				{
					Code:   "invalidRequest",
					Detail: err.Error(),
					Attr:   "offset, limit",
				},
			},
		})

		return
	}

	orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)
	buID, buOK := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

	if !ok || !buOK {
		tools.ResponseWithError(w, http.StatusInternalServerError, types.ValidationErrorResponse{
			Type: "internalError",
			Errors: []types.ValidationErrorDetail{
				{
					Code:   "internalError",
					Detail: "Organization ID or Business Unit ID not found in the request context",
					Attr:   "orgID, buID",
				},
			},
		})

		return
	}

	glAccounts, count, err := services.NewGeneralLedgerAccountOps().GetGeneralLedgerAccounts(r.Context(), limit, offset, orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	nextURL := tools.GetNextPageURL(r, limit, offset, count)
	prevURL := tools.GetPrevPageURL(r, limit, offset)

	responses := make([]GeneralLedgerAccountResponse, len(glAccounts))
	for i, account := range glAccounts {
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
			DateOpened:     account.DateOpened,
			DateClosed:     account.DateClosed,
			Notes:          account.Notes,
			IsTaxRelevant:  account.IsTaxRelevant,
			IsReconciled:   account.IsReconciled,
			CreatedAt:      account.CreatedAt,
			UpdatedAt:      account.UpdatedAt,
			TagIDs:         tagIDs,
			Edges:          account.Edges,
		}
	}

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results:  responses,
		Count:    count,
		Next:     nextURL,
		Previous: prevURL,
	})
}

// CreateGeneralLedgerAccount creates a new general ledger account for an organization.
func CreateGeneralLedgerAccount(w http.ResponseWriter, r *http.Request) {
	var newGLAccount services.GeneralLedgerAccountRequest

	orgID, ok := r.Context().Value(middleware.ContextKeyOrgID).(uuid.UUID)
	buID, buOK := r.Context().Value(middleware.ContextKeyBuID).(uuid.UUID)

	if !ok || !buOK {
		tools.ResponseWithError(w, http.StatusInternalServerError, types.ValidationErrorResponse{
			Type: "internalError",
			Errors: []types.ValidationErrorDetail{
				{
					Code:   "internalError",
					Detail: "Organization ID or Business Unit ID not found in the request context",
					Attr:   "orgID, buID",
				},
			},
		})

		return
	}

	newGLAccount.BusinessUnitID = buID
	newGLAccount.OrganizationID = orgID

	if err := tools.ParseBodyAndValidate(w, r, &newGLAccount); err != nil {
		return
	}

	createdGLAccount, err := services.NewGeneralLedgerAccountOps().
		CreateGeneralLedgerAccount(r.Context(), newGLAccount)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusCreated, createdGLAccount)
}

// UpdateGeneralLedgterAccount updates a general ledger account by ID.
func UpdateGeneralLedgerAccount(w http.ResponseWriter, r *http.Request) {
	glAccountID := chi.URLParam(r, "glAccountID")
	if glAccountID == "" {
		return
	}

	var glAccountData services.GeneralLedgerAccountUpdateRequest

	if err := tools.ParseBodyAndValidate(w, r, &glAccountData); err != nil {
		return
	}

	glAccountData.ID = uuid.MustParse(glAccountID)

	glAccount, err := services.NewGeneralLedgerAccountOps().UpdateGeneralLedgerAccount(r.Context(), glAccountData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusBadRequest, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, glAccount)
}
