import { describe, expect, it } from "vitest";
import type { AIControlSegment } from "@/lib/graphql/ai-control";
import { novaTarget } from "../use-nova-segments";

const segment = (target: string | null, providerId: string | null = null): AIControlSegment =>
  ({ text: "x", strong: false, target, providerId, tone: null }) as AIControlSegment;

describe("novaTarget", () => {
  it("leads each target the server names to its place", () => {
    expect(novaTarget(segment("watchtower"))).toEqual({ kind: "watchtower" });
    expect(novaTarget(segment("routing"))).toEqual({ kind: "routing" });
    expect(novaTarget(segment("agents:shadow"))).toEqual({ kind: "agents", filter: "shadow" });
    expect(novaTarget(segment("provider", "aiprv_1"))).toEqual({
      kind: "provider",
      providerId: "aiprv_1",
    });
  });

  it("is plain text for no target, an unknown one, or a provider without its ID", () => {
    expect(novaTarget(segment(null))).toBeNull();
    expect(novaTarget(segment("somewhere-new"))).toBeNull();
    expect(novaTarget(segment("provider"))).toBeNull();
  });
});
