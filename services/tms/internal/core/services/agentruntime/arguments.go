package agentruntime

import "strings"

// declaredArguments keeps the arguments a tool's schema declares and drops
// the rest, when the schema closes itself with additionalProperties: false.
//
// A model invents arguments — a runId it thinks the tool wants, most often.
// The tool never reads them, but the proposal carried them to the card as
// though they were part of the request, and an approver was left deciding
// on a field the write would not use. The record of what the model said is
// kept in the message; the proposal carries what the tool will take.
//
// An open schema, or one that declares no properties, is left alone: without
// a declared set there is nothing to call undeclared.
func declaredArguments(schema, args map[string]any) map[string]any {
	if args == nil {
		return nil
	}
	closed, ok := schema["additionalProperties"].(bool)
	if !ok || closed {
		return args
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok || len(properties) == 0 {
		return args
	}

	kept := make(map[string]any, len(args))
	for key, value := range args {
		if _, declared := properties[key]; declared {
			kept[key] = value
		}
	}

	return kept
}

// aliasedArguments renames an argument the model sent under the wrong name to
// the required parameter it plainly meant, and leaves everything else alone.
//
// A small model asked for a shipment by {"id": ...} when get_shipment takes
// shipmentId, and the call failed on a missing parameter it had in fact
// supplied. Two readings are safe enough to act on: the same name spelled
// another way (shipment_id, ShipmentID), and a bare "id" when exactly one
// required *Id parameter is missing. Anything less certain is left for the
// tool to refuse, which names the parameter it wanted.
//
// The model's own call is not changed: the result is a copy, so the thread
// records what was sent and the tool reads what was meant.
func aliasedArguments(schema, args map[string]any) map[string]any {
	if len(args) == 0 {
		return args
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok || len(properties) == 0 {
		return args
	}

	missing := missingRequired(schema, properties, args)
	if len(missing) == 0 {
		return args
	}

	var aliased map[string]any
	rename := func(from, to string) {
		if aliased == nil {
			aliased = make(map[string]any, len(args))
			for key, value := range args {
				aliased[key] = value
			}
		}
		aliased[to] = aliased[from]
		delete(aliased, from)
	}

	byShape := make(map[string]string, len(missing))
	for _, name := range missing {
		byShape[argumentShape(name)] = name
	}
	for key := range args {
		if _, declared := properties[key]; declared {
			continue
		}
		if target, found := byShape[argumentShape(key)]; found {
			rename(key, target)
			delete(byShape, argumentShape(key))
		}
	}

	if _, declared := properties["id"]; !declared {
		if value, sent := args["id"]; sent && value != nil {
			idTargets := make([]string, 0, 1)
			for _, name := range byShape {
				if len(name) > 2 && strings.HasSuffix(name, "Id") {
					idTargets = append(idTargets, name)
				}
			}
			if len(idTargets) == 1 && (aliased == nil || aliased["id"] != nil) {
				rename("id", idTargets[0])
			}
		}
	}

	if aliased == nil {
		return args
	}

	return aliased
}

// missingRequired is the schema's required parameters the arguments lack.
func missingRequired(schema, properties, args map[string]any) []string {
	var required []string
	switch values := schema["required"].(type) {
	case []string:
		required = values
	case []any:
		required = make([]string, 0, len(values))
		for _, value := range values {
			if name, ok := value.(string); ok {
				required = append(required, name)
			}
		}
	}

	missing := make([]string, 0, len(required))
	for _, name := range required {
		if _, declared := properties[name]; !declared {
			continue
		}
		if value, sent := args[name]; !sent || value == nil {
			missing = append(missing, name)
		}
	}

	return missing
}

// argumentShape is a name with its spelling taken out: case, underscores and
// hyphens, so shipment_id, ShipmentID and shipmentId read alike.
func argumentShape(name string) string {
	return strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(name))
}
