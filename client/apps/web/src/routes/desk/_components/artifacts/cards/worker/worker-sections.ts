import { Resource } from "@trenova/shared/types/permission";

/**
 * The worker card's sections built from records other than the worker, by the
 * resource each reads. Someone who may read the worker but not one of these
 * sees the card without that section. Its own module, so the skeleton reads it
 * without the card's code.
 */
export const WORKER_SECTIONS = {
  shipments: Resource.Shipment,
  credentials: Resource.WorkerCredential,
} as const;
