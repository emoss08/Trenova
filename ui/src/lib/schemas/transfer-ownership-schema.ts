/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import * as z from "zod/v4";
import { nullableStringSchema } from "./helpers";

export const transferOwnershipSchema = z.object({
  ownerId: nullableStringSchema,
  shipmentId: z.string().min(1, { error: "Shipment ID is required" }),
});

export type TransferOwnershipSchema = z.infer<typeof transferOwnershipSchema>;
