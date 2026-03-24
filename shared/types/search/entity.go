package search

type EntityType string

const (
	EntityTypeShipment = EntityType("shipment")
	EntityTypeCustomer = EntityType("customer")
	EntityTypeWorker   = EntityType("worker")
	EntityTypeDocument = EntityType("document")
)
