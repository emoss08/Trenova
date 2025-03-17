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
