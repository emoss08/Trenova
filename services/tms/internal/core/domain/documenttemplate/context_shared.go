package documenttemplate

// Several kinds publish the same variable. Sharing the definition rather than
// repeating it keeps the description identical everywhere, which matters because the
// editor shows it verbatim: an administrator authoring an invoice and a detention
// notice should not be told two different things about the same field.

func logoVariable() VariableDefinition {
	return VariableDefinition{
		Path:        "LogoDataURI",
		Type:        VariableString,
		Description: "The organization logo, already embedded. Use it as an image source directly; a template never names a URL the renderer would fetch.",
	}
}

func companyNameVariable() VariableDefinition {
	return VariableDefinition{
		Path:        "CompanyName",
		Type:        VariableString,
		Description: "Your organization's name, as the recipient knows it.",
	}
}

func customerNameVariable(required bool, description string) VariableDefinition {
	return VariableDefinition{
		Path:        "CustomerName",
		Type:        VariableString,
		Required:    required,
		Description: description,
	}
}

func proNumberVariable(required bool, description string) VariableDefinition {
	return VariableDefinition{
		Path:        "ShipmentProNumber",
		Type:        VariableString,
		Required:    required,
		Description: description,
	}
}

func titleVariable(description string) VariableDefinition {
	return VariableDefinition{
		Path:        "Title",
		Type:        VariableString,
		Required:    true,
		Description: description,
	}
}

func descriptionVariable() VariableDefinition {
	return VariableDefinition{
		Path:        "Description",
		Type:        VariableString,
		Description: "What the report shows.",
	}
}

func requestedByVariable() VariableDefinition {
	return VariableDefinition{
		Path:        "RequestedBy",
		Type:        VariableString,
		Description: "Who ran the report or owns the schedule.",
	}
}

func rowCountVariable() VariableDefinition {
	return VariableDefinition{
		Path:        "RowCount",
		Type:        VariableInt,
		Description: "How many rows the run returned.",
	}
}

func truncatedVariable() VariableDefinition {
	return VariableDefinition{
		Path:        "Truncated",
		Type:        VariableBool,
		Description: "True when the row limit cut the results short. Say so when it is, or a reader will treat a partial answer as the whole one.",
	}
}

func invoiceNumberVariable(required bool, description string) VariableDefinition {
	return VariableDefinition{
		Path:        "InvoiceNumber",
		Type:        VariableString,
		Required:    required,
		Description: description,
	}
}

func invoiceDateVariable() VariableDefinition {
	return VariableDefinition{
		Path:        "InvoiceDate",
		Type:        VariableDate,
		Description: "The date the invoice was issued, in the organization timezone.",
	}
}

func dueDateVariable() VariableDefinition {
	return VariableDefinition{
		Path:        "DueDate",
		Type:        VariableDate,
		Description: "When payment is due. Blank when the billing settings hide it.",
	}
}

func paymentTermVariable() VariableDefinition {
	return VariableDefinition{
		Path:        "PaymentTerm",
		Type:        VariableString,
		Description: "The payment term code, such as Net30.",
	}
}

func currencyCodeVariable() VariableDefinition {
	return VariableDefinition{
		Path:        "CurrencyCode",
		Type:        VariableString,
		Description: "The invoice currency.",
	}
}

func addressBlockFields() []VariableDefinition {
	return []VariableDefinition{
		{Path: "Name", Type: VariableString, Description: "Party name."},
		{
			Path:        "Lines",
			Type:        VariableStringList,
			Description: "Address lines. Join them with the join function to skip the blanks and avoid a dangling separator.",
		},
		{
			Path:        "Details",
			Type:        VariableCollection,
			Description: "Extra labelled detail for this party.",
			Fields:      keyValueFields(),
		},
	}
}

func keyValueFields() []VariableDefinition {
	return []VariableDefinition{
		{Path: "Label", Type: VariableString, Description: "The label."},
		{Path: "Value", Type: VariableString, Description: "The value."},
	}
}

func reportCellFields() []VariableDefinition {
	return []VariableDefinition{
		{Path: "Text", Type: VariableString, Description: "The formatted cell value."},
		{
			Path: "Tone",
			Type: VariableString,
			Description: "Emphasis the report asked for: positive, warning, negative, " +
				"neutral, or blank. Use it to colour exceptions.",
		},
	}
}
