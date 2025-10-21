import { parseAsInteger, parseAsString, parseAsStringLiteral } from "nuqs";

export const searchParamsParser = {
  entityId: parseAsString.withOptions({
    shallow: true,
  }),
  modalType: parseAsStringLiteral(["edit", "create"]).withOptions({
    shallow: true,
  }),
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
