import { http } from "@/lib/http-client";
import type { Resource } from "@/types/audit-entry";
import type { LimitOffsetResponse } from "@/types/server";
import type {
  TableConfig,
  TableConfiguration,
} from "@/types/table-configuration";

export class TableConfigurationAPI {
  /**
   * Fetch the table configuration for the given identifier. If none exists, the
   * API should create a default record and return it.
   */
  async get(resource: Resource) {
    try {
      const { data } = await http.get<TableConfiguration>(
        `/table-configurations/${resource}/`,
      );
      return data;
    } catch (error: any) {
      if (error?.status === 404) {
        return undefined as unknown as TableConfiguration;
      }
      throw error;
    }
  }

  async listUserConfigurations(resource: Resource) {
    const { data } = await http.get<LimitOffsetResponse<TableConfiguration>>(
      `/table-configurations/me/${resource}`,
    );

    return data;
  }

  /**
   * Partially update the table configuration JSON blob.
   */
  async patch(id: string, tableConfig: Partial<TableConfig>) {
    const { data } = await http.patch<TableConfiguration>(
      `/table-configurations/${id}/`,
      { tableConfig },
    );
    return data;
  }

  /**
   * Create a new table configuration.
   */
  async create(payload: {
    name: string;
    resource: Resource;
    visibility: "Private" | "Public" | "Shared";
    tableConfig: TableConfig;
  }) {
    const { data } = await http.post<TableConfiguration>(
      "/table-configurations/",
      payload,
    );
    return data;
  }

  /**
   * Delete a table configuration.
   * @Note: This returns no content.
   */
  async delete(id: string) {
    await http.delete(`/table-configurations/${id}/`);
  }
}
