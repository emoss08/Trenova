import { http } from "@/lib/http-client";
import type { AccessorialChargeSchema } from "@/lib/schemas/accessorial-charge-schema";

export class AccessorialChargeAPI {
  async getById(accID: AccessorialChargeSchema["id"]) {
    const response = await http.get<AccessorialChargeSchema>(
      `/accessorial-charges/${accID}`,
    );

    return response.data;
  }
}
