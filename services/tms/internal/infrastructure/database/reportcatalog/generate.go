package reportcatalog

//go:generate go run ./gen -manifest=reportcatalog.yml -domain=../../../modules/*/domain -enumpkgs=../../../../pkg/domaintypes -output=../../../../pkg/reportcatalog/catalog_gen.go
