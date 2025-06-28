import { http } from "@/lib/http-client";
import type { TableConfigurationSchema } from "@/lib/schemas/table-configuration-schema";
import type { Resource } from "@/types/audit-entry";
import type { LimitOffsetResponse } from "@/types/server";
import type { TableConfiguration } from "@/types/table-configuration";

export class TableConfigurationAPI {
  /**
   * Fetch the table configuration for the given identifier. If none exists, the
   * API should create a default record and return it.
   */
  async getDefaultOrLatestConfiguration(resource: Resource) {
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
   * Create a new table configuration.
   */
  async create(payload: TableConfigurationSchema) {
    const { data } = await http.post<TableConfigurationSchema>(
      "/table-configurations/",
      payload,
    );
    return data;
  }

  async update(id: string, payload: TableConfigurationSchema) {
    const { data } = await http.put<TableConfigurationSchema>(
      `/table-configurations/${id}/`,
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
