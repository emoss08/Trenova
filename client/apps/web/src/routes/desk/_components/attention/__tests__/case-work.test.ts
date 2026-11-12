import type { DeskAttentionGroup, DeskAttentionItem } from "@/lib/graphql/desk-attention";
import type { WatchtowerAgentWork } from "@/lib/graphql/watchtower";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { watchtowerItem } from "../../watchtower/__tests__/item-fixture";
import { caseWorkLabel, caseWorkOf, caseWorkShort, itemsWorkedElsewhere } from "../case-work";

const t = ((text: string, ...args: unknown[]) =>
  text.replace(/\{(\d+)\}/g, (_match, index: string) =>
    String(args[Number(index)]),
  )) as unknown as TranslateFn;

function item(overrides: Partial<DeskAttentionItem> = {}): DeskAttentionItem {
  return { ...watchtowerItem(), deskLens: "Watchtower", aboutId: "shp_1", ...overrides };
}

function work(overrides: Partial<WatchtowerAgentWork> = {}): WatchtowerAgentWork {
  return {
    agentId: "agdef_1",
    agent: null,
    runId: "ar_1",
    status: "Working",
    step: 0,
    steps: [],
    note: "",
    failed: false,
    handedAt: 1_800_000_000,
    handedById: null,
    handedBy: null,
    ...overrides,
  } as WatchtowerAgentWork;
}

function group(overrides: Partial<DeskAttentionGroup> = {}): DeskAttentionGroup {
  return {
    key: "shp_1",
    record: {
      kind: "shipment",
      noun: "Shipment",
      id: "shp_1",
      label: "S-240412",
      subtitle: "",
      status: "",
      readable: true,
    },
    severity: "Warning",
    dueAt: null,
    dueLabel: null,
    unseen: false,
    items: [item()],
    ownership: { owner: null, auto: false, canWork: true },
    ...overrides,
  };
}

const DANA = { id: "usr_1", name: "Dana Ruiz" };

describe("whether a row of Needs you is being worked in its case", () => {
  it("is not while nobody has picked it up, whoever the record would route it to", () => {
    expect(caseWorkOf(group())).toBeNull();
    expect(
      caseWorkOf(group({ ownership: { owner: DANA, auto: true, canWork: false } })),
    ).toBeNull();
    expect(caseWorkOf(group({ ownership: null }))).toBeNull();
  });

  it("is once a person took the case or was given it", () => {
    const theirs = caseWorkOf(group({ ownership: { owner: DANA, auto: false, canWork: false } }));
    const mine = caseWorkOf(group({ ownership: { owner: DANA, auto: false, canWork: true } }));

    expect(theirs).toEqual({ owner: DANA, mine: false });
    expect(caseWorkLabel(t, theirs!)).toBe("Dana Ruiz's case");
    expect(caseWorkLabel(t, mine!)).toBe("Your case");
    expect(caseWorkShort(t, theirs!)).toBe("Dana Ruiz");
    expect(caseWorkShort(t, mine!)).toBe("You");
  });

  it("is while an agent it was handed to is on it, and not once the agent hands it back", () => {
    const handed = caseWorkOf(group({ items: [item(), item({ id: "wt_2", agentWork: work() })] }));

    expect(handed).toEqual({ owner: null, mine: true });
    expect(caseWorkLabel(t, handed!)).toBe("In a case");
    expect(caseWorkShort(t, handed!)).toBe("");
    expect(
      caseWorkOf(group({ items: [item({ agentWork: work({ status: "NeedsYou" }) })] })),
    ).not.toBeNull();
    expect(
      caseWorkOf(group({ items: [item({ agentWork: work({ status: "Done" }) })] })),
    ).toBeNull();
    expect(caseWorkOf(group({ items: [item({ agentWork: work({ failed: true }) })] }))).toBeNull();
  });

  it("is while a change about the record waits to be approved", () => {
    expect(
      caseWorkOf(group({ items: [item(), item({ id: "ap_1", deskLens: "Decisions" })] })),
    ).toEqual({ owner: null, mine: true });
  });

  it("never is for a row with no case to open", () => {
    const owned = { owner: DANA, auto: false, canWork: true };

    expect(caseWorkOf(group({ record: null, ownership: owned }))).toBeNull();
    expect(
      caseWorkOf(
        group({
          record: { ...group().record!, readable: false, label: "" },
          ownership: owned,
        }),
      ),
    ).toBeNull();
  });

  it("names every item of a worked row, and none of an unworked one", () => {
    const worked = group({
      key: "shp_1",
      items: [item({ id: "wt_1" }), item({ id: "wt_2" })],
      ownership: { owner: DANA, auto: false, canWork: true },
    });
    const open = group({ key: "shp_2", items: [item({ id: "wt_3" })] });

    expect([...itemsWorkedElsewhere([worked, open])]).toEqual(["wt_1", "wt_2"]);
  });

  it("names a message on an unworked row, which is worked in Triage, and not the condition beside it", () => {
    const mixed = group({
      key: "shp_4",
      items: [item({ id: "wt_7" }), item({ id: "wt_8", deskLens: "Triage" })],
    });

    expect([...itemsWorkedElsewhere([mixed])]).toEqual(["wt_8"]);
  });
});
