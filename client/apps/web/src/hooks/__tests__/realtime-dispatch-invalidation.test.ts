import { describe, expect, it } from "vitest";
import { RESOURCE_QUERY_KEY_MAP } from "@trenova/shared/hooks/realtime-patching";
import {
  DISPATCH_BOARD_KEY,
  DISPATCH_LIVE_TENDER_KEY,
  DISPATCH_SHIPMENT_TENDERS_KEY,
} from "@/lib/queries/dispatch-console";

/**
 * tenderservice publishes its lifecycle transitions on the "shipments" resource
 * with custom actions, and a custom action always takes the invalidation path.
 * The dispatch console's tender views only see those transitions if their query
 * keys are on the shipments fan-out; without them the console falls back to its
 * 15s/30s poll and a carrier's acceptance sits invisible for up to half a minute.
 */
describe("shipment realtime invalidation reaches the dispatch console tender views", () => {
  it("fans out to every dispatch-console query key that renders tender state", () => {
    const keys = RESOURCE_QUERY_KEY_MAP.shipments;

    expect(keys).toContain(DISPATCH_BOARD_KEY);
    expect(keys).toContain(DISPATCH_LIVE_TENDER_KEY);
    expect(keys).toContain(DISPATCH_SHIPMENT_TENDERS_KEY);
  });
});
