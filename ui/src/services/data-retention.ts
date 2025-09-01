import { http } from "@/lib/http-client";
import { DataRetentionSchema } from "@/lib/schemas/data-retention-schema";

export class DataRetentionAPI {
  async get() {
    const response = await http.get<DataRetentionSchema>("/data-retention/");
    return response.data;
  }

  async update(data: DataRetentionSchema) {
    const response = await http.put<DataRetentionSchema>(
      "/data-retention/",
      data,
    );
    return response.data;
  }
}
