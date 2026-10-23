package formulatemplateservice

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	tarifferrors "github.com/emoss08/trenova/internal/modules/rating/app/tariff/errors"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	previewEvaluationTimeout = 2 * time.Second
	MaxPreviewInputs         = 100
	MaxTestNameLength        = 100

	CodeTimeout = "timeout"
	CodeInvalid = "invalid"
)

type PreviewRequest struct {
	TenantInfo pagination.TenantInfo
	Source     string
	Inputs     tarifftypes.Inputs
	ShipmentID *pulid.ID
}

func previewBounds(source string, inputs tarifftypes.Inputs) error {
	multiErr := errortypes.NewMultiError()
	switch {
	case strings.TrimSpace(source) == "":
		multiErr.Add(fieldSource, errortypes.ErrRequired, "Source is required")
	case len(source) > tarifftypes.MaxSourceLength:
		multiErr.Add(fieldSource, errortypes.ErrInvalid, "Source cannot exceed 64 KiB")
	}
	if len(inputs) > MaxPreviewInputs {
		multiErr.Add(
			fieldInputs,
			errortypes.ErrInvalid,
			"A preview takes at most {0} inputs",
			MaxPreviewInputs,
		)
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func timeoutMessage() rules.Message {
	return rules.Message{
		Template: "The formula took longer than {0} to price, so it was stopped",
		Args:     []any{previewEvaluationTimeout.String()},
	}
}

func (s *Service) Preview(
	ctx context.Context,
	req *PreviewRequest,
) (*formulatemplateports.PreviewResult, error) {
	if err := previewBounds(req.Source, req.Inputs); err != nil {
		return nil, err
	}

	content, err := s.checkContent(ctx, &contentRequest{
		tenant: req.TenantInfo,
		source: req.Source,
		mode:   tariff.LibrariesLatest,
	})
	if err != nil {
		return nil, err
	}

	result := previewOf(content)
	if !content.checks() {
		return result, nil
	}

	shipment, err := s.shipmentRecord(ctx, req.TenantInfo, req.ShipmentID)
	if err != nil {
		return nil, err
	}

	rated, err := s.rateSource(ctx, &sourceRating{
		tenant:    req.TenantInfo,
		source:    req.Source,
		libraries: content.checked.Context.Libraries,
		inputs:    req.Inputs,
		shipment:  shipment,
		trace:     true,
	})
	if err != nil {
		failure, ok := previewFailure(ctx, err)
		if !ok {
			return nil, err
		}
		result.Error = failure
		return result, nil
	}

	result.Bill = rated.WireBill
	result.RatedBill = rated.Bill
	result.Refusal = rated.Refusal
	result.Trace = rated.Trace
	result.Lookups = rated.Lookups
	return result, nil
}

func previewOf(content *contentCheck) *formulatemplateports.PreviewResult {
	check := content.checked.Check
	result := &formulatemplateports.PreviewResult{
		Valid:       content.checks(),
		Diagnostics: check.Diagnostics,
		Inputs:      tarifftypes.InputDeclsFrom(check.Inputs),
		Imports:     content.checked.Imports,
		Charges:     check.Charges,
		Tests:       content.tests.Tests,
	}
	if result.Diagnostics == nil {
		result.Diagnostics = []rules.TariffDiagnostic{}
	}
	if result.Charges == nil {
		result.Charges = []rules.TariffChargeRef{}
	}
	if result.Imports == nil {
		result.Imports = []tarifftypes.Import{}
	}
	if result.Tests == nil {
		result.Tests = []rules.TariffTestResult{}
	}
	return result
}

func previewFailure(ctx context.Context, err error) (*formulatemplateports.PreviewError, bool) {
	locale := i18n.FromContext(ctx)
	if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
		return &formulatemplateports.PreviewError{
			Code:    CodeTimeout,
			Message: timeoutMessage().Localize(locale),
		}, true
	}

	var evaluation *tarifferrors.EvaluationError
	if errors.As(err, &evaluation) {
		return &formulatemplateports.PreviewError{
			Code:    evaluation.Code,
			Message: evaluation.Message.Localize(locale),
			Range:   evaluation.Range,
		}, true
	}

	var business *errortypes.BusinessError
	if errors.As(err, &business) {
		code := business.Params["code"]
		if code == "" {
			code = CodeInvalid
		}
		return &formulatemplateports.PreviewError{
			Code:    code,
			Message: errortypes.LocalizedText(ctx, err),
		}, true
	}

	if errortypes.IsError(err) {
		return &formulatemplateports.PreviewError{
			Code:    CodeInvalid,
			Message: errortypes.LocalizedText(ctx, err),
		}, true
	}

	return nil, false
}

type sourceRating struct {
	tenant    pagination.TenantInfo
	name      string
	source    string
	libraries []rules.TariffLibrary
	inputs    tarifftypes.Inputs
	shipment  *tariffports.RatedShipment
	trace     bool
}

func (s *Service) rateSource(
	ctx context.Context,
	req *sourceRating,
) (*tariffports.RateResult, error) {
	leashed, cancel := context.WithTimeout(ctx, previewEvaluationTimeout)
	defer cancel()

	rate := &tariffports.RateRequest{
		TenantInfo: req.tenant,
		Template: &tariffports.Template{
			Name:      req.name,
			Source:    req.source,
			Libraries: req.libraries,
		},
		Inputs: req.inputs,
		Record: rules.NoneValue(),
		Trace:  req.trace,
	}
	if req.shipment != nil {
		rate.Record = req.shipment.Record
		rate.Currency = req.shipment.Currency
		rate.Exchange = req.shipment.Exchange
		if rate.Inputs == nil {
			rate.Inputs = req.shipment.Inputs
		}
	}

	result, err := s.rater.Rate(leashed, rate)
	if err != nil && ctx.Err() == nil && errors.Is(leashed.Err(), context.DeadlineExceeded) {
		return nil, timeoutMessage().ValidationError(fieldSource, errortypes.ErrInvalid).
			WithInternal(context.DeadlineExceeded)
	}
	return result, err
}

func (s *Service) shipmentRecord(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID *pulid.ID,
) (*tariffports.RatedShipment, error) {
	if shipmentID == nil || shipmentID.IsNil() {
		return nil, nil //nolint:nilnil // no shipment means the sample record
	}
	if s.records == nil {
		return nil, errortypes.NewBusinessError(
			"Previewing a formula on a shipment is not available on this server",
		)
	}
	record, err := s.records.ShipmentRecord(ctx, &tariffports.ShipmentRecordRequest{
		TenantInfo: tenantInfo,
		ShipmentID: *shipmentID,
	})
	if err != nil {
		return nil, err
	}
	return &tariffports.RatedShipment{ShipmentID: *shipmentID, Record: record}, nil
}

type StudioLibrary struct {
	Path          string   `json:"path"`
	LibraryID     pulid.ID `json:"libraryId"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	VersionNumber int64    `json:"versionNumber"`
}

type StudioContext struct {
	Context     *rules.TariffContext `json:"context"`
	Libraries   []StudioLibrary      `json:"libraries"`
	TablesStamp string               `json:"tablesStamp"`
}

func (s *Service) StudioContext(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*StudioContext, error) {
	versions, err := s.libraries.ListLatestApproved(
		ctx,
		&formulalibraryports.ListApprovedLibrariesRequest{TenantInfo: tenantInfo},
	)
	if err != nil {
		return nil, err
	}

	stamp, err := s.checker.TableStamp(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	set := tariff.LibrarySet(versions)
	tariffCtx, err := s.checker.Context(ctx, tenantInfo, set)
	if err != nil {
		return nil, err
	}

	return &StudioContext{
		Context:     tariffCtx,
		Libraries:   studioLibraries(versions),
		TablesStamp: stamp,
	}, nil
}

const (
	MaxStudioCellNeeds = 200
	maxNeedKeys        = 4
	maxNeedTextLength  = 200
	fieldNeeds         = "needs"
)

type StudioCellsRequest struct {
	TenantInfo pagination.TenantInfo
	Needs      []rules.TariffNeed
	Whole      bool
}

func (s *Service) StudioCells(
	ctx context.Context,
	req *StudioCellsRequest,
) (*tariff.TableCells, error) {
	if err := validateNeeds(req.Needs); err != nil {
		return nil, err
	}
	return s.checker.TableCells(ctx, &tariff.TableCellsRequest{
		TenantInfo: req.TenantInfo,
		Needs:      req.Needs,
		Whole:      req.Whole,
	})
}

func validateNeeds(needs []rules.TariffNeed) error {
	switch {
	case len(needs) == 0:
		return errortypes.NewValidationError(
			fieldNeeds,
			errortypes.ErrRequired,
			"Name the rate table cells to load",
		)
	case len(needs) > MaxStudioCellNeeds:
		return errortypes.NewValidationError(
			fieldNeeds,
			errortypes.ErrInvalid,
			"Load at most 200 rate table key combinations at a time",
		)
	}
	for i := range needs {
		if !validNeed(&needs[i]) {
			return errortypes.NewValidationError(
				fieldNeeds,
				errortypes.ErrInvalid,
				"Each rate table need names a table and at most four keys",
			)
		}
	}
	return nil
}

func validNeed(need *rules.TariffNeed) bool {
	if need.Table == "" || len(need.Table) > maxNeedTextLength || len(need.Keys) > maxNeedKeys {
		return false
	}
	for _, key := range need.Keys {
		if len(key) > maxNeedTextLength {
			return false
		}
	}
	return true
}

func studioLibraries(versions []*formulalibrary.FormulaLibraryVersion) []StudioLibrary {
	out := make([]StudioLibrary, 0, len(versions))
	for _, version := range versions {
		out = append(out, StudioLibrary{
			Path:          version.Path(),
			LibraryID:     version.LibraryID,
			Name:          version.Name,
			Description:   version.Description,
			VersionNumber: version.VersionNumber,
		})
	}
	return out
}

type ScaffoldTestRequest struct {
	TenantInfo pagination.TenantInfo
	Source     string
	Inputs     tarifftypes.Inputs
	ShipmentID pulid.ID
	Name       string
}

type ScaffoldTestResponse struct {
	Test string `json:"test"`
}

func (s *Service) ScaffoldTest(
	ctx context.Context,
	req *ScaffoldTestRequest,
) (*ScaffoldTestResponse, error) {
	if err := previewBounds(req.Source, req.Inputs); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	switch {
	case name == "":
		return nil, errortypes.NewValidationError("name", errortypes.ErrRequired, "Name the test")
	case utf8.RuneCountInString(name) > MaxTestNameLength:
		return nil, errortypes.NewValidationError(
			"name",
			errortypes.ErrInvalid,
			"A test name cannot be longer than 100 characters",
		)
	case req.ShipmentID.IsNil():
		return nil, errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrRequired,
			"Choose the shipment the test is written from",
		)
	}

	content, err := s.checkContent(ctx, &contentRequest{
		tenant: req.TenantInfo,
		source: req.Source,
		mode:   tariff.LibrariesLatest,
	})
	if err != nil {
		return nil, err
	}
	if refusal := content.refusal(); refusal != nil {
		return nil, refusal
	}

	shipment, err := s.shipmentRecord(ctx, req.TenantInfo, &req.ShipmentID)
	if err != nil {
		return nil, err
	}

	inputs := req.Inputs
	if inputs == nil {
		inputs = shipment.Inputs
	}
	scaffold, err := s.checker.ScaffoldTest(ctx, &tariff.ScaffoldRequest{
		TenantInfo: req.TenantInfo,
		Source:     req.Source,
		Context:    content.checked.Context,
		Record:     shipment.Record,
		Inputs:     map[string]rules.RawJSON(inputs),
		Name:       name,
	})
	if err != nil {
		return nil, err
	}
	if scaffold.Error != nil {
		return nil, tarifferrors.Evaluation(fieldSource, scaffold.Error)
	}
	if scaffold.Test == nil {
		return nil, errortypes.NewValidationError(
			fieldSource,
			errortypes.ErrInvalid,
			"The formula did not rate this shipment, so no test could be written from it",
		)
	}

	return &ScaffoldTestResponse{Test: *scaffold.Test}, nil
}

type TestSourceRequest struct {
	TenantInfo pagination.TenantInfo
	Source     string
}

type TestSourceResult struct {
	Check   *rules.TariffCheck      `json:"check"`
	Inputs  []tarifftypes.InputDecl `json:"inputs"`
	Imports []tarifftypes.Import    `json:"imports"`
	Tests   *rules.TariffTestRun    `json:"tests"`
}

func (s *Service) TestSource(
	ctx context.Context,
	req *TestSourceRequest,
) (*TestSourceResult, error) {
	if err := previewBounds(req.Source, nil); err != nil {
		return nil, err
	}

	content, err := s.checkContent(ctx, &contentRequest{
		tenant: req.TenantInfo,
		source: req.Source,
		mode:   tariff.LibrariesLatest,
	})
	if err != nil {
		return nil, err
	}

	return &TestSourceResult{
		Check:   content.checked.Check,
		Inputs:  tarifftypes.InputDeclsFrom(content.checked.Check.Inputs),
		Imports: content.checked.Imports,
		Tests:   content.tests,
	}, nil
}
