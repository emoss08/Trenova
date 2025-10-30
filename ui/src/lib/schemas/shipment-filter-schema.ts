/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import * as z from "zod";
import { ShipmentStatus } from "./shipment-schema";

export const shipmentFilterSchema = z.object({
  search: z.string().optional(),
  status: ShipmentStatus.optional(),
});

export type ShipmentFilterSchema = z.infer<typeof shipmentFilterSchema>;
