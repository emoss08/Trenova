package assistantservice

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/maputils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	// exportPageRows is how many rows each read of a list asks for: the most
	// a list tool hands back at once.
	exportPageRows = 50
	// maxExportRows bounds a table's download. A list read again page by page
	// is a set of queries, not a report run, and a person who wants more than
	// this wants the report.
	maxExportRows = 10000
)

var errNotExportable = errortypes.NewValidationError(
	"artifact", errortypes.ErrInvalid, "Only a table or a report preview can be downloaded as CSV",
)

var errNotReread = errortypes.NewBusinessError(
	"This table does not say how it was read, so it cannot be read again. Ask Nova to read it again",
)

// exportSource is an artifact's table read again from where it came from.
type exportSource struct {
	artifact  *assistantartifact.Artifact
	arguments map[string]any
}

func (s *Service) exportSourceOf(
	ctx context.Context,
	req agentports.GetThreadRequest,
	artifactID pulid.ID,
) (*exportSource, error) {
	if _, err := s.conversations.GetThread(ctx, req); err != nil {
		return nil, err
	}
	if s.artifacts == nil {
		return nil, errortypes.NewNotFoundError("Artifact not found")
	}
	artifact, err := s.artifacts.GetByID(ctx, assistantartifactports.GetArtifactRequest{
		ID:         artifactID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if artifact.ThreadID != req.ID {
		return nil, errortypes.NewNotFoundError("Artifact not found")
	}
	if !rereadable(artifact) {
		return nil, errNotExportable
	}
	arguments, recorded := artifact.Payload[payloadReadWith].(map[string]any)
	if !recorded {
		return nil, errNotReread
	}

	return &exportSource{artifact: artifact, arguments: arguments}, nil
}

func rereadable(artifact *assistantartifact.Artifact) bool {
	if artifact.Kind == assistantartifact.KindReportPreview {
		return true
	}

	return artifact.Kind == assistantartifact.KindTableView &&
		assistantartifact.FamilyOf(artifact.Kind, artifact.Payload) == assistantartifact.FamilyTable
}

func noteReadWith(artifact *assistantartifact.Artifact, arguments map[string]any) {
	if arguments != nil && rereadable(artifact) {
		artifact.Payload[payloadReadWith] = maputils.CloneDeep(arguments)
	}
}

// ArtifactCSVName is the file a table downloads as, named for its title.
func (s *Service) ArtifactCSVName(
	ctx context.Context,
	req agentports.GetThreadRequest,
	artifactID pulid.ID,
) (string, error) {
	source, err := s.exportSourceOf(ctx, req, artifactID)
	if err != nil {
		return "", err
	}

	return assistantartifact.SlugBase(source.artifact.Title) + ".csv", nil
}

// ExportArtifactCSV writes a table or a report preview whole. The pane holds
// the rows the agent was shown, cut to what fits; the download is read again
// from the source, so it holds every row the person may see.
func (s *Service) ExportArtifactCSV(
	ctx context.Context,
	req agentports.GetThreadRequest,
	artifactID pulid.ID,
	sink io.Writer,
) error {
	source, err := s.exportSourceOf(ctx, req, artifactID)
	if err != nil {
		return err
	}
	actor := permissionports.UserActor(req.TenantInfo)

	if source.artifact.Kind == assistantartifact.KindReportPreview {
		return s.exportReport(ctx, actor, source, sink)
	}

	return s.exportTable(ctx, req, actor, source, sink)
}

func (s *Service) exportReport(
	ctx context.Context,
	actor *permissionports.RequestActor,
	source *exportSource,
	sink io.Writer,
) error {
	if s.reports == nil {
		return errortypes.NewBusinessError("Reports cannot be downloaded here")
	}
	previewed, err := agentquerytoolservice.ReportOfPreview(ctx, s.reports, actor, source.arguments)
	if err != nil {
		return errortypes.NewBusinessError("This preview can no longer be run: " + err.Error())
	}

	_, err = s.reports.StreamCSV(ctx, &reporting.StreamCSVRequest{
		Request:    agentquerytoolservice.ReportingRequestFor(actor),
		Title:      source.artifact.Title,
		Definition: previewed.Definition,
		Params:     previewed.Params,
	}, sink)

	return err
}

// exportTable reads the list the table came from again, page by page, with
// the arguments it was asked with and the person's own access.
func (s *Service) exportTable(
	ctx context.Context,
	req agentports.GetThreadRequest,
	actor *permissionports.RequestActor,
	source *exportSource,
	sink io.Writer,
) error {
	list, err := s.listOf(ctx, &req, actor, source)
	if err != nil {
		return err
	}

	return writeTableCSV(sink, func(at tablePosition) (map[string]any, error) {
		page := map[string]any{argLimit: exportPageRows}
		switch {
		case at.after != "":
			page[argAfter] = at.after
		case at.offset > 0:
			page[argOffset] = at.offset
		}

		return list.read(ctx, page, false)
	}, list.entity)
}

type tablePosition struct {
	offset int
	after  string
}

// writeTableCSV pages through a list and writes what a person reads of it:
// the columns the table shows, with their labels, and every row.
func writeTableCSV(
	sink io.Writer,
	read func(at tablePosition) (map[string]any, error),
	entity string,
) error {
	writer := csv.NewWriter(sink)
	var columns []assistantartifact.DisplayColumn
	written := 0
	at := tablePosition{}
	for written < maxExportRows {
		page, err := read(at)
		if err != nil {
			return err
		}
		rows, _ := page["items"].([]any)
		if len(rows) == 0 {
			break
		}
		projection := projectTable(entity, stringsOf(page["columns"]), rows)
		if columns == nil {
			columns = projection.columns
			header := make([]string, 0, len(columns))
			for _, column := range columns {
				header = append(header, column.Label)
			}
			if err = writer.Write(header); err != nil {
				return err
			}
		}
		for _, row := range projection.rows {
			record, _ := row.(map[string]any)
			cells := make([]string, 0, len(columns))
			for _, column := range columns {
				cells = append(cells, csvCell(record[column.Key]))
			}
			if err = writer.Write(cells); err != nil {
				return err
			}
			written++
			if written == maxExportRows {
				break
			}
		}
		if !typeutils.BoolOf(page["hasMore"]) {
			break
		}
		if cursor := typeutils.StringOfTrimmed(page["nextCursor"]); cursor != "" {
			if cursor == at.after {
				return errors.New("the list returned the same page cursor twice")
			}
			at = tablePosition{after: cursor}
			continue
		}
		next := int(numberOf(page, "nextOffset"))
		if next <= at.offset {
			next = at.offset + len(rows)
		}
		at = tablePosition{offset: next}
	}
	writer.Flush()

	return writer.Error()
}

// csvCell is a projected value as a spreadsheet reads it: text as itself, a
// named record by its name, a list joined.
func csvCell(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case map[string]any:
		for _, key := range []string{"label", "name", "value", "code"} {
			if text := typeutils.StringOfTrimmed(typed[key]); text != "" {
				return text
			}
		}

		return ""
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, csvCell(item))
		}

		return strings.Join(parts, "; ")
	default:
		return displayValue(typed)
	}
}

