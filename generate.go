//go:build generate

package generate

//go:generate go run cmd/services/generate_services.go -root . -models pkg/models -services services
//go:generate go run cmd/handlers/generate_handlers.go -root . -models pkg/models -handlers handlers -services services
