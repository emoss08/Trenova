package documentpacketrule

type ResourceType string

const (
	ResourceTypeShipment = ResourceType("Shipment")
	ResourceTypeTrailer  = ResourceType("Trailer")
	ResourceTypeTractor  = ResourceType("Tractor")
	ResourceTypeWorker   = ResourceType("Worker")
)

type ItemStatus string

const (
	ItemStatusMissing      = ItemStatus("Missing")
	ItemStatusComplete     = ItemStatus("Complete")
	ItemStatusExpiringSoon = ItemStatus("ExpiringSoon")
	ItemStatusExpired      = ItemStatus("Expired")
	ItemStatusNeedsReview  = ItemStatus("NeedsReview")
)

type PacketStatus string

const (
	PacketStatusComplete     = PacketStatus("Complete")
	PacketStatusIncomplete   = PacketStatus("Incomplete")
	PacketStatusExpiringSoon = PacketStatus("ExpiringSoon")
	PacketStatusExpired      = PacketStatus("Expired")
	PacketStatusNeedsReview  = PacketStatus("NeedsReview")
)
