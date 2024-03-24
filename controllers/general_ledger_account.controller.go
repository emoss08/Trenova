package controllers

import (
	"net/http"

	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/middleware"
	"github.com/emoss08/trenova/services"
	"github.com/emoss08/trenova/tools"
	"github.com/emoss08/trenova/tools/types"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

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

	glAccounts, count, err := services.NewGeneralLedgerAccountOps(r.Context()).GetGeneralLedgerAccounts(limit, offset, orgID, buID)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusInternalServerError, errorResponse)

		return
	}

	nextURL := tools.GetNextPageURL(r, limit, offset, count)
	prevUrl := tools.GetPrevPageURL(r, limit, offset)

	tools.ResponseWithJSON(w, http.StatusOK, types.HTTPResponse{
		Results:  glAccounts,
		Count:    count,
		Next:     nextURL,
		Previous: prevUrl,
	})
}

// CreateGeneralLedgerAccount creates a new general ledger account for an organization.
func CreateGeneralLedgerAccount(w http.ResponseWriter, r *http.Request) {
	var newGLAccount ent.GeneralLedgerAccount

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

	createdGLAccount, err := services.NewGeneralLedgerAccountOps(r.Context()).
		CreateGeneralLedgerAccount(newGLAccount)
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

	var glAccountData ent.GeneralLedgerAccount

	if err := tools.ParseBodyAndValidate(w, r, &glAccountData); err != nil {
		return
	}

	glAccountData.ID = uuid.MustParse(glAccountID)

	glAccount, err := services.NewGeneralLedgerAccountOps(r.Context()).UpdateGeneralLedgerAccount(glAccountData)
	if err != nil {
		errorResponse := tools.CreateDBErrorResponse(err)
		tools.ResponseWithError(w, http.StatusBadRequest, errorResponse)

		return
	}

	tools.ResponseWithJSON(w, http.StatusOK, glAccount)
}
