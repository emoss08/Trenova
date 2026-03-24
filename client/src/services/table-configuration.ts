import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  tableConfigurationResponseSchema,
  tableConfigurationSchema,
  type ConfigurationVisibility,
  type TableConfiguration,
  type TableConfigurationFormValues,
  type TableConfigurationResponse,
} from "@/types/table-configuration";

export type ListTableConfigurationsParams = {
  resource: string;
  visibility?: ConfigurationVisibility;
  limit?: number;
  offset?: number;
  query?: string;
};

export class TableConfigurationService {
  public async list(params: ListTableConfigurationsParams) {
    const searchParams = new URLSearchParams();

    searchParams.set("resource", params.resource);
    if (params.visibility) searchParams.set("visibility", params.visibility);
    if (params.limit) searchParams.set("limit", String(params.limit));
    if (params.offset) searchParams.set("offset", String(params.offset));
    if (params.query) searchParams.set("query", params.query);

    const queryString = searchParams.toString();

    const response = await api.get<TableConfigurationResponse>(
      `/table-configurations/?${queryString}`,
    );

    return safeParse(tableConfigurationResponseSchema, response, "Table Configuration List");
  }

  public async get(id: TableConfiguration["id"]) {
    const response = await api.get<TableConfiguration>(
      `/table-configurations/${id}`,
    );

    return safeParse(tableConfigurationSchema, response, "Table Configuration");
  }

  public async getDefault(resource: TableConfiguration["resource"]) {
    const response = await api.get<TableConfiguration | null>(
      `/table-configurations/default?resource=${resource}`,
    );

    if (!response) return null;

    return safeParse(tableConfigurationSchema, response, "Table Configuration");
  }

  public async create(data: TableConfigurationFormValues) {
    const response = await api.post<TableConfiguration>(
      `/table-configurations/`,
      data,
    );

    return safeParse(tableConfigurationSchema, response, "Table Configuration");
  }

  public async update(
    id: TableConfiguration["id"],
    data: Partial<TableConfigurationFormValues>,
  ) {
    const response = await api.put<TableConfiguration>(
      `/table-configurations/${id}`,
      data,
    );

    return safeParse(tableConfigurationSchema, response, "Table Configuration");
  }

  public async delete(id: TableConfiguration["id"]) {
    await api.delete(`/table-configurations/${id}`);
  }

  public async setDefault(id: TableConfiguration["id"]) {
    const response = await api.post<TableConfiguration>(
      `/table-configurations/${id}/set-default`,
    );

    return safeParse(tableConfigurationSchema, response, "Table Configuration");
  }
}
