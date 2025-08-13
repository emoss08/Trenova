package services

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/edi/internal/core/domain"
	"github.com/emoss08/trenova/shared/edi/internal/core/ports"
	"github.com/emoss08/trenova/shared/edi/internal/dto"
	"github.com/emoss08/trenova/shared/edi/internal/services"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type EDIProcessorService struct {
	logger          *zap.Logger
	docRepo         ports.EDIDocumentRepository
	txRepo          ports.EDITransactionRepository
	shipmentRepo    ports.EDIShipmentRepository
	profileService  *ProfileService
	parser          *services.IntegratedParser
}

type EDIProcessorParams struct {
	fx.In
	Logger         *zap.Logger
	DocRepo        ports.EDIDocumentRepository
	TxRepo         ports.EDITransactionRepository
	ShipmentRepo   ports.EDIShipmentRepository
	ProfileService *ProfileService
}

func NewEDIProcessorService(params EDIProcessorParams) *EDIProcessorService {
	return &EDIProcessorService{
		logger:         params.Logger,
		docRepo:        params.DocRepo,
		txRepo:         params.TxRepo,
		shipmentRepo:   params.ShipmentRepo,
		profileService: params.ProfileService,
		parser:         nil, // Will be initialized with proper options
	}
}

func (s *EDIProcessorService) ProcessEDIDocument(ctx context.Context, partnerID string, ediContent string) (*domain.EDIDocument, error) {
	s.logger.Info("processing EDI document",
		zap.String("partner_id", partnerID),
		zap.Int("content_length", len(ediContent)),
	)

	// Initialize parser if not already done
	if s.parser == nil {
		parser, err := services.NewIntegratedParser(services.IntegratedParserOptions{
			StrictMode: false,
			AutoAck:    false,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize parser: %w", err)
		}
		s.parser = parser
	}

	req := services.ParseRequest{
		Data:            []byte(ediContent),
		PartnerID:       partnerID,
		ValidateContent: true,
		GenerateAck:     false,
	}

	result, err := s.parser.Parse(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to parse EDI: %w", err)
	}

	var version, controlNumber, transactionSet string
	if result.Document != nil {
		version = result.Document.Metadata.Version
		controlNumber = result.Document.Metadata.ISAControlNumber
		transactionSet = result.Document.Metadata.TransactionType
	}
	if transactionSet == "" {
		transactionSet = "204"
	}

	doc := &domain.EDIDocument{
		PartnerID:      partnerID,
		TransactionSet: transactionSet,
		Version:        version,
		ControlNumber:  controlNumber,
		Direction:      "inbound",
		Status:         "processed",
		RawContent:     ediContent,
	}

	if result.Document != nil {
		parsedJSON, _ := sonic.Marshal(result.Document)
		doc.ParsedContent = parsedJSON
	}

	if len(result.ValidationIssues) > 0 {
		doc.Status = "processed_with_errors"
		errorJSON, _ := sonic.Marshal(result.ValidationIssues)
		doc.ErrorMessages = errorJSON
	}

	now := time.Now()
	doc.ProcessedAt = &now

	if err := s.docRepo.Create(ctx, doc); err != nil {
		return nil, fmt.Errorf("failed to save EDI document: %w", err)
	}

	// For now, we'll skip transaction processing as we need to implement
	// proper conversion from Document to Shipment DTO
	// TODO: Implement document to shipment conversion
	s.logger.Info("EDI document processed successfully",
		zap.String("document_id", doc.ID),
		zap.String("status", doc.Status),
	)

	return doc, nil
}

func (s *EDIProcessorService) processTransactions(ctx context.Context, doc *domain.EDIDocument, parsedData any) error {
	switch data := parsedData.(type) {
	case *dto.Shipment:
		return s.processSingleShipment(ctx, doc, data)
	case []*dto.Shipment:
		for _, shipment := range data {
			if err := s.processSingleShipment(ctx, doc, shipment); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *EDIProcessorService) processSingleShipment(ctx context.Context, doc *domain.EDIDocument, shipment *dto.Shipment) error {
	shipmentJSON, err := sonic.Marshal(shipment)
	if err != nil {
		return fmt.Errorf("failed to marshal shipment: %w", err)
	}

	tx := &domain.EDITransaction{
		DocumentID:      doc.ID,
		TransactionType: "204",
		ControlNumber:   doc.ControlNumber,
		ReferenceID:     shipment.ShipmentID,
		Status:          "processed",
		Data:            shipmentJSON,
	}

	if err := s.txRepo.Create(ctx, tx); err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	ediShipment := &domain.EDIShipment{
		TransactionID: tx.ID,
		ShipmentID:    shipment.ShipmentID,
		CarrierSCAC:   shipment.CarrierSCAC,
		ServiceLevel:  shipment.ServiceLevel,
		Status:        "pending",
		Data:          shipmentJSON,
	}

	// Parse weight from string if available
	if shipment.Totals.Weight != "" {
		// For now, store as 0 if parsing fails
		var weight float64
		if _, err := fmt.Sscanf(shipment.Totals.Weight, "%f", &weight); err == nil {
			ediShipment.TotalWeight = weight
		}
	}
	ediShipment.TotalPieces = shipment.Totals.Pieces

	if err := s.shipmentRepo.Create(ctx, ediShipment); err != nil {
		return fmt.Errorf("failed to create shipment: %w", err)
	}

	for i, stop := range shipment.Stops {
		ediStop := &domain.EDIStop{
			ShipmentID:   ediShipment.ID,
			StopNumber:   i + 1,
			StopType:     stop.Type,
			LocationName: stop.Location.Name,
			Address:      stop.Location.Address1,
			City:         stop.Location.City,
			State:        stop.Location.State,
			PostalCode:   stop.Location.PostalCode,
			Country:      stop.Location.Country,
		}

		// Parse appointment dates if available
		if len(stop.Appointments) > 0 {
			for _, appt := range stop.Appointments {
				if appt.Date != "" {
					// Try to parse the date - for now, we'll just store the first appointment
					if parsedTime, err := time.Parse("20060102", appt.Date); err == nil {
						if ediStop.EarliestDate == nil {
							ediStop.EarliestDate = &parsedTime
						} else {
							ediStop.LatestDate = &parsedTime
						}
					}
				}
			}
		}

		stopJSON, _ := sonic.Marshal(stop)
		ediStop.Data = stopJSON

		if err := s.shipmentRepo.CreateStop(ctx, ediStop); err != nil {
			return fmt.Errorf("failed to create stop: %w", err)
		}
	}

	return nil
}

func (s *EDIProcessorService) GetDocumentByID(ctx context.Context, id string) (*domain.EDIDocument, error) {
	return s.docRepo.GetByID(ctx, id)
}

func (s *EDIProcessorService) ListDocuments(ctx context.Context, partnerID string, limit, offset int) ([]*domain.EDIDocument, error) {
	return s.docRepo.List(ctx, partnerID, limit, offset)
}