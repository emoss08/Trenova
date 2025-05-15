package billingclientprovider

type ShipmentReadyToBillCard struct {
	Count int `json:"count"`
}

type CompletedShipmentsWithNoDocumentsCard struct {
	Count int `json:"count"`
}
