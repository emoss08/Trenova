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
