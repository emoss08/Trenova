import { http } from "@/lib/http-client";
import type {
  TableConfig,
  TableConfiguration,
} from "@/types/table-configuration";

export class TableConfigurationAPI {
  /**
   * Fetch the table configuration for the given identifier. If none exists, the
   * API should create a default record and return it.
   */
  async get(tableIdentifier: string) {
    try {
      const { data } = await http.get<TableConfiguration>(
        `/table-configurations/${tableIdentifier}/`,
      );
      return data;
    } catch (error: any) {
      if (error?.status === 404) {
        return undefined as unknown as TableConfiguration;
      }
      throw error;
    }
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
  async createConfiguration(payload: {
    name: string;
    tableIdentifier: string;
    visibility: "Private" | "Public" | "Shared";
    tableConfig: TableConfig;
  }) {
    const { data } = await http.post<TableConfiguration>(
      "/table-configurations/",
      payload,
    );
    return data;
  }
}
