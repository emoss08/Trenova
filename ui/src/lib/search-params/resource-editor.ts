/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { createSerializer, parseAsString, parseAsStringLiteral } from "nuqs";

export const resourceEditorSearchParamsParser = {
  selectedTable: parseAsString.withOptions({
    shallow: true,
  }),
  aceTheme: parseAsStringLiteral(["dawn", "tomorrow_night_bright"])
    .withOptions({
      shallow: true,
    })
    .withDefault("dawn"),
};

export const resourceEditorSearchParamsSerializer = createSerializer(
  resourceEditorSearchParamsParser,
);
