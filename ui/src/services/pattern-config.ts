import { http } from "@/lib/http-client";
import type { PatternConfigSchema } from "@/lib/schemas/pattern-config-schema";

export class PatternConfigAPI {
  async get() {
    const response = await http.get<PatternConfigSchema>("/pattern-config/");
    return response.data;
  }

  async update(data: PatternConfigSchema) {
    const response = await http.put<PatternConfigSchema>(
      `/pattern-config/`,
      data,
    );
    return response.data;
  }
}
