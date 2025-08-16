import { http } from "@/lib/http-client";
import { HoldReasonSchema } from "@/lib/schemas/hold-reason-schema";

export class HoldReasonAPI {
  async create(holdReason: HoldReasonSchema) {
    return await http.post<HoldReasonSchema>("/hold-reasons/", holdReason);
  }

  async update(
    holdReasonID: HoldReasonSchema["id"],
    holdReason: HoldReasonSchema,
  ) {
    return await http.put<HoldReasonSchema>(
      `/hold-reasons/${holdReasonID}/`,
      holdReason,
    );
  }
}
