import type { TriageRule } from "@/lib/graphql/inbox";
import { describe, expect, it } from "vitest";
import {
  EMPTY_DRAFT,
  draftFromRule,
  draftFromSeed,
  isComplete,
  missingBlanks,
  ruleInput,
  sameRule,
  senderOptions,
  senderShortName,
  type RuleDraft,
} from "../rule-model";

const COMPLETE: RuleDraft = {
  classification: "ProofOfDelivery",
  sender: { scope: "Domain", value: "drivers.trenova.example" },
  mailboxId: "imb_intake",
  action: "FileOnShipment",
  assigneeId: null,
};

const USER = { id: "usr_dana", name: "Dana", emailAddress: "", thumbnailUrl: "" };

function rule(overrides: Partial<TriageRule> = {}): TriageRule {
  return {
    id: "rule_1",
    classification: "Invoice",
    senderScope: "Anyone",
    sender: "",
    mailboxId: null,
    action: "FileAndQueuePay",
    assigneeId: null,
    assignee: null,
    enabled: true,
    createdBy: null,
    matchesThisMonth: 61,
    matchCount: 300,
    lastMatchedAt: null,
    lastMatch: null,
    version: 3,
    createdAt: 0,
    ...overrides,
  };
}

describe("a rule's blanks", () => {
  it("starts with only the sender and the source filled, as anyone on any source", () => {
    expect(missingBlanks(EMPTY_DRAFT)).toEqual(["kind", "action"]);
    expect(isComplete(EMPTY_DRAFT)).toBe(false);
  });

  it("is complete with a kind, a sender and an action; the source may stay any source", () => {
    expect(isComplete(COMPLETE)).toBe(true);
    expect(isComplete({ ...COMPLETE, mailboxId: null })).toBe(true);
    expect(missingBlanks({ ...COMPLETE, sender: null })).toEqual(["sender"]);
  });

  it("asks who to hand it to only when the action is to assign it", () => {
    const assign: RuleDraft = { ...COMPLETE, action: "Assign" };
    expect(missingBlanks(assign)).toEqual(["assignee"]);
    expect(isComplete({ ...assign, assigneeId: USER.id })).toBe(true);
    expect(missingBlanks({ ...COMPLETE, assigneeId: USER.id })).toEqual([]);
  });

  it("takes a seed from outside the page and leaves the rest as a new rule starts", () => {
    expect(draftFromSeed({ classification: "Tender" })).toEqual({
      ...EMPTY_DRAFT,
      classification: "Tender",
    });
    expect(
      draftFromSeed({
        classification: "RateConfirmation",
        senderScope: "Domain",
        sender: "BlueRidgeBrokerage.example",
        mailboxId: "imb_intake",
        action: "FileOnShipment",
      }),
    ).toEqual({
      classification: "RateConfirmation",
      sender: { scope: "Domain", value: "blueridgebrokerage.example" },
      mailboxId: "imb_intake",
      action: "FileOnShipment",
      assigneeId: null,
    });
    expect(draftFromSeed(null)).toEqual(EMPTY_DRAFT);
  });

  it("reads a saved rule back into its blanks", () => {
    expect(draftFromRule(rule({ senderScope: "AnyCarrier" }))).toEqual({
      classification: "Invoice",
      sender: { scope: "AnyCarrier" },
      mailboxId: null,
      action: "FileAndQueuePay",
      assigneeId: null,
    });
  });
});

describe("what a rule sends", () => {
  it("is nothing until every blank is filled", () => {
    expect(ruleInput(EMPTY_DRAFT)).toBeNull();
    expect(ruleInput({ ...COMPLETE, action: "Assign" })).toBeNull();
  });

  it("names the domain or address only for those scopes", () => {
    expect(ruleInput(COMPLETE)).toEqual({
      classification: "ProofOfDelivery",
      senderScope: "Domain",
      sender: "drivers.trenova.example",
      mailboxId: "imb_intake",
      action: "FileOnShipment",
      assigneeId: null,
      enabled: true,
    });
    expect(ruleInput({ ...COMPLETE, sender: { scope: "AnyCustomer" }, mailboxId: null })).toEqual({
      classification: "ProofOfDelivery",
      senderScope: "AnyCustomer",
      sender: null,
      mailboxId: null,
      action: "FileOnShipment",
      assigneeId: null,
      enabled: true,
    });
  });

  it("carries the assignee only for an assign rule, and keeps a rule's on or off", () => {
    expect(ruleInput({ ...COMPLETE, assigneeId: USER.id })?.assigneeId).toBeNull();
    expect(
      ruleInput({ ...COMPLETE, action: "Assign", assigneeId: USER.id }, false),
    ).toMatchObject({ action: "Assign", assigneeId: USER.id, enabled: false });
  });

  it("knows when an edit changes nothing, so nothing is saved", () => {
    const saved = rule();
    expect(sameRule(saved, draftFromRule(saved))).toBe(true);
    expect(sameRule(saved, { ...draftFromRule(saved), mailboxId: "imb_billing" })).toBe(false);
  });
});

describe("who a rule's messages come from", () => {
  const domains = ["acme.example", "blueridgebrokerage.example", "drivers.trenova.example"];

  it("offers anyone, any customer, any carrier and the domains that have written in", () => {
    expect(senderOptions(domains, "").map((option) => option.key)).toEqual([
      "Anyone",
      "AnyCustomer",
      "AnyCarrier",
      "Domain:acme.example",
      "Domain:blueridgebrokerage.example",
      "Domain:drivers.trenova.example",
    ]);
  });

  it("narrows the domains to what is typed", () => {
    expect(senderOptions(domains, "Blue").map((option) => option.key)).toEqual([
      "Domain:blueridgebrokerage.example",
    ]);
  });

  it("offers a typed address as one sender", () => {
    expect(senderOptions(domains, " Ops@Acme.example ")).toEqual([
      {
        key: "Address:ops@acme.example",
        choice: { scope: "Address", value: "ops@acme.example" },
        typed: true,
      },
    ]);
  });

  it("offers a typed domain nobody has written from yet, but not one already listed", () => {
    expect(senderOptions(domains, "@newcarrier.example").map((option) => option.key)).toEqual([
      "Domain:newcarrier.example",
    ]);
    expect(senderOptions(domains, "acme.example").map((option) => option.key)).toEqual([
      "Domain:acme.example",
    ]);
  });

  it("offers nothing typed for text that is neither an address nor a domain", () => {
    expect(senderOptions(domains, "not a domain")).toEqual([]);
  });

  it("shortens a domain to its name for a sentence, and keeps an address whole", () => {
    expect(senderShortName("Domain", "blueridgebrokerage.example")).toBe("blueridgebrokerage");
    expect(senderShortName("Address", "ops@acme.example")).toBe("ops@acme.example");
  });
});
