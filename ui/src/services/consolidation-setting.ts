import { http } from "@/lib/http-client";
import { ConsolidationSettingSchema } from "@/lib/schemas/consolidation-setting-schema";

export class ConsolidationSettingsAPI {
  async get() {
    const response = await http.get<ConsolidationSettingSchema>(
      "/consolidation-settings/",
    );
    return response.data;
  }

  async update(data: ConsolidationSettingSchema) {
    const response = await http.put<ConsolidationSettingSchema>(
      "/consolidation-settings/",
      data,
    );

    return response.data;
  }
}
