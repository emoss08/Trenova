import { parseAsInteger } from "nuqs";

export const DataTableSearchParams = {
  page: parseAsInteger.withDefault(1),
  pageSize: parseAsInteger.withDefault(10),
};
