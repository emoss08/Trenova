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
