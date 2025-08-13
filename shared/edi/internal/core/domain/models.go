package domain

import (
	"time"

	"github.com/emoss08/trenova/shared/edi/internal/profiles"
	"github.com/uptrace/bun"
)

type EDIDocument struct {
	bun.BaseModel `bun:"table:edi_documents,alias:ed"`

	ID             string     `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	PartnerID      string     `bun:"partner_id,type:varchar(100),notnull"`
	TransactionSet string     `bun:"transaction_set,type:varchar(10),notnull"`
	Version        string     `bun:"version,type:varchar(20),notnull"`
	ControlNumber  string     `bun:"control_number,type:varchar(50),notnull"`
	Direction      string     `bun:"direction,type:varchar(20),notnull"`
	Status         string     `bun:"status,type:varchar(20),notnull,default:'pending'"`
	RawContent     string     `bun:"raw_content,type:text,notnull"`
	ParsedContent  []byte     `bun:"parsed_content,type:jsonb"`
	ErrorMessages  []byte     `bun:"error_messages,type:jsonb"`
	ProcessedAt    *time.Time `bun:"processed_at"`
	CreatedAt      time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt      time.Time  `bun:"updated_at,notnull,default:current_timestamp"`

	Transactions    []*EDITransaction    `bun:"rel:has-many,join:id=document_id"`
	Acknowledgments []*EDIAcknowledgment `bun:"rel:has-many,join:id=document_id"`
}

type EDITransaction struct {
	bun.BaseModel `bun:"table:edi_transactions,alias:et"`

	ID               string    `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	DocumentID       string    `bun:"document_id,type:uuid,notnull"`
	TransactionType  string    `bun:"transaction_type,type:varchar(10),notnull"`
	ControlNumber    string    `bun:"control_number,type:varchar(50),notnull"`
	ReferenceID      string    `bun:"reference_id,type:varchar(100)"`
	Status           string    `bun:"status,type:varchar(20),notnull,default:'pending'"`
	Data             []byte    `bun:"data,type:jsonb,notnull"`
	ValidationErrors []byte    `bun:"validation_errors,type:jsonb"`
	CreatedAt        time.Time `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt        time.Time `bun:"updated_at,notnull,default:current_timestamp"`

	Document *EDIDocument `bun:"rel:belongs-to,join:document_id=id"`
	Shipment *EDIShipment `bun:"rel:has-one,join:id=transaction_id"`
}

type EDIShipment struct {
	bun.BaseModel `bun:"table:edi_shipments,alias:es"`

	ID            string     `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	TransactionID string     `bun:"transaction_id,type:uuid,notnull,unique"`
	ShipmentID    string     `bun:"shipment_id,type:varchar(100),notnull"`
	CarrierSCAC   string     `bun:"carrier_scac,type:varchar(10)"`
	PickupDate    *time.Time `bun:"pickup_date"`
	DeliveryDate  *time.Time `bun:"delivery_date"`
	TotalWeight   float64    `bun:"total_weight,type:decimal(10,2)"`
	TotalPieces   int        `bun:"total_pieces,type:integer"`
	ServiceLevel  string     `bun:"service_level,type:varchar(50)"`
	Status        string     `bun:"status,type:varchar(20),notnull,default:'pending'"`
	Data          []byte     `bun:"data,type:jsonb,notnull"`
	CreatedAt     time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt     time.Time  `bun:"updated_at,notnull,default:current_timestamp"`

	Transaction *EDITransaction `bun:"rel:belongs-to,join:transaction_id=id"`
	Stops       []*EDIStop      `bun:"rel:has-many,join:id=shipment_id"`
}

type EDIStop struct {
	bun.BaseModel `bun:"table:edi_stops,alias:est"`

	ID           string     `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	ShipmentID   string     `bun:"shipment_id,type:uuid,notnull"`
	StopNumber   int        `bun:"stop_number,type:integer,notnull"`
	StopType     string     `bun:"stop_type,type:varchar(20),notnull"`
	LocationName string     `bun:"location_name,type:varchar(200)"`
	Address      string     `bun:"address,type:varchar(200)"`
	City         string     `bun:"city,type:varchar(100)"`
	State        string     `bun:"state,type:varchar(2)"`
	PostalCode   string     `bun:"postal_code,type:varchar(20)"`
	Country      string     `bun:"country,type:varchar(3),default:'USA'"`
	EarliestDate *time.Time `bun:"earliest_date"`
	LatestDate   *time.Time `bun:"latest_date"`
	Data         []byte     `bun:"data,type:jsonb"`
	CreatedAt    time.Time  `bun:"created_at,notnull,default:current_timestamp"`

	Shipment *EDIShipment `bun:"rel:belongs-to,join:shipment_id=id"`
}

type EDIAcknowledgment struct {
	bun.BaseModel `bun:"table:edi_acknowledgments,alias:ea"`

	ID            string     `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	DocumentID    string     `bun:"document_id,type:uuid,notnull"`
	AckType       string     `bun:"ack_type,type:varchar(10),notnull"`
	ControlNumber string     `bun:"control_number,type:varchar(50),notnull"`
	Status        string     `bun:"status,type:varchar(20),notnull"`
	AckContent    string     `bun:"ack_content,type:text,notnull"`
	SentAt        *time.Time `bun:"sent_at"`
	CreatedAt     time.Time  `bun:"created_at,notnull,default:current_timestamp"`

	Document *EDIDocument `bun:"rel:belongs-to,join:document_id=id"`
}

type EDIPartnerProfile struct {
	bun.BaseModel `bun:"table:edi_partner_profiles,alias:epp"`

	ID            string    `json:"id"            bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	PartnerID     string    `json:"partner_id"    bun:"partner_id,type:varchar(100),notnull,unique"`
	PartnerName   string    `json:"partner_name"  bun:"partner_name,type:varchar(200),notnull"`
	Active        bool      `json:"active"        bun:"active,type:boolean,default:true"`
	Description   string    `json:"description"   bun:"description,type:text"`
	Configuration profiles.PartnerProfile `json:"configuration" bun:"configuration,type:jsonb,notnull"` // Full PartnerProfile struct as JSON
	CreatedAt     time.Time `json:"created_at"    bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt     time.Time `json:"updated_at"    bun:"updated_at,notnull,default:current_timestamp"`
}
