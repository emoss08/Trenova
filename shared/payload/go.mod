module github.com/emoss08/trenova/shared/payload

go 1.24

require github.com/emoss08/trenova/shared/pulid v0.0.0-00010101000000-000000000000

require (
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/rotisserie/eris v0.5.4 // indirect
)

replace github.com/emoss08/trenova/shared/pulid => ../pulid
