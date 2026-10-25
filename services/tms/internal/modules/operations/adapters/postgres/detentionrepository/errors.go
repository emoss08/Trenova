package detentionrepository

import "github.com/emoss08/trenova/pkg/dberror"

var occurrenceErrors = dberror.Mapping{
	Entity: "DetentionOccurrence",
	Unique: []dberror.Constraint{
		{
			Name:    "uq_detention_occurrences_stop",
			Field:   "stopId",
			Message: "Stop already has a detention occurrence",
		},
	},
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_detention_occurrences_shipment",
			Field:   "shipmentId",
			Message: "Shipment does not exist in your organization",
		},
		{
			Name:    "fk_detention_occurrences_policy",
			Field:   "detentionPolicyId",
			Message: "Detention policy does not exist in your organization",
		},
		{
			Name:    "fk_detention_occurrences_additional_charge",
			Field:   "additionalChargeId",
			Message: "Additional charge does not exist",
		},
	},
}

var evidenceErrors = dberror.Mapping{
	Entity: "DetentionEvidence",
	Unique: []dberror.Constraint{
		{
			Name:    "uq_detention_evidence_sequence",
			Field:   "sequence",
			Message: "Detention evidence was appended concurrently; retry",
		},
	},
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_detention_evidence_occurrence",
			Field:   "detentionOccurrenceId",
			Message: "Detention occurrence does not exist in your organization",
		},
	},
}

var noticeErrors = dberror.Mapping{
	Entity: "DetentionNotice",
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_detention_notices_occurrence",
			Field:   "detentionOccurrenceId",
			Message: "Detention occurrence does not exist in your organization",
		},
		{
			Name:    "fk_detention_notices_template_version",
			Field:   "templateVersionId",
			Message: "Document template version does not exist in your organization",
		},
		{
			Name:    "fk_detention_notices_pdf_document",
			Field:   "pdfDocumentId",
			Message: "Document does not exist in your organization",
		},
	},
}

var policyErrors = dberror.Mapping{
	Entity: "DetentionPolicy",
	Unique: []dberror.Constraint{
		{
			Name:    "uq_detention_policies_code",
			Field:   "code",
			Message: "Detention policy with this code already exists in your organization",
		},
		{
			Name:    "uq_detention_policies_org_default",
			Field:   "isOrgDefault",
			Message: "Your organization already has a default detention policy",
		},
	},
	ForeignKey: []dberror.Constraint{
		{
			Name:    "fk_detention_policies_accessorial",
			Field:   "accessorialChargeId",
			Message: "Accessorial charge does not exist in your organization",
		},
		{
			Name:    "fk_detention_policies_layover_accessorial",
			Field:   "layoverAccessorialChargeId",
			Message: "Layover accessorial charge does not exist in your organization",
		},
	},
}