// mayRead checks the person may read what the tool reads, as the runtime
// does before the agent runs it for them.
func (s *Service) mayRead(
	ctx context.Context,
	actor *permissionports.RequestActor,
	policy agentports.ToolPolicy,
) error {
	if policy.Resource == "" {
		return nil
	}
	if s.permissions == nil {
		return errortypes.NewAuthorizationError("This table could not be authorized")
	}
	operation := policy.Operation
	if operation == "" {
		operation = permission.OpRead
	}
	result, err := s.permissions.Check(ctx, &permissionports.PermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Resource:       policy.Resource.String(),
		Operation:      operation,
	})
	if err != nil {
		return err
	}
	if !result.Allowed {
		return errortypes.NewAuthorizationError(
			fmt.Sprintf("You do not have %s access to %s", operation, policy.Resource.String()),
		)
	}

	return nil
}

// timezoneOf is the person's own timezone, which is what "today" in the
// arguments the agent passed was read in.
func (s *Service) timezoneOf(ctx context.Context, req agentports.GetThreadRequest) string {
	if s.users == nil {
		return ""
	}
	user, err := s.users.GetByID(ctx, tenantports.GetUserByIDRequest{
		TenantInfo:   req.TenantInfo,
		LookupUserID: req.UserID,
	})
	if err != nil || user == nil {
		return ""
	}

	return user.Timezone
}

// ceilingOf is how sensitive a field the conversation's agent may read, which
// bounds what its table showed and so what its download may hold.
func (s *Service) ceilingOf(
	ctx context.Context,
	req agentports.GetThreadRequest,
) permission.FieldSensitivity {
	definition := s.threadDefinition(ctx, req)
	if definition == nil {
		return permission.SensitivityPublic
	}

	return definition.DataAccessSensitivity()
}

// threadDefinition is the agent the conversation is with, or nil.
func (s *Service) threadDefinition(
	ctx context.Context,
	req agentports.GetThreadRequest,
) *agentdefinition.Definition {
	if s.definitions == nil {
		return nil
	}
	thread, err := s.conversations.GetThread(ctx, req)
	if err != nil || thread.AgentDefinitionID.IsNil() {
		return nil
	}
	definition, err := s.definitions.GetByID(
		ctx,
		agentdefinitionports.GetAgentDefinitionByIDRequest{
			ID:         thread.AgentDefinitionID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil
	}

	return definition
}
