/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { Resource } from "@/types/audit-entry";
import { parseAsString, parseAsStringEnum } from "nuqs";

export const searchParams = {
  selectedFolder: parseAsStringEnum(Object.values(Resource)).withOptions({
    shallow: true,
  }),
  selectedSubFolder: parseAsString.withDefault("").withOptions({
    shallow: true,
  }),
};
