package rlslint

var allowed = map[string]string{
	"internal/infrastructure/postgres/capabilities.go:CapabilityProbe.probeLocked": "Reads the server's extension catalog, which belongs to no tenant",
	"internal/infrastructure/postgres/connection_rls.go:Connection.verifySystemRole": "Startup check that the system role is a bypass member",
}

var rawPoolAllowed = map[string]string{}
