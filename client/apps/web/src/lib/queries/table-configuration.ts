import {
  tableConfigurationService,
  type ListTableConfigurationsParams,
} from "@/services/table-configuration";
import type { TableConfiguration } from "@/types/table-configuration";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const tableConfiguration = createQueryKeys("tableConfiguration", {
  all: (params: ListTableConfigurationsParams) => ({
    queryKey: [params],
    queryFn: async ({ signal }) => tableConfigurationService.list(params, { signal }),
  }),
  detail: (id: TableConfiguration["id"]) => ({
    queryKey: [id],
    queryFn: async ({ signal }) => tableConfigurationService.get(id, { signal }),
  }),
  default: (resource: TableConfiguration["resource"]) => ({
    queryKey: [resource],
    queryFn: async ({ signal }) => tableConfigurationService.getDefault(resource, { signal }),
  }),
});
