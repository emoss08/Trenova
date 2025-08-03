/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { ConsolidationStatus } from "@/lib/schemas/consolidation-schema";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";

export type ConsolidationQueryParams = {
  pageIndex?: number;
  pageSize?: number;
  query?: string;
  status?: ConsolidationStatus;
  expandDetails?: boolean;
  originLocationId?: string;
  destinationLocationId?: string;
  customerId?: string;
  dateFrom?: string;
  dateTo?: string;
};

export type ConsolidationMetrics = {
  totalWeight: number;
  totalVolume: number;
  totalPallets: number;
  totalPieces: number;
  estimatedSavings: number;
  routeEfficiency: number;
  consolidationScore: number;
};

export type AvailableShipment = ShipmentSchema & {
  isSelected?: boolean;
  consolidationScore?: number;
  distanceFromRoute?: number;
};
