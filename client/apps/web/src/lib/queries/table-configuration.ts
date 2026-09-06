import { apiService } from "@/services/api";
import type { ListTableConfigurationsParams } from "@/services/table-configuration";
import type { TableConfiguration } from "@/types/table-configuration";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const tableConfiguration = createQueryKeys("tableConfiguration", {
  all: (params: ListTableConfigurationsParams) => ({
    queryKey: [params],
    queryFn: async ({ signal }) => apiService.tableConfigurationService.list(params, { signal }),
  }),
  detail: (id: TableConfiguration["id"]) => ({
    queryKey: [id],
    queryFn: async ({ signal }) => apiService.tableConfigurationService.get(id, { signal }),
  }),
  default: (resource: TableConfiguration["resource"]) => ({
    queryKey: [resource],
    queryFn: async ({ signal }) =>
      apiService.tableConfigurationService.getDefault(resource, { signal }),
  }),
});
