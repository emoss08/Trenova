import {
  AGENT_ACCENT_ORDER,
  AGENT_ACCENTS,
  AGENT_ICONS,
  agentMonogram,
  agentSigil,
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

    expect(identity).toMatchObject({ icon: "gauge", accent: "rose", iconChosen: true });
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

describe("template icons", () => {
  it("gives every starter the server knows an icon of its own", async () => {
    const { agentTemplateKindSchema } = await import("@/types/assistant");
    const { TEMPLATE_ICON } = await import("@/components/agent-identity/agent-identity");

    for (const kind of agentTemplateKindSchema.options) {
      expect(TEMPLATE_ICON[kind], kind).toBeDefined();
    }
    expect(agentTemplateKindSchema.options).toContain("CashApplication");
  });
});

describe("agentMonogram", () => {
  it("takes the initials of the first two words", () => {
    expect(agentMonogram("Billing exceptions")).toBe("BE");
    expect(agentMonogram("Night dispatch desk")).toBe("ND");
  });

  it("takes two letters from a single word", () => {
    expect(agentMonogram("Dispatch")).toBe("DI");
  });

  it("has nothing to show for an unnamed agent", () => {
    expect(agentMonogram("")).toBe("");
    expect(agentMonogram("   ")).toBe("");
    expect(agentMonogram(null)).toBe("");
  });
});

describe("agentSigil", () => {
  it("is the same every time for the same agent", () => {
    expect(agentSigil("agdef_01JABC")).toEqual(agentSigil("agdef_01JABC"));
  });

  it("stays on the declared vocabulary", () => {
    for (let index = 0; index < 200; index++) {
      const sigil = agentSigil(`agdef_${index}`);
      expect([0, 45, 90, 135, 180, 225, 270, 315]).toContain(sigil.rotation);
      expect([14, 22, 30]).toContain(sigil.length);
    }
  });

  // The reason the sigil exists. With eight accents and a fallback icon,
  // unconfigured agents collide on both constantly; the arc is what is left
  // to tell them apart, so it has to actually spread.
  it("spreads agents across its variants", () => {
    const seen = new Set<string>();
    for (let index = 0; index < 200; index++) {
      const sigil = agentSigil(`agdef_${index}`);
      seen.add(`${sigil.rotation}:${sigil.length}`);
    }
    expect(seen.size).toBeGreaterThan(18);
  });
});

describe("resolveAgentIdentity icon provenance", () => {
  it("knows an icon the agent chose", () => {
    expect(resolveAgentIdentity({ id: "agdef_1", icon: "truck" }).iconChosen).toBe(true);
  });

  it("knows an icon its starter implies", () => {
    expect(
      resolveAgentIdentity({ id: "agdef_1", template: "ComplianceAssistant" }).iconChosen,
    ).toBe(true);
  });

  // The fallback is the same robot for every unconfigured agent, so the mark
  // has to know it is a fallback and draw initials instead.
  it("knows when it is only the fallback", () => {
    expect(resolveAgentIdentity({ id: "agdef_1", name: "Night dispatch" }).iconChosen).toBe(false);
    expect(resolveAgentIdentity({ id: "agdef_1", icon: "skull" }).iconChosen).toBe(false);
  });
});
