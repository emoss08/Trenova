import { Resource } from "@trenova/shared/types/permission";

/**
 * The worker settlement card's sections built from records other than the
 * settlement, by the resource each reads. Someone who may read the settlement
 * but not one of these sees the card without that section. Its own module, so
 * the skeleton reads it without the card's code.
 */
export const DRIVER_SETTLEMENT_SECTIONS = {
  shipments: Resource.Shipment,
  disputes: Resource.SettlementDispute,
  worker: Resource.Worker,
  activity: Resource.AuditLog,
} as const;
