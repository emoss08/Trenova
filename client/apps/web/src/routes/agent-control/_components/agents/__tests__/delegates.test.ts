import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import { canDelegate, delegatesLine, savedDelegates, type DelegateSummary } from "../delegates";

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
