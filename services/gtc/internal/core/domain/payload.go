package domain

import "fmt"

func SelectFields(data map[string]any, fields []string) (map[string]any, error) {
	if data == nil {
		return nil, fmt.Errorf("projection payload is empty")
	}

	if len(fields) == 0 {
		document := make(map[string]any, len(data))
		for key, value := range data {
			document[key] = value
		}
		return document, nil
	}

	document := make(map[string]any, len(fields))
	for _, field := range fields {
		document[field] = data[field]
	}

	return document, nil
}
