package validation

// Strictness controls how strictly to enforce rules.
type Strictness string

const (
    Strict Strictness = "strict"
    Lenient Strictness = "lenient"
)

// Profile controls validation behavior and requiredness per partner/version.
type Profile struct {
    Version                  string
    Strictness               Strictness
    RequireB2ShipID          bool
    RequireN1SH              bool
    RequireN1ST              bool
    RequirePickupAndDelivery bool
    EnforceSECount           bool // when false, SE count mismatches produce warnings
}

// DefaultProfileForVersion returns baseline requiredness based on common guides.
// - 004010: require B2-03 shipment id; enforce SE count; require both pickup and delivery.
// - 005010/006010: B2-03 optional; enforce SE count; require both pickup and delivery.
func DefaultProfileForVersion(ver string) Profile {
    p := Profile{
        Version:                  ver,
        Strictness:               Strict,
        RequireB2ShipID:          false,
        RequireN1SH:              false,
        RequireN1ST:              false,
        RequirePickupAndDelivery: false,
        EnforceSECount:           true,
    }
    if len(ver) >= 3 && (ver[:3] == "005" || ver[:3] == "006") {
        p.RequireB2ShipID = false
    }
    return p
}
