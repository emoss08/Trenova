package ediprofilesecret

import (
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func ProfileSecretAAD(
	tenantInfo pagination.TenantInfo,
	profileID pulid.ID,
	key string,
) encryptionservice.AAD {
	return encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeEDICommunicationProfileItem,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ResourceID:     profileID.String() + ":" + key,
	}
}
