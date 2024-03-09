package handlers

import (
	"net/http"
	"trenova-go-backend/app/models"
	"trenova-go-backend/utils"

	"github.com/wader/gormstore/v2"
	"gorm.io/gorm"
)

func GetOrganization(db *gorm.DB, store *gormstore.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgId, ok := utils.GetUserOrgFromSession(r, store)

		if !ok {
			utils.ResponseWithError(w, http.StatusUnauthorized, utils.ValidationErrorDetail{
				Code:   "unauthorized",
				Detail: "Unauthorized",
				Attr:   "organizationId",
			})
			return
		}

		var org models.Organization
		if err := db.Model(&models.Organization{}).Where("id = ?", orgId).First(&org).Error; err != nil {
			utils.ResponseWithError(w, http.StatusInternalServerError, utils.ValidationErrorDetail{
				Code:   "notFound",
				Detail: err.Error(),
				Attr:   "id",
			})
			return
		}

		utils.ResponseWithJSON(w, http.StatusOK, org)
	}
}
