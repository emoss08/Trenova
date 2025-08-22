import { http } from "@/lib/http-client";
import { HoldReasonSchema } from "@/lib/schemas/hold-reason-schema";

export class HoldReasonAPI {
  async getById(id: HoldReasonSchema["id"]) {
    const { data } = await http.get<HoldReasonSchema>(`/hold-reasons/${id}/`);
    return data;
  }
}
