package edirepository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

func (r *repository) ListDocumentTypes(
	ctx context.Context,
	req repositories.ListEDIDocumentTypesRequest,
) ([]*edi.EDIDocumentType, error) {
	entities := make([]*edi.EDIDocumentType, 0, 8)
	query := r.db.DBForContext(ctx).NewSelect().Model(&entities).Order("edt.code ASC")
	if req.Standard != "" {
		query = query.Where("edt.standard = ?", req.Standard)
	}
	if req.TransactionSet != "" {
		query = query.Where("edt.transaction_set = ?", req.TransactionSet)
	}
	if req.Direction != "" {
		query = query.Where("edt.direction = ?", req.Direction)
	}
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}
	return entities, nil
}

func (r *repository) ListTemplates(
	ctx context.Context,
	req *repositories.ListEDITemplatesRequest,
) (*pagination.ListResult[*edi.EDITemplate], error) {
	entities := make([]*edi.EDITemplate, 0, req.Filter.Pagination.SafeLimit())
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation("DocumentType").
		Where("et.organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("et.business_unit_id = ?", req.Filter.TenantInfo.BuID)
	if req.TransactionSet != "" {
		query = query.Where("et.transaction_set = ?", req.TransactionSet)
	}
	if req.Direction != "" {
		query = query.Where("et.direction = ?", req.Direction)
	}
	total, err := query.
		Order("et.created_at DESC").
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}
	return &pagination.ListResult[*edi.EDITemplate]{Items: entities, Total: total}, nil
}

func (r *repository) GetTemplateByID(
	ctx context.Context,
	req repositories.GetEDITemplateByIDRequest,
) (*edi.EDITemplate, error) {
	entity := new(edi.EDITemplate)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("DocumentType").
		Relation("Versions", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("version_number DESC")
		}).
		Where("et.id = ?", req.ID).
		Where("et.organization_id = ?", req.TenantInfo.OrgID).
		Where("et.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDITemplate")
	}
	return entity, nil
}

func (r *repository) GetActiveTemplateVersion(
	ctx context.Context,
	req repositories.GetActiveEDITemplateVersionRequest,
) (*edi.EDITemplateVersion, error) {
	entity := new(edi.EDITemplateVersion)
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("Segments", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("sequence ASC")
		}).
		Where("etv.template_id = ?", req.TemplateID).
		Where("etv.organization_id = ?", req.TenantInfo.OrgID).
		Where("etv.business_unit_id = ?", req.TenantInfo.BuID)
	if !req.VersionID.IsNil() {
		query = query.Where("etv.id = ?", req.VersionID)
	} else {
		query = query.Where("etv.is_active = TRUE")
	}
	if err := query.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDITemplateVersion")
	}
	return entity, nil
}

