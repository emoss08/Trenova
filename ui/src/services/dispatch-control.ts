import { http } from "@/lib/http-client";
import type { DispatchControlSchema } from "@/lib/schemas/dispatchcontrol-schema";

export class DispatchControlAPI {
  async get() {
    const response = await http.get<DispatchControlSchema>(
      "/dispatch-controls/",
    );
    return response.data;
  }

  async update(data: DispatchControlSchema) {
    const response = await http.put<DispatchControlSchema>(
      `/dispatch-controls/`,
      data,
    );
    return response.data;
  }
}
