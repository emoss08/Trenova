/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package cdcutils

import (
	"github.com/emoss08/trenova/internal/pkg/utils/maputils"
	"github.com/emoss08/trenova/shared/cdctypes"
)

func ExtractCDCSource(source map[string]any) cdctypes.CDCSource {
	var isSnapshot bool
	if snapshotField := source["snapshot"]; snapshotField != nil {
		switch v := snapshotField.(type) {
		case map[string]any:
			if snapshotVal, sOk := v["string"].(string); sOk {
				isSnapshot = snapshotVal != "false"
			}
		case string:
			isSnapshot = v != "false"
		}
	}

	return cdctypes.CDCSource{
		Database:  maputils.ExtractStringField(source, "db"),
		Schema:    maputils.ExtractStringField(source, "schema"),
		Table:     maputils.ExtractStringField(source, "table"),
		Connector: maputils.ExtractStringField(source, "connector"),
		Version:   maputils.ExtractStringField(source, "version"),
		Snapshot:  isSnapshot,
	}
}

func NormalizeOperation(op string) string {
	switch op {
	case "c":
		return "create"
	case "u":
		return "update"
	case "d":
		return "delete"
	case "r":
		return "read"
	default:
		return op
	}
}

func ExtractDataState(avroData map[string]any, key string) map[string]any {
	var data map[string]any
	if dataField := avroData[key]; dataField != nil {
		if dataMap, ok := dataField.(map[string]any); ok {
			data = cdctypes.ExtractValueField(dataMap)
		}
	}
	for k, v := range data {
		data[k] = cdctypes.ConvertAvroOptionalField(v)
	}
	return data
}

func ExtractTransactionID(avroData map[string]any) string {
	if txData, ok := avroData["transaction"].(map[string]any); ok {
		if txID, idOk := txData["id"].(string); idOk {
			return txID
		}
	}
	return ""
}