func (r *repository) EnsureBase204Template(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*edi.EDITemplate, *edi.EDITemplateVersion, error) {
	template := new(edi.EDITemplate)
	version := new(edi.EDITemplateVersion)
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		err := r.db.DBForContext(c).
			NewSelect().
			Model(template).
			Where("et.organization_id = ?", tenantInfo.OrgID).
			Where("et.business_unit_id = ?", tenantInfo.BuID).
			Where("et.standard = ?", edi.EDIStandardX12).
			Where("et.transaction_set = ?", edi.TransactionSet204).
			Where("et.direction = ?", edi.DocumentDirectionOutbound).
			Where("et.name = ?", "Base X12 204 Outbound").
			Limit(1).
			Scan(c)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == nil {
			existing, versionErr := r.GetActiveTemplateVersion(c, repositories.GetActiveEDITemplateVersionRequest{
				TemplateID: template.ID,
				TenantInfo: tenantInfo,
			})
			if versionErr != nil {
				return versionErr
			}
			*version = *existing
			return nil
		}

		documentTypes, err := r.ListDocumentTypes(c, repositories.ListEDIDocumentTypesRequest{
			Standard:       edi.EDIStandardX12,
			TransactionSet: edi.TransactionSet204,
			Direction:      edi.DocumentDirectionOutbound,
		})
		if err != nil {
			return err
		}
		if len(documentTypes) == 0 {
			return fmt.Errorf("x12 204 outbound document type is not seeded")
		}

		template = &edi.EDITemplate{
			BusinessUnitID: tenantInfo.BuID,
			OrganizationID: tenantInfo.OrgID,
			DocumentTypeID: documentTypes[0].ID,
			Name:           "Base X12 204 Outbound",
			Description:    "Tenant-scoped base outbound X12 204 template",
			Direction:      edi.DocumentDirectionOutbound,
			Standard:       edi.EDIStandardX12,
			TransactionSet: edi.TransactionSet204,
			Status:         edi.TemplateStatusActive,
		}
		if _, err = r.db.DBForContext(c).NewInsert().Model(template).Returning("*").Exec(c); err != nil {
			return err
		}

		version = &edi.EDITemplateVersion{
			BusinessUnitID:    tenantInfo.BuID,
			OrganizationID:    tenantInfo.OrgID,
			TemplateID:        template.ID,
			VersionNumber:     1,
			X12Version:        edi.DefaultX12204Version,
			FunctionalGroupID: "SM",
			Status:            edi.TemplateStatusActive,
			IsActive:          true,
			Notes:             "Seeded base 004010 Motor Carrier Load Tender profile",
		}
		if _, err = r.db.DBForContext(c).NewInsert().Model(version).Returning("*").Exec(c); err != nil {
			return err
		}

		segments := base204TemplateSegments(tenantInfo, version.ID)
		if _, err = r.db.DBForContext(c).NewInsert().Model(&segments).Exec(c); err != nil {
			return err
		}
		version.Segments = segments
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return template, version, nil
}

func (r *repository) ListPartnerDocumentProfiles(
	ctx context.Context,
	req *repositories.ListEDIPartnerDocumentProfilesRequest,
) (*pagination.ListResult[*edi.EDIPartnerDocumentProfile], error) {
	entities := make([]*edi.EDIPartnerDocumentProfile, 0, req.Filter.Pagination.SafeLimit())
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation("Partner").
		Relation("DocumentType").
		Relation("Template").
		Where("epdp.organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("epdp.business_unit_id = ?", req.Filter.TenantInfo.BuID)
	if req.TransactionSet != "" {
		query = query.Where("epdp.transaction_set = ?", req.TransactionSet)
	}
	if req.Direction != "" {
		query = query.Where("epdp.direction = ?", req.Direction)
	}
	total, err := query.
		Order("epdp.created_at DESC").
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}
	return &pagination.ListResult[*edi.EDIPartnerDocumentProfile]{Items: entities, Total: total}, nil
}

func (r *repository) GetPartnerDocumentProfileByID(
	ctx context.Context,
	req repositories.GetEDIPartnerDocumentProfileByIDRequest,
) (*edi.EDIPartnerDocumentProfile, error) {
	entity := new(edi.EDIPartnerDocumentProfile)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("Partner").
		Relation("DocumentType").
		Relation("Template").
		Relation("TemplateVersion").
		Where("epdp.id = ?", req.ID).
		Where("epdp.organization_id = ?", req.TenantInfo.OrgID).
		Where("epdp.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDIPartnerDocumentProfile")
	}
	return entity, nil
}

func (r *repository) GetActivePartnerDocumentProfile(
	ctx context.Context,
	req repositories.GetActiveEDIPartnerDocumentProfileRequest,
) (*edi.EDIPartnerDocumentProfile, error) {
	entity := new(edi.EDIPartnerDocumentProfile)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("DocumentType").
		Relation("Template").
		Where("epdp.edi_partner_id = ?", req.PartnerID).
		Where("epdp.organization_id = ?", req.TenantInfo.OrgID).
		Where("epdp.business_unit_id = ?", req.TenantInfo.BuID).
		Where("epdp.transaction_set = ?", req.TransactionSet).
		Where("epdp.direction = ?", req.Direction).
		Where("epdp.status = ?", edi.DocumentStatusActive).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDIPartnerDocumentProfile")
	}
	return entity, nil
}

