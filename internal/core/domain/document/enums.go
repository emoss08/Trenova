package document

// DocumentType categorize the document
type DocumentType string

const (
	DocumentTypeLicense         = DocumentType("License")
	DocumentTypeRegistration    = DocumentType("Registration")
	DocumentTypeInsurance       = DocumentType("Insurance")
	DocumentTypeInvoice         = DocumentType("Invoice")
	DocumentTypeProofOfDelivery = DocumentType("ProofOfDelivery")
	DocumentTypeBillOfLading    = DocumentType("BillOfLading")
	DocumentTypeDriverLog       = DocumentType("DriverLog")
	DocumentTypeMedicalCert     = DocumentType("MedicalCert")
	DocumentTypeContract        = DocumentType("Contract")
	DocumentTypeMaintenance     = DocumentType("Maintenance")
	DocumentTypeAccidentReport  = DocumentType("AccidentReport")
	DocumentTypeTrainingRecord  = DocumentType("TrainingRecord")
	DocumentTypeOther           = DocumentType("Other")
)

// DocumentStatus represents the current status of a document
type DocumentStatus string

const (
	DocumentStatusDraft           = DocumentStatus("Draft")
	DocumentStatusActive          = DocumentStatus("Active")
	DocumentStatusArchived        = DocumentStatus("Archived")
	DocumentStatusExpired         = DocumentStatus("Expired")
	DocumentStatusRejected        = DocumentStatus("Rejected")
	DocumentStatusPendingApproval = DocumentStatus("PendingApproval")
)
