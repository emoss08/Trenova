import { http } from "@/lib/http-client";
import type {
  CopyTableConfigurationSchema,
  TableConfigurationSchema,
} from "@/lib/schemas/table-configuration-schema";
import type { Resource } from "@/types/audit-entry";
import type { LimitOffsetResponse } from "@/types/server";

export class TableConfigurationAPI {
  /**
   * Fetch the table configuration for the given identifier. If none exists, the
   * API should create a default record and return it.
   */
  async getDefaultOrLatestConfiguration(resource: Resource) {
    try {
      const { data } = await http.get<TableConfigurationSchema>(
        `/table-configurations/${resource}/`,
      );
      return data;
    } catch (error: any) {
      if (error?.status === 404) {
        return undefined as unknown as TableConfigurationSchema;
      }
      throw error;
    }
  }

  async listUserConfigurations(resource: Resource) {
    const { data } = await http.get<
      LimitOffsetResponse<TableConfigurationSchema>
    >(`/table-configurations/me/${resource}/`);

    return data;
  }

  async listPublicConfigurations(resource: Resource) {
    const { data } = await http.get<
      LimitOffsetResponse<TableConfigurationSchema>
    >(`/table-configurations/public/${resource}/`);

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

  async copy(payload: CopyTableConfigurationSchema) {
    const { data } = await http.post<TableConfigurationSchema>(
      "/table-configurations/copy/",
      payload,
    );
    return data;
  }

  async delete(id: string) {
    await http.delete(`/table-configurations/${id}/`);
  }
}
