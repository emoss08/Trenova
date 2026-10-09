import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import {
  canDelegate,
  delegateCandidates,
  delegatesLine,
  savedDelegates,
  type DelegateSummary,
} from "../delegates";

const t = translate;

function delegate(name: string, enabled = true): DelegateSummary {
  return { id: `agdef_${name}`, name, icon: "", accent: "", enabled, triggerMode: "Chat" };
}

describe("delegatesLine", () => {
  it("says nothing for an agent that asks nobody", () => {
    expect(delegatesLine([], t)).toBe("");
  });

  it("names up to two agents and counts the rest", () => {
    expect(delegatesLine([delegate("Report Builder")], t)).toBe("Can ask Report Builder");
    expect(delegatesLine([delegate("Report Builder"), delegate("Dispatch desk")], t)).toBe(
      "Can ask Report Builder, Dispatch desk",
    );
    expect(
      delegatesLine(
        [delegate("Report Builder"), delegate("Dispatch desk"), delegate("Billing", false)],
        t,
      ),
    ).toBe("Can ask Report Builder, Dispatch desk +1");
  });
});

describe("savedDelegates", () => {
  // `delegates` omits an agent deleted since; `delegateIds` keeps the order.
  it("keeps the configured order and drops an id with no agent behind it", () => {
    const listed = [delegate("A"), delegate("B", false)];
    expect(
      savedDelegates({ delegateIds: ["agdef_B", "agdef_gone", "agdef_A"], delegates: listed }).map(
        (entry) => entry.id,
      ),
    ).toEqual(["agdef_B", "agdef_A"]);
  });

  it("keeps an agent the server lists but did not order, after the ordered ones", () => {
    expect(
      savedDelegates({
        delegateIds: ["agdef_A"],
        delegates: [delegate("B"), delegate("A")],
      }).map((entry) => entry.id),
    ).toEqual(["agdef_A", "agdef_B"]);
  });
});

describe("canDelegate", () => {
  it("allows an allowlist only on an agent people talk to", () => {
    expect(canDelegate("Chat")).toBe(true);
    expect(canDelegate("Scheduled")).toBe(false);
    expect(canDelegate("Event")).toBe(false);
    expect(canDelegate("Continuous")).toBe(false);
  });
});

describe("delegateCandidates", () => {
  type Candidate = Parameters<typeof delegateCandidates>[0][number];
  function agent(id: string, patch: Partial<Candidate> = {}): Candidate {
    return { id, enabled: true, triggerMode: "Chat", systemKey: "", ...patch };
  }

  const roster = [
    agent("self"),
    agent("chat"),
    agent("off", { enabled: false }),
    agent("scheduled", { triggerMode: "Scheduled" }),
    agent("event", { triggerMode: "Event" }),
    agent("watch", { triggerMode: "Continuous" }),
    agent("import", { systemKey: "import_assistant" }),
    agent("formula", { systemKey: "formula_assistant" }),
    agent("system-chat", { systemKey: "dispatch_desk" }),
  ];

  it("offers only enabled agents people talk to, other than itself and page agents", () => {
    expect(
      delegateCandidates(roster, { selfId: "self", chosen: [] }).map((entry) => [
        entry.agent.id,
        entry.unavailable,
      ]),
    ).toEqual([
      ["chat", null],
      ["system-chat", null],
    ]);
  });

  it("offers every agent but none of the excluded ones for a new agent", () => {
    expect(
      delegateCandidates(roster, { selfId: null, chosen: [] }).map((entry) => entry.agent.id),
    ).toEqual(["self", "chat", "system-chat"]);
  });

  it("keeps an agent already chosen that can no longer be asked, so it can be taken off", () => {
    expect(
      delegateCandidates(roster, { selfId: "self", chosen: ["off", "scheduled", "import"] }).map(
        (entry) => [entry.agent.id, entry.unavailable],
      ),
    ).toEqual([
      ["chat", null],
      ["off", "disabled"],
      ["scheduled", "background"],
      ["import", "page"],
      ["system-chat", null],
    ]);
  });
});
