package integration

// Config keys shared by every fuel card feed. The SFTP half matches what
// shared/sftp needs to dial, so a connector can build its transport straight from
// the stored configuration.
const (
	ConfigKeyFuelCardProvider   = "cardProvider"
	ConfigKeyFuelAccountNumber  = "accountNumber"
	ConfigKeyFuelFileFormat     = "fileFormat"
	ConfigKeyFuelHost           = "host"
	ConfigKeyFuelPort           = "port"
	ConfigKeyFuelUsername       = "username"
	ConfigKeyFuelAuthMode       = "authMode"
	ConfigKeyFuelPassword       = "password"
	ConfigKeyFuelPrivateKey     = "privateKey"
	ConfigKeyFuelKnownHostKey   = "knownHostKey"
	ConfigKeyFuelRemoteDir      = "remoteDirectory"
	ConfigKeyFuelArchiveDir     = "archiveDirectory"
	ConfigKeyFuelFilePattern    = "filePattern"
	ConfigKeyFuelClientID       = "clientId"
	ConfigKeyFuelClientSecret   = "clientSecret"
	ConfigKeyFuelBaseURL        = "baseUrl"
	ConfigKeyFuelDefaultCardEnd = "createUnknownCards"

	ConfigKeyFuelDefaultFuelType  = "defaultFuelType"
	ConfigKeyFuelCurrency         = "currency"
	ConfigKeyFuelFixedWidthLayout = "fixedWidthLayout"
)

const (
	FuelFileFormatDelimited  = "Delimited"
	FuelFileFormatFixedWidth = "FixedWidth"

	FuelAuthModePassword   = "password"
	FuelAuthModePrivateKey = "privateKey"
)

// sftpFeedFields is the transport half of a file-based fuel feed. Both fleet
// networks let a customer schedule an export to their own SFTP server, which is
// the only path that works without a partner agreement with the network.
func sftpFeedFields(defaultFormat, patternHelp string) []ConfigFieldSpec {
	return []ConfigFieldSpec{
		{
			Key:         ConfigKeyFuelHost,
			Label:       "SFTP Host",
			Type:        ConfigFieldTypeString,
			Required:    true,
			Placeholder: "sftp.example.com",
			HelpText:    "The server Trenova reads transaction exports from.",
		},
		{
			Key:         ConfigKeyFuelPort,
			Label:       "Port",
			Type:        ConfigFieldTypeNumber,
			Default:     "22",
			Placeholder: "22",
		},
		{
			Key:      ConfigKeyFuelUsername,
			Label:    "Username",
			Type:     ConfigFieldTypeString,
			Required: true,
		},
		{
			Key:      ConfigKeyFuelAuthMode,
			Label:    "Authentication",
			Type:     ConfigFieldTypeSelect,
			Required: true,
			Default:  FuelAuthModePassword,
			Options:  []string{FuelAuthModePassword, FuelAuthModePrivateKey},
		},
		{
			Key:       ConfigKeyFuelPassword,
			Label:     "Password",
			Type:      ConfigFieldTypePassword,
			Sensitive: true,
			HelpText:  "Required when authentication is set to password.",
		},
		{
			Key:       ConfigKeyFuelPrivateKey,
			Label:     "Private Key",
			Type:      ConfigFieldTypePassword,
			Sensitive: true,
			HelpText:  "PEM-encoded OpenSSH key. Required when authentication is set to private key.",
		},
		{
			Key:      ConfigKeyFuelKnownHostKey,
			Label:    "Known Host Key",
			Type:     ConfigFieldTypeString,
			Required: true,
			HelpText: "The server's public host key, in authorized_keys or known_hosts form. Trenova refuses to connect to a server whose key does not match.",
		},
		{
			Key:         ConfigKeyFuelRemoteDir,
			Label:       "Remote Directory",
			Type:        ConfigFieldTypeString,
			Required:    true,
			Placeholder: "/outbound",
			HelpText:    "Where the provider drops transaction exports.",
		},
		{
			Key:         ConfigKeyFuelArchiveDir,
			Label:       "Archive Directory",
			Type:        ConfigFieldTypeString,
			Placeholder: "/outbound/processed",
			HelpText:    "Where a file is moved once it has been read. Defaults to a processed folder beside the remote directory.",
		},
		{
			Key:         ConfigKeyFuelFilePattern,
			Label:       "File Name Pattern",
			Type:        ConfigFieldTypeString,
			Placeholder: "*",
			HelpText:    patternHelp,
		},
		{
			Key:      ConfigKeyFuelFileFormat,
			Label:    "File Format",
			Type:     ConfigFieldTypeSelect,
			Required: true,
			Default:  defaultFormat,
			Options:  []string{FuelFileFormatDelimited, FuelFileFormatFixedWidth},
		},
		{
			Key:         ConfigKeyFuelFixedWidthLayout,
			Label:       "Fixed-Width Record Layout",
			Type:        ConfigFieldTypeString,
			Placeholder: "Transaction Date:1-8, Card Number:9-24, Unit Number:25-34",
			HelpText:    "Required when the file format is fixed width. One entry per field, written as Name:start-end using the column numbers from the provider's record layout document. The names are matched to Trenova fields the same way spreadsheet headers are, and lines beginning with # are notes.",
		},
	}
}

