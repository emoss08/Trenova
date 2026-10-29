package edidocument

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const DefaultX12204Version = "004010"

type X12EnvelopeSettings struct {
	InterchangeSenderQualifier   string `json:"interchangeSenderQualifier"`
	InterchangeSenderID          string `json:"interchangeSenderId"`
	InterchangeReceiverQualifier string `json:"interchangeReceiverQualifier"`
	InterchangeReceiverID        string `json:"interchangeReceiverId"`
	ApplicationSenderCode        string `json:"applicationSenderCode"`
	ApplicationReceiverCode      string `json:"applicationReceiverCode"`
	InterchangeUsageIndicator    string `json:"interchangeUsageIndicator"`
	ElementSeparator             string `json:"elementSeparator"`
	SegmentTerminator            string `json:"segmentTerminator"`
	ComponentSeparator           string `json:"componentSeparator"`
	RepetitionSeparator          string `json:"repetitionSeparator"`
}

func DefaultX12EnvelopeSettings() X12EnvelopeSettings {
	return X12EnvelopeSettings{
		InterchangeSenderQualifier:   "ZZ",
		InterchangeSenderID:          "TRENOVA",
		InterchangeReceiverQualifier: "ZZ",
		InterchangeReceiverID:        "PARTNER",
		ApplicationSenderCode:        "TRENOVA",
		ApplicationReceiverCode:      "PARTNER",
		InterchangeUsageIndicator:    "T",
		ElementSeparator:             "*",
		SegmentTerminator:            "~",
		ComponentSeparator:           ">",
		RepetitionSeparator:          "^",
	}
}

type EDIDocumentType struct {
	keel.Table `keel:"edi_document_types"`

	ID               pulid.ID          `json:"id"               keel:"id,pk,autoid:edidt_"`
	Code             string            `json:"code"             keel:"code"`
	Name             string            `json:"name"             keel:"name"`
	Standard         EDIStandard       `json:"standard"         keel:"standard"`
	TransactionSet   TransactionSet    `json:"transactionSet"   keel:"transaction_set"`
	TransactionSetID pulid.ID          `json:"transactionSetId" keel:"transaction_set_id"`
	Direction        DocumentDirection `json:"direction"        keel:"direction"`
	DefaultVersion   string            `json:"defaultVersion"   keel:"default_version"`
	Status           DocumentStatus    `json:"status"           keel:"status"`
	CreatedAt        int64             `json:"createdAt"        keel:"created_at,autocreate"`
	UpdatedAt        int64             `json:"updatedAt"        keel:"updated_at,autoupdate"`

	TransactionSetRef *EDITransactionSet `json:"transactionSetRef,omitempty" keel:"belongs:transaction_set_id"`
}

func (t *EDIDocumentType) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(t,
		validation.Field(&t.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, maxEDICodeLength).
				Error("Code cannot be longer than 20 characters"),
		),
		validation.Field(&t.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxNameLength).
				Error("Name cannot be longer than 200 characters"),
		),
		validation.Field(&t.Standard,
			validation.Required.Error("Standard is required"),
			domainvalidation.ValidEnum[EDIStandard]("Standard is invalid"),
		),
		validation.Field(&t.TransactionSet,
			validation.Required.Error("Transaction set is required"),
			domainvalidation.ValidEnum[TransactionSet]("Transaction set is invalid"),
		),
		validation.Field(&t.TransactionSetID,
			validation.Required.Error("Transaction set definition is required"),
		),
		validation.Field(&t.Direction,
			validation.Required.Error("Direction is required"),
			domainvalidation.ValidEnum[DocumentDirection]("Direction is invalid"),
		),
		validation.Field(&t.DefaultVersion,
			validation.Required.Error("Default version is required"),
			validation.Length(1, maxEDICodeLength).
				Error("Default version cannot be longer than 20 characters"),
		),
		validation.Field(&t.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[DocumentStatus]("Status is invalid"),
		),
	))
}

const (
	maxEDICodeLength        = 20
	maxNameLength           = 200
	functionalGroupIDLength = 2
	maxLoopIDLength         = 50
)