func (r *repository) CreatePartnerDocumentProfile(
	ctx context.Context,
	entity *edi.EDIPartnerDocumentProfile,
) (*edi.EDIPartnerDocumentProfile, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) UpdatePartnerDocumentProfile(
	ctx context.Context,
	entity *edi.EDIPartnerDocumentProfile,
) (*edi.EDIPartnerDocumentProfile, error) {
	ov := entity.Version
	entity.Version++
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		Column(
			"template_id",
			"template_version_id",
			"name",
			"status",
			"x12_version_override",
			"functional_group_id",
			"envelope",
			"acknowledgment",
			"validation_mode",
			"partner_settings",
			"version",
			"updated_at",
		).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, "EDIPartnerDocumentProfile", entity.ID.String()); err != nil {
		return nil, err
	}
	return entity, nil
}

func (r *repository) AllocateControlNumbers(
	ctx context.Context,
	req repositories.AllocateEDIControlNumbersRequest,
) (map[edi.ControlNumberKind]int64, error) {
	allocated := make(map[edi.ControlNumberKind]int64, len(req.Kinds))
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		for _, kind := range req.Kinds {
			sequence := &edi.EDIControlNumberSequence{
				BusinessUnitID: req.TenantInfo.BuID,
				OrganizationID: req.TenantInfo.OrgID,
				EDIPartnerID:   req.PartnerID,
				DocumentTypeID: req.DocumentTypeID,
				Kind:           kind,
			}
			_, err := r.db.DBForContext(c).
				NewInsert().
				Model(sequence).
				On(`CONFLICT ("edi_partner_id", "business_unit_id", "organization_id", "document_type_id", "kind") DO NOTHING`).
				Exec(c)
			if err != nil {
				return err
			}

			if err = r.db.DBForContext(c).
				NewSelect().
				Model(sequence).
				Where("ecns.edi_partner_id = ?", req.PartnerID).
				Where("ecns.business_unit_id = ?", req.TenantInfo.BuID).
				Where("ecns.organization_id = ?", req.TenantInfo.OrgID).
				Where("ecns.document_type_id = ?", req.DocumentTypeID).
				Where("ecns.kind = ?", kind).
				For("UPDATE").
				Scan(c); err != nil {
				return err
			}

			value := sequence.NextValue
			next := value + 1
			if next > sequence.MaxValue {
				next = sequence.MinValue
			}
			sequence.NextValue = next
			sequence.Version++
			if _, err = r.db.DBForContext(c).
				NewUpdate().
				Model(sequence).
				WherePK().
				Column("next_value", "version", "updated_at").
				Exec(c); err != nil {
				return err
			}
			allocated[kind] = value
		}
		return nil
	})
	return allocated, err
}

func (r *repository) ListMessages(
	ctx context.Context,
	req *repositories.ListEDIMessagesRequest,
) (*pagination.ListResult[*edi.EDIMessage], error) {
	entities := make([]*edi.EDIMessage, 0, req.Filter.Pagination.SafeLimit())
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Relation("PartnerDocumentProfile").
		Where("emsg.organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("emsg.business_unit_id = ?", req.Filter.TenantInfo.BuID)
	if req.TransactionSet != "" {
		query = query.Where("emsg.transaction_set = ?", req.TransactionSet)
	}
	if req.Direction != "" {
		query = query.Where("emsg.direction = ?", req.Direction)
	}
	total, err := query.
		Order("emsg.generated_at DESC").
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}
	return &pagination.ListResult[*edi.EDIMessage]{Items: entities, Total: total}, nil
}

