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
};

export const searchParamsSerializer = createSerializer(searchParamsParser);

export type SearchParamsType = inferParserType<typeof searchParamsParser>;
