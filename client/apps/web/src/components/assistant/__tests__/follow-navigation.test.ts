import type { AssistantArtifactEvent } from "@/types/assistant";
import { afterEach, describe, expect, it } from "vitest";
import { claimNavigation, nextNavigation } from "../follow-navigation";

function event(overrides: Partial<AssistantArtifactEvent>): AssistantArtifactEvent {
  return {
    id: "aart_1",
    kind: "navigation",
    status: "Ready",
    title: "Invoices",
    sourceToolCallId: "call_1",
    path: "/billing/invoices",
    ...overrides,
  };
}

/** A claim that remembers, like the tab's, without touching storage. */
function claims() {
  const seen = new Set<string>();
  return (id: string) => {
    if (seen.has(id)) return false;
    seen.add(id);
    return true;
  };
}

afterEach(() => {
  sessionStorage.clear();
});

describe("nextNavigation", () => {
  it("follows a page the assistant opened", () => {
    expect(nextNavigation([event({})], claims())).toBe("/billing/invoices");
  });

  it("follows it once, however often the turn is replayed", () => {
    const claim = claims();
    const announced = [event({})];

    expect(nextNavigation(announced, claim)).toBe("/billing/invoices");
    expect(nextNavigation(announced, claim)).toBeNull();
  });

  it("lands on the last place when a turn moved twice, and does not replay the first", () => {
    const claim = claims();
    const announced = [
      event({ id: "aart_1", path: "/billing/invoices" }),
      event({ id: "aart_2", path: "/hr/workers" }),
    ];

    expect(nextNavigation(announced, claim)).toBe("/hr/workers");
    expect(nextNavigation(announced.slice(0, 1), claim)).toBeNull();
  });

  it("ignores every other kind of artifact, even one with a path", () => {
    expect(
      nextNavigation([event({ kind: "entity_card", path: "/hr/workers" })], claims()),
    ).toBeNull();
  });

  it.each(["", "https://example.com", "//example.com/x", "/\\example.com", "javascript:alert(1)"])(
    "never leaves the app for %j",
    (path) => {
      expect(nextNavigation([event({ path })], claims())).toBeNull();
    },
  );
});

describe("claimNavigation", () => {
  it("claims an id once for the tab", () => {
    expect(claimNavigation("aart_claim_once")).toBe(true);
    expect(claimNavigation("aart_claim_once")).toBe(false);
  });

  // A reload keeps the tab's session storage, so a reply rejoined after it
  // does not move the person again.
  it("remembers what was followed across a reload of the page", () => {
    sessionStorage.setItem(
      "trenova-assistant-followed-navigation",
      JSON.stringify(["aart_before_reload"]),
    );

    expect(claimNavigation("aart_before_reload")).toBe(false);
  });

  it("reads unreadable storage as nothing followed", () => {
    sessionStorage.setItem("trenova-assistant-followed-navigation", "{not json");

    expect(claimNavigation("aart_after_garbage")).toBe(true);
  });
});
