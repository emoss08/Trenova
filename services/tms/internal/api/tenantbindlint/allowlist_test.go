package tenantbindlint

var allowed = map[string]string{
	"github.com/emoss08/trenova/internal/api/handlers/userhandler.Handler.switchOrganization:ShouldBindJSON:*github.com/emoss08/trenova/internal/api/handlers/userhandler.SwitchOrganizationRequest": "organizationId names the organization the caller asks to switch into; the service only switches into one the user holds a membership in",
}
