/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { http } from "@/lib/http-client";
import type { ShipmentControlSchema } from "@/lib/schemas/shipmentcontrol-schema";

export class ShipmentControlAPI {
  async get() {
    const response = await http.get<ShipmentControlSchema>(
      "/shipment-controls/",
    );
    return response.data;
  }

  async update(data: ShipmentControlSchema) {
    const response = await http.put<ShipmentControlSchema>(
      `/shipment-controls/`,
      data,
    );
    return response.data;
  }
}
