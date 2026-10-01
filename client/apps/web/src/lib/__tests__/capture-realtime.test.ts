import { capture } from "@/lib/queries/capture";
import {
  RESOURCE_QUERY_KEY_MAP,
  invalidationFor,
  queryKeyPrefix,
} from "@trenova/shared/hooks/realtime-patching";
import { describe, expect, it } from "vitest";

/*
 * The server announces capture writes under three resources. An open intake
 * queue, an open batch, or a Documents tab watching a scan in flight moves only
 * if those names reach the keys the screens are cached under.
 */
function reachedBy(resource: string) {
  const roots = (RESOURCE_QUERY_KEY_MAP[resource] ?? []).map(queryKeyPrefix);

  return (key: readonly unknown[]) =>
    roots.some((root) => root.every((part, index) => key[index] === part));
}

describe("capture invalidation", () => {
  const batch = reachedBy("capture_batch");

  it.each([
    ["the intake queue", capture.batches({ statuses: ["Ready"] }).queryKey],
    ["the queue's counts", capture.batchCount({ statuses: ["Ready"] }).queryKey],
    ["an open batch", capture.batch("cbat_1").queryKey],
    ["a record's requests", capture.requests("shipment", "shp_1").queryKey],
  ])("capture_batch reaches %s", (_, key) => {
    expect(batch(key)).toBe(true);
  });

  it("capture_batch leaves devices and profiles alone", () => {
    expect(batch(capture.myDevices(null).queryKey)).toBe(false);
    expect(batch(capture.profiles(null, "").queryKey)).toBe(false);
  });

  it("a page arriving refetches the queue and that batch on screen, nothing else", () => {
    const event = {
      organizationId: "org_1",
      businessUnitId: "bu_1",
      resource: "capture_batch",
      action: "page.received",
      recordId: "cbat_1",
    };
    const { roots, activeOnly } = invalidationFor(event);
    expect(activeOnly).toBe(true);
    const reaches = (key: readonly unknown[]) =>
      roots.map(queryKeyPrefix).some((root) => root.every((part, i) => key[i] === part));
    expect(reaches(capture.batches({ statuses: ["Ready"] }).queryKey)).toBe(true);
    expect(reaches(capture.batchCount({ statuses: ["Ready"] }).queryKey)).toBe(true);
    expect(reaches(capture.batch("cbat_1").queryKey)).toBe(true);
    expect(reaches(capture.batch("cbat_2").queryKey)).toBe(false);
    expect(reaches(capture.requests("shipment", "shp_1").queryKey)).toBe(false);
  });

  it("any other batch change reaches everything capture_batch names", () => {
    const { roots, activeOnly } = invalidationFor({
      organizationId: "org_1",
      businessUnitId: "bu_1",
      resource: "capture_batch",
      action: "updated",
      recordId: "cbat_1",
    });
    expect(activeOnly).toBe(false);
    expect(roots).toBe(RESOURCE_QUERY_KEY_MAP.capture_batch);
  });

  it("capture_device reaches both device lists", () => {
    const device = reachedBy("capture_device");
    expect(device(capture.myDevices("Active").queryKey)).toBe(true);
    expect(device(capture.devices(null, "").queryKey)).toBe(true);
  });

  it("capture_profile reaches administration and the scan picker", () => {
    const profile = reachedBy("capture_profile");
    expect(profile(capture.profiles("Active", "").queryKey)).toBe(true);
    expect(profile(capture.availableProfiles().queryKey)).toBe(true);
  });
});
