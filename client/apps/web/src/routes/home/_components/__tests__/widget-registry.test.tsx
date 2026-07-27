import { describe, expect, it } from "vitest";
import { WIDGET_COMPONENTS, widgetComponentFor } from "../widget-registry";

/**
 * Mirrors the widget keys in
 * services/tms/internal/core/domain/homelayout/catalog.go. The server decides
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
  "expiring-credentials",
  // Pulse
  "kpi",
  "kpi-row",
  "ar-snapshot",
  "revenue-trend",
  "fleet-status",
  "on-time-goal",
  // Insight
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
  it.each(CATALOG_KEYS)("draws %s", (key) => {
    expect(widgetComponentFor(key)).toBeTypeOf("function");
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