func (r *repository) GetMessageByID(
	ctx context.Context,
	req repositories.GetEDIMessageByIDRequest,
) (*edi.EDIMessage, error) {
	entity := new(edi.EDIMessage)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Relation("PartnerDocumentProfile").
		Relation("ValidationErrors", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("created_at ASC")
		}).
		Where("emsg.id = ?", req.ID).
		Where("emsg.organization_id = ?", req.TenantInfo.OrgID).
		Where("emsg.business_unit_id = ?", req.TenantInfo.BuID).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "EDIMessage")
	}
	return entity, nil
}

func (r *repository) CreateMessageWithDiagnostics(
	ctx context.Context,
	req repositories.CreateEDIMessageWithDiagnosticsRequest,
) (*edi.EDIMessage, error) {
	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		if _, err := r.db.DBForContext(c).NewInsert().Model(req.Message).Returning("*").Exec(c); err != nil {
			return err
		}
		for _, diagnostic := range req.Diagnostics {
			diagnostic.MessageID = req.Message.ID
			diagnostic.BusinessUnitID = req.Message.BusinessUnitID
			diagnostic.OrganizationID = req.Message.OrganizationID
		}
		if len(req.Diagnostics) > 0 {
			if _, err := r.db.DBForContext(c).NewInsert().Model(&req.Diagnostics).Exec(c); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	req.Message.ValidationErrors = req.Diagnostics
	return req.Message, nil
}

func (r *repository) ListTestCases(
	ctx context.Context,
	req *repositories.ListEDITestCasesRequest,
) (*pagination.ListResult[*edi.EDITestCase], error) {
	entities := make([]*edi.EDITestCase, 0, req.Filter.Pagination.SafeLimit())
	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		Where("etc.organization_id = ?", req.Filter.TenantInfo.OrgID).
		Where("etc.business_unit_id = ?", req.Filter.TenantInfo.BuID)
	if !req.PartnerDocumentProfileID.IsNil() {
		query = query.Where("etc.partner_document_profile_id = ?", req.PartnerDocumentProfileID)
	}
	total, err := query.
		Order("etc.created_at DESC").
		Limit(req.Filter.Pagination.SafeLimit()).
		Offset(req.Filter.Pagination.SafeOffset()).
		ScanAndCount(ctx)
	if err != nil {
		return nil, err
	}
	return &pagination.ListResult[*edi.EDITestCase]{Items: entities, Total: total}, nil
}

func (r *repository) CreateTestCase(
	ctx context.Context,
	entity *edi.EDITestCase,
) (*edi.EDITestCase, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		return nil, err
	}
	return entity, nil
}

