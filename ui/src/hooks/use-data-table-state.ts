/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import {
  createSerializer,
  parseAsInteger,
  parseAsString,
  parseAsStringLiteral,
  type inferParserType,
} from "nuqs";

export const searchParamsParser = {
  // * Required for selection of entity
  entityId: parseAsString.withOptions({
    shallow: true,
  }),
  modalType: parseAsStringLiteral(["edit", "create"]).withOptions({
    shallow: true,
  }),
  // * Required for pagination
  page: parseAsInteger
    .withOptions({
      shallow: false,
    })
    .withDefault(1),
  pageSize: parseAsInteger
    .withOptions({
      shallow: false,
    })
    .withDefault(10),
  // * Enhanced filtering and sorting
  query: parseAsString.withOptions({
    shallow: false,
  }),
  filters: parseAsString.withOptions({
    shallow: false,
  }),
  sort: parseAsString.withOptions({
    shallow: false,
  }),
};

export const searchParamsSerializer = createSerializer(searchParamsParser);

export type SearchParamsType = inferParserType<typeof searchParamsParser>;
