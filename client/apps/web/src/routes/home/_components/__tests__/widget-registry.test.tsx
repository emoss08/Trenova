import { beforeAll, describe, expect, it } from "vitest";
import { WIDGET_COMPONENTS, WIDGET_LOADERS, widgetComponentFor } from "../widget-registry";

/**
 * Mirrors the widget keys in
 * services/tms/internal/modules/workspace/domain/homelayout/catalog.go. The server decides
 * which widgets exist; this list is the client's promise that it can draw each
 * of them. Adding a widget to the Go catalog without adding it here means the
 * server offers a card the client renders as nothing, so the two lists are kept
 * in step by this test failing.
 */
const CATALOG_KEYS = [
  // Work
  "attention",
  "unassigned",
  "exceptions",
  "detention-watch",
  "tomorrows-pickups",
  "my-approvals",
  "billing-queue",
  "service-failures",
  "edi-attention",
  "worker-attention",
  "expiring-credentials",
  // Pulse
  "kpi",
  "kpi-row",
  "ar-snapshot",
  "revenue-trend",
  "fleet-status",
  "on-time-goal",
  // Insight
  "ai-insights",
  "report",
  "dashboard-link",
  "lane-heatmap",
  "customer-mix",
  // Orientation
  "quick-actions",
  "favorites",
  "jump-back-in",
  "saved-views",
  "activity",
  "notifications",
  // Comms
  "announcement",
  "map",
];

describe("widget registry", () => {
  // The first read of a widget module transforms everything it draws with, which is
  // slower than one test's allowance; reading them all once here keeps each case fast.
  beforeAll(async () => {
    await Promise.all(Object.values(WIDGET_LOADERS).map((load) => load()));
  }, 120_000);

  // The registry holds loaders, so the proof that a key draws is that its loader
  // lands on a real component: a renamed export would otherwise pass here and leave
  // the tile on its skeleton for ever.
  it.each(CATALOG_KEYS)("loads the component that draws %s", async (key) => {
    expect(await WIDGET_LOADERS[key]()).toBeTypeOf("function");
    expect(widgetComponentFor(key)).not.toBeNull();
  });

  it("registers nothing the catalog does not offer", () => {
    expect(Object.keys(WIDGET_COMPONENTS).sort()).toEqual([...CATALOG_KEYS].sort());
  });

  // A key the deployed server knows and this build does not must degrade to an
  // absent card, never to a crash — that is what makes a widget safe to ship
  // ahead of a client release.
  it("returns null for an unknown key", () => {
    expect(widgetComponentFor("widget-from-the-future")).toBeNull();
  });
});