func base204TemplateSegments(
	tenantInfo pagination.TenantInfo,
	versionID pulid.ID,
) []*edi.EDITemplateSegment {
	segment := func(sequence int64, id, name, repeatPath string, required bool, elements []edi.TemplateElement) *edi.EDITemplateSegment {
		return &edi.EDITemplateSegment{
			BusinessUnitID:    tenantInfo.BuID,
			OrganizationID:    tenantInfo.OrgID,
			TemplateVersionID: versionID,
			SegmentID:         id,
			Name:              name,
			Sequence:          sequence,
			RepeatPath:        repeatPath,
			Required:          required,
			MaxUse:            1,
			Elements:          elements,
		}
	}
	el := func(position int, name string, source edi.TemplateElementSource, value string, required bool) edi.TemplateElement {
		return edi.TemplateElement{
			Position: position,
			Name:     name,
			Source:   source,
			Value:    value,
			Validation: edi.TemplateValidationRule{
				Required: required,
				Code:     "required",
				Message:  name + " is required",
			},
		}
	}
	field := func(position int, name, path, fallback string, required bool) edi.TemplateElement {
		element := el(position, name, edi.TemplateElementSourceFieldPath, "", required)
		element.FieldPath = path
		element.Default = fallback
		return element
	}
	runtime := func(position int, name, key string, required bool) edi.TemplateElement {
		element := el(position, name, edi.TemplateElementSourceRuntime, "", required)
		element.RuntimeKey = key
		return element
	}
	return []*edi.EDITemplateSegment{
		segment(10, "ISA", "Interchange Control Header", "", true, []edi.TemplateElement{
			el(1, "Authorization Information Qualifier", edi.TemplateElementSourceConstant, "00", true),
			el(2, "Authorization Information", edi.TemplateElementSourceConstant, "          ", true),
			el(3, "Security Information Qualifier", edi.TemplateElementSourceConstant, "00", true),
			el(4, "Security Information", edi.TemplateElementSourceConstant, "          ", true),
			el(5, "Interchange ID Qualifier", edi.TemplateElementSourceConstant, "ZZ", true),
			runtime(6, "Interchange Sender ID", "interchangeSenderId", true),
			el(7, "Interchange ID Qualifier", edi.TemplateElementSourceConstant, "ZZ", true),
			runtime(8, "Interchange Receiver ID", "interchangeReceiverId", true),
			runtime(9, "Interchange Date", "interchangeDate", true),
			runtime(10, "Interchange Time", "interchangeTime", true),
			runtime(11, "Repetition Separator", "repetitionSeparator", true),
			el(12, "Interchange Control Version", edi.TemplateElementSourceConstant, "00401", true),
			runtime(13, "Interchange Control Number", "isaControlNumber", true),
			el(14, "Acknowledgment Requested", edi.TemplateElementSourceConstant, "0", true),
			runtime(15, "Usage Indicator", "usageIndicator", true),
			runtime(16, "Component Separator", "componentSeparator", true),
		}),
		segment(20, "GS", "Functional Group Header", "", true, []edi.TemplateElement{
			runtime(1, "Functional Identifier Code", "functionalGroupId", true),
			runtime(2, "Application Sender Code", "applicationSenderCode", true),
			runtime(3, "Application Receiver Code", "applicationReceiverCode", true),
			runtime(4, "Group Date", "groupDate", true),
			runtime(5, "Group Time", "groupTime", true),
			runtime(6, "Group Control Number", "groupControlNumber", true),
			el(7, "Responsible Agency Code", edi.TemplateElementSourceConstant, "X", true),
			runtime(8, "Version", "x12Version", true),
		}),
		segment(30, "ST", "Transaction Set Header", "", true, []edi.TemplateElement{
			el(1, "Transaction Set Identifier", edi.TemplateElementSourceConstant, "204", true),
			runtime(2, "Transaction Control Number", "transactionControlNumber", true),
		}),
		segment(40, "B2", "Beginning Segment for Shipment Information", "", true, []edi.TemplateElement{
			el(1, "Standard Carrier Alpha Code", edi.TemplateElementSourcePartnerSetting, "", false),
			field(2, "Shipment Identification Number", "shipmentId", "", true),
			field(4, "Shipment Method of Payment", "ratingDetail.paymentMethod", "PP", false),
		}),
		segment(50, "B2A", "Set Purpose", "", true, []edi.TemplateElement{
			el(1, "Transaction Set Purpose Code", edi.TemplateElementSourceConstant, "00", true),
		}),
		segment(60, "L11", "Reference Identification", "", false, []edi.TemplateElement{
			field(1, "Reference Identification", "bol", "", false),
			el(2, "Reference Identification Qualifier", edi.TemplateElementSourceConstant, "BM", false),
		}),
		segment(70, "G62", "Date Time", "moves.0.stops", false, []edi.TemplateElement{
			el(1, "Date Qualifier", edi.TemplateElementSourceConstant, "37", false),
			field(2, "Date", "repeat.scheduledWindowStart", "", false),
			el(3, "Time Qualifier", edi.TemplateElementSourceConstant, "I", false),
			field(4, "Time", "repeat.scheduledWindowStart", "", false),
		}),
		segment(80, "NTE", "Note", "", false, []edi.TemplateElement{
			el(1, "Note Reference Code", edi.TemplateElementSourceConstant, "ADD", false),
			field(2, "Description", "ratingDetail.note", "", false),
		}),
		segment(90, "N1", "Name", "moves.0.stops", false, []edi.TemplateElement{
			field(1, "Entity Identifier Code", "repeat.type", "SF", false),
			field(2, "Name", "repeat.locationName", "", false),
		}),
		segment(100, "N3", "Address", "moves.0.stops", false, []edi.TemplateElement{
			field(1, "Address Information", "repeat.locationAddressLine1", "", false),
			field(2, "Address Information", "repeat.locationAddressLine2", "", false),
		}),
		segment(110, "N4", "Geographic Location", "moves.0.stops", false, []edi.TemplateElement{
			field(1, "City Name", "repeat.locationCity", "", false),
			field(2, "State or Province Code", "repeat.locationStateCode", "", false),
			field(3, "Postal Code", "repeat.locationPostalCode", "", false),
		}),
		segment(120, "G61", "Contact", "", false, []edi.TemplateElement{
			el(1, "Contact Function Code", edi.TemplateElementSourceConstant, "IC", false),
			el(2, "Name", edi.TemplateElementSourcePartnerSetting, "", false),
			el(3, "Communication Number Qualifier", edi.TemplateElementSourceConstant, "TE", false),
			el(4, "Communication Number", edi.TemplateElementSourcePartnerSetting, "", false),
		}),
		segment(130, "S5", "Stop Off Details", "moves.0.stops", true, []edi.TemplateElement{
			field(1, "Stop Sequence Number", "repeat.sequence", "", true),
			field(2, "Stop Reason Code", "repeat.type", "LD", true),
			field(3, "Weight", "repeat.weight", "", false),
			el(4, "Weight Unit Code", edi.TemplateElementSourceConstant, "L", false),
			field(5, "Number of Units Shipped", "repeat.pieces", "", false),
			el(6, "Unit or Basis for Measurement Code", edi.TemplateElementSourceConstant, "PCS", false),
		}),
		segment(140, "AT8", "Shipment Weight Packaging and Quantity Data", "", false, []edi.TemplateElement{
			el(1, "Weight Qualifier", edi.TemplateElementSourceConstant, "G", false),
			el(2, "Weight Unit Code", edi.TemplateElementSourceConstant, "L", false),
			field(3, "Weight", "weight", "", false),
			field(4, "Lading Quantity", "pieces", "", false),
		}),
		segment(150, "L5", "Description Marks and Numbers", "commodities", false, []edi.TemplateElement{
			field(1, "Lading Line Item Number", "repeat.sequence", "", false),
			field(2, "Lading Description", "repeat.commodityDescription", "", false),
		}),
		segment(160, "L3", "Total Weight and Charges", "", false, []edi.TemplateElement{
			field(1, "Weight", "weight", "", false),
			el(2, "Weight Qualifier", edi.TemplateElementSourceConstant, "G", false),
			field(5, "Charge", "totalChargeAmount", "", false),
		}),
		segment(170, "SE", "Transaction Set Trailer", "", true, []edi.TemplateElement{
			runtime(1, "Segment Count", "transactionSegmentCount", true),
			runtime(2, "Transaction Control Number", "transactionControlNumber", true),
		}),
		segment(180, "GE", "Functional Group Trailer", "", true, []edi.TemplateElement{
			el(1, "Number of Transaction Sets", edi.TemplateElementSourceConstant, "1", true),
			runtime(2, "Group Control Number", "groupControlNumber", true),
		}),
		segment(190, "IEA", "Interchange Control Trailer", "", true, []edi.TemplateElement{
			el(1, "Number of Functional Groups", edi.TemplateElementSourceConstant, "1", true),
			runtime(2, "Interchange Control Number", "isaControlNumber", true),
		}),
	}
}
