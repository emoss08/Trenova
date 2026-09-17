import {
  AGENT_ACCENT_ORDER,
  AGENT_ACCENTS,
  AGENT_ICONS,
  resolveAgentIdentity,
} from "@/components/agent-identity/agent-identity";
import { describe, expect, it } from "vitest";

describe("resolveAgentIdentity", () => {
  it("prefers what the organization picked", () => {
    const identity = resolveAgentIdentity({
      id: "agdef_1",
      icon: "gauge",
      accent: "rose",
      template: "BillingAssistant",
    });

    expect(identity).toEqual({ icon: "gauge", accent: "rose" });
  });

  it("falls back to the starter for the icon and the id for the accent", () => {
    const identity = resolveAgentIdentity({ id: "agdef_1", template: "ComplianceAssistant" });

    expect(identity.icon).toBe("shield");
    expect(AGENT_ACCENT_ORDER).toContain(identity.accent);
  });

  it("gives a custom agent the generic icon", () => {
    expect(resolveAgentIdentity({ id: "agdef_1" }).icon).toBe("bot");
  });

  it("keeps a face for life and ignores a rename", () => {
    const first = resolveAgentIdentity({ id: "agdef_01JABC", name: "Dispatch desk" });
    const renamed = resolveAgentIdentity({ id: "agdef_01JABC", name: "Night dispatch" });

    expect(renamed).toEqual(first);
  });

  it("previews an agent that has not been saved yet", () => {
    const identity = resolveAgentIdentity({ name: "Billing exceptions" });

    expect(AGENT_ACCENT_ORDER).toContain(identity.accent);
  });

  it("reaches every accent across many agents", () => {
    const seen = new Set<string>();
    for (let index = 0; index < 500; index++) {
      seen.add(resolveAgentIdentity({ id: `agdef_${index}` }).accent);
    }

    expect(seen.size).toBe(AGENT_ACCENT_ORDER.length);
  });

  it("ignores a value the server does not know", () => {
    const identity = resolveAgentIdentity({ id: "agdef_1", icon: "skull", accent: "chartreuse" });

    expect(identity.icon).toBe("bot");
    expect(AGENT_ACCENT_ORDER).toContain(identity.accent);
  });

  it("has a glyph and a token for every registered name", () => {
    for (const accent of AGENT_ACCENT_ORDER) {
      expect(AGENT_ACCENTS[accent]).toMatch(/^var\(--agent-/);
    }
    for (const [name, icon] of Object.entries(AGENT_ICONS)) {
      expect(icon, `${name} has no glyph`).toBeDefined();
    }
  });
});
