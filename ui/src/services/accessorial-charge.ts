/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
