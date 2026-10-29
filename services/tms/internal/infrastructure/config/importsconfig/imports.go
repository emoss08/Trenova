package importsconfig

type Config struct {
	Spreadsheet SpreadsheetConfig `mapstructure:"spreadsheet"`
}

// SpreadsheetImportConfig bounds reading one uploaded rate sheet, fuel card
// statement or factoring remittance. A zero value takes the reader's default.
type SpreadsheetConfig struct {
	MaxRows          int   `mapstructure:"maxRows"          validate:"omitempty,min=1"`
	MaxColumns       int   `mapstructure:"maxColumns"       validate:"omitempty,min=1"`
	MaxCellBytes     int   `mapstructure:"maxCellBytes"     validate:"omitempty,min=1"`
	MaxSheetBytes    int   `mapstructure:"maxSheetBytes"    validate:"omitempty,min=1"`
	MaxUnzippedBytes int64 `mapstructure:"maxUnzippedBytes" validate:"omitempty,min=1"`
	MaxPartBytes     int64 `mapstructure:"maxPartBytes"     validate:"omitempty,min=1"`
}