// feedDefaultsFields fill in what a file may not carry. A fixed-width export
// often omits the product description entirely, and a feed has no person to ask,
// so the connection says what to assume.
func feedDefaultsFields() []ConfigFieldSpec {
	return []ConfigFieldSpec{
		{
			Key:      ConfigKeyFuelDefaultFuelType,
			Label:    "Default Fuel Type",
			Type:     ConfigFieldTypeSelect,
			Options:  fuelTypeOptions(),
			HelpText: "Used when a file has no product column at all. A file that names its product per row ignores this.",
		},
		{
			Key:         ConfigKeyFuelCurrency,
			Label:       "Currency",
			Type:        ConfigFieldTypeString,
			Default:     "USD",
			Placeholder: "USD",
			HelpText:    "Used when a transaction does not name its currency.",
		},
	}
}

func fuelTypeOptions() []string {
	return []string{
		"Diesel", "Gasoline", "Biodiesel", "Propane", "LNG", "CNG",
		"Ethanol", "Methanol", "E85", "M85", "A55", "Electricity", "Hydrogen",
	}
}

func fuelAccountField(helpText string) ConfigFieldSpec {
	return ConfigFieldSpec{
		Key:      ConfigKeyFuelAccountNumber,
		Label:    "Account Number",
		Type:     ConfigFieldTypeString,
		Required: true,
		HelpText: helpText,
	}
}

func discoverCardsField() ConfigFieldSpec {
	return ConfigFieldSpec{
		Key:      ConfigKeyFuelDefaultCardEnd,
		Label:    "Create Cards Automatically",
		Type:     ConfigFieldTypeBoolean,
		Default:  "true",
		HelpText: "When a transaction arrives on a card nobody has registered, add it as an unassigned card so it can be assigned to a tractor or driver. Leave this off to have those transactions wait in the review queue instead.",
	}
}

func wexFuelSpec() IntegrationSpec {
	fields := []ConfigFieldSpec{
		{
			Key:      ConfigKeyFuelCardProvider,
			Label:    "Card Brand",
			Type:     ConfigFieldTypeSelect,
			Required: true,
			Default:  "WEX",
			Options:  []string{"WEX", "EFS"},
			HelpText: "WEX acquired EFS in 2016 and both run on the same network. Pick the brand printed on the cards so purchases match the right ones.",
		},
		fuelAccountField("The WEX or EFS account the exports cover."),
	}
	fields = append(fields, sftpFeedFields(
		FuelFileFormatDelimited,
		"Glob matched against file names, for example WEX_*.csv. Leave blank to read every file in the directory.",
	)...)

	fields = append(fields, feedDefaultsFields()...)

	return IntegrationSpec{
		Fields:              append(fields, discoverCardsField()),
		SupportsTestConnect: true,
	}
}

func comdataFuelSpec() IntegrationSpec {
	fields := []ConfigFieldSpec{
		fuelAccountField("The Comdata account code the exports cover."),
	}
	fields = append(fields, sftpFeedFields(
		FuelFileFormatFixedWidth,
		"Glob matched against file names, for example AC00029*. Leave blank to read every file in the directory.",
	)...)

	fields = append(fields, feedDefaultsFields()...)

	return IntegrationSpec{
		Fields:              append(fields, discoverCardsField()),
		SupportsTestConnect: true,
	}
}

func rampFuelSpec() IntegrationSpec {
	return IntegrationSpec{
		Fields: []ConfigFieldSpec{
			{
				Key:      ConfigKeyFuelClientID,
				Label:    "Client ID",
				Type:     ConfigFieldTypeString,
				Required: true,
				HelpText: "From the API app a Ramp admin creates under Developer settings.",
			},
			{
				Key:       ConfigKeyFuelClientSecret,
				Label:     "Client Secret",
				Type:      ConfigFieldTypePassword,
				Required:  true,
				Sensitive: true,
			},
			{
				Key:         ConfigKeyFuelBaseURL,
				Label:       "Base URL",
				Type:        ConfigFieldTypeURL,
				Default:     "https://api.ramp.com/developer/v1",
				Placeholder: "https://api.ramp.com/developer/v1",
			},
			{
				Key:      ConfigKeyFuelDefaultFuelType,
				Label:    "Default Fuel Type",
				Type:     ConfigFieldTypeSelect,
				Required: true,
				Default:  "Diesel",
				Options:  fuelTypeOptions(),
				HelpText: "Ramp transactions never name a product, so every row needs a fuel type to fall back on. Rows still wait for review, because they carry no gallons either.",
			},
			{
				Key:         ConfigKeyFuelCurrency,
				Label:       "Currency",
				Type:        ConfigFieldTypeString,
				Default:     "USD",
				Placeholder: "USD",
				HelpText:    "Used when a transaction does not name its currency.",
			},
			discoverCardsField(),
		},
		SupportsTestConnect: true,
	}
}
