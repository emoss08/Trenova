package agentruntime

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
