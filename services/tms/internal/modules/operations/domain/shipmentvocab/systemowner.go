package shipmentvocab

type SystemOwner string

const (
	SystemOwnerNone      = SystemOwner("")
	SystemOwnerFuel      = SystemOwner("FuelSurcharge")
	SystemOwnerDetention = SystemOwner("Detention")
	SystemOwnerTariff    = SystemOwner("Tariff")
	SystemOwnerAgreement = SystemOwner("RateAgreement")
)
