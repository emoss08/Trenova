import {
  ediMappingPreviewSchema,
  ediTransferSchema,
  type EDIMappingPreview,
  type EDITransfer,
} from "@trenova/shared/types/edi";
import approved from "./approved.json";
import failed from "./failed.json";
import mappingRequired from "./mapping-required.json";
import mappingRequiredPreview from "./mapping-required.preview.json";
import outbound from "./outbound.json";
import rejected from "./rejected.json";
import superseded from "./superseded.json";

/**
 * The wire fixtures were written by the Go service itself: each transfer is
 * what `GET /edi/transfers/:id/` returns for an `EDITransfer` (a nil id as
 * `null`, an unset string as `""`, a decimal as its shortest string, both
 * partners joined), and the preview what `GET /edi/transfers/:id/mapping-preview/`
 * returns.
 *
 * - mappingRequired: Acme's change tender LT-88241 from an inbound 204, its
 *   San Antonio delivery and fuel surcharge not yet mapped.
 * - approved: Acme's LT-88240, accepted, its shipment built.
 * - rejected: LT-88199, declined for lack of capacity.
 * - superseded: LT-88237, canceled when Acme's change replaced it.
 * - failed: LT-88233, accepted but the shipment could not be built.
 * - outbound: our S2610-1366 tendered to Lone Star, a Trenova organization,
 *   waiting on their approval.
 */
const WIRE_TRANSFERS = {
  mappingRequired,
  approved,
  rejected,
  superseded,
  failed,
  outbound,
} as const;

export type WireTransferName = keyof typeof WIRE_TRANSFERS;

export const NOW = 1_791_720_000;

/** A fresh copy of a wire transfer, with any top-level field replaced. */
export function wireTransfer(
  name: WireTransferName,
  overrides: Record<string, unknown> = {},
): Record<string, unknown> {
  return { ...structuredClone(WIRE_TRANSFERS[name]), ...overrides };
}

export function wireMappingPreview(): Record<string, unknown> {
  return structuredClone(mappingRequiredPreview);
}

export function transfer(
  name: WireTransferName,
  overrides: Record<string, unknown> = {},
): EDITransfer {
  return ediTransferSchema.parse(wireTransfer(name, overrides));
}

export function mappingPreview(): EDIMappingPreview {
  return ediMappingPreviewSchema.parse(wireMappingPreview());
}
