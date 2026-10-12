package documentpacketrule

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
