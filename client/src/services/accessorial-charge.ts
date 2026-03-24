import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  accessorialChargeSchema,
  type AccessorialCharge,
} from "@/types/accessorial-charge";

export class AccessorialChargeService {
  public async patch(
    id: AccessorialCharge["id"],
    data: Partial<AccessorialCharge>,
  ) {
    const response = await api.patch<AccessorialCharge>(
      `/accessorial-charges/${id}/`,
      data,
    );

    return safeParse(accessorialChargeSchema, response, "Accessorial Charge");
  }
}
