import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { deskTitle, type DeskTitleParts } from "../desk-title";

const t = ((text: string, ...args: unknown[]) =>
  text.replace(/\{(\d+)\}/g, (_, index: string) => String(args[Number(index)]))) as TranslateFn;

function title(parts: Partial<DeskTitleParts>): string {
  return deskTitle(t, {
    place: "today",
    threadTitle: null,
    agentName: null,
    caseLabel: null,
    pageTitle: null,
    lens: null,
    ...parts,
  });
}

describe("the browser tab while the Desk is in front", () => {
  it("names the place, then the Desk", () => {
    expect(title({ place: "today" })).toBe("Today · Desk");
    expect(title({ place: "scheduled" })).toBe("Scheduled · Desk");
    expect(title({ place: "memory" })).toBe("Memory · Desk");
  });

  it("names the conversation in front, and says so when it has no name yet", () => {
    expect(title({ place: "thread", threadTitle: "Late loads this week" })).toBe(
      "Late loads this week · Desk",
    );
    expect(title({ place: "thread", threadTitle: "  " })).toBe("Untitled conversation · Desk");
  });

  it("is the Desk alone while the conversation in front is still being read", () => {
    expect(title({ place: "thread", threadTitle: null })).toBe("Desk");
  });

  it("names the record of a case, and the case alone until its record is read", () => {
    expect(title({ place: "case", caseLabel: "SEED-SHP-005" })).toBe("SEED-SHP-005 · Case · Desk");
    expect(title({ place: "case" })).toBe("Case · Desk");
  });

  it("names the lens of Needs you only when one is chosen", () => {
    expect(title({ place: "needs" })).toBe("Needs you · Desk");
    expect(title({ place: "needs", lens: "watchtower" })).toBe("Watchtower · Needs you · Desk");
    expect(title({ place: "needs", lens: "nonsense" })).toBe("Needs you · Desk");
  });

  it("names an agent's page, a view and a handled line by what they call themselves", () => {
    expect(title({ place: "agent", agentName: "Credential desk" })).toBe("Credential desk · Desk");
    expect(title({ place: "view", pageTitle: "Open detention" })).toBe("Open detention · Desk");
    expect(title({ place: "handled" })).toBe("Agents handled · Desk");
  });
});
