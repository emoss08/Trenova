import { apiService } from "@/services/api";
import type { ListTableConfigurationsParams } from "@/services/table-configuration";
import type { TableConfiguration } from "@/types/table-configuration";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const tableConfiguration = createQueryKeys("tableConfiguration", {
  all: (params: ListTableConfigurationsParams) => ({
    queryKey: [params],
    queryFn: async () => apiService.tableConfigurationService.list(params),
  }),
  detail: (id: TableConfiguration["id"]) => ({
    queryKey: [id],
    queryFn: async () => apiService.tableConfigurationService.get(id),
  }),
  default: (resource: TableConfiguration["resource"]) => ({
    queryKey: [resource],
    queryFn: async () =>
      apiService.tableConfigurationService.getDefault(resource),
  }),
});
