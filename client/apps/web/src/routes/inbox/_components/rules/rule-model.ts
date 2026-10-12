import type {
  InboundClassification,
  TriageRule,
  TriageRuleAction,
  TriageRuleInput,
  TriageSenderScope,
} from "@/lib/graphql/inbox";
import type { RuleSeed } from "@/stores/triage-store";

/**
 * A rule is a sentence with blanks: "When a {kind} arrives from {sender} on
 * {source}, Nova should {action}." This module is the sentence without the
 * words — which blanks are filled, what a filled sentence sends to the
 * server, and who can fill the sender blank — so the add row and every saved
 * rule agree on it.
 */
export type SenderChoice =
  | { scope: "Anyone" }
  | { scope: "AnyCustomer" }
  | { scope: "AnyCarrier" }
  | { scope: "Domain"; value: string }
  | { scope: "Address"; value: string };

export type RuleDraft = {
  classification: InboundClassification | null;
  sender: SenderChoice | null;
  /** Null is every source. */
  mailboxId: string | null;
  action: TriageRuleAction | null;
  /** Read only when the action is to assign it. */
  assigneeId: string | null;
};

export type RuleBlank = "kind" | "sender" | "source" | "action" | "assignee";

/** A new rule: from anyone, on any source, with the kind and the action left to fill. */
export const EMPTY_DRAFT: RuleDraft = {
  classification: null,
  sender: { scope: "Anyone" },
  mailboxId: null,
  action: null,
  assigneeId: null,
};

/** Lower case, no surrounding space and no leading "@", as the server compares senders. */
export function normalizeSender(text: string): string {
  return text.trim().toLowerCase().replace(/^@+/, "");
}

const ADDRESS = /^[^\s@]+@[a-z0-9-]+(\.[a-z0-9-]+)+$/;
const DOMAIN = /^[a-z0-9-]+(\.[a-z0-9-]+)+$/;

export function senderChoiceOf(scope: TriageSenderScope, sender: string): SenderChoice | null {
  switch (scope) {
    case "Anyone":
    case "AnyCustomer":
    case "AnyCarrier":
      return { scope };
    case "Domain":
    case "Address": {
      const value = normalizeSender(sender);
      return value === "" ? null : { scope, value };
    }
  }
}

/** One key per sender, for a list's identity and for comparing two choices. */
export function senderKey(choice: SenderChoice): string {
  return choice.scope === "Domain" || choice.scope === "Address"
    ? `${choice.scope}:${choice.value}`
    : choice.scope;
}

export function draftFromSeed(seed: RuleSeed | null): RuleDraft {
  if (seed === null) {
    return EMPTY_DRAFT;
  }
  const sender =
    seed.senderScope === undefined
      ? EMPTY_DRAFT.sender
      : senderChoiceOf(seed.senderScope, seed.sender ?? "");

  return {
    classification: seed.classification ?? null,
    sender,
    mailboxId: seed.mailboxId ?? null,
    action: seed.action ?? null,
    assigneeId: null,
  };
}

export function draftFromRule(rule: TriageRule): RuleDraft {
  return {
    classification: rule.classification,
    sender: senderChoiceOf(rule.senderScope, rule.sender),
    mailboxId: rule.mailboxId ?? null,
    action: rule.action,
    assigneeId: rule.action === "Assign" ? (rule.assigneeId ?? null) : null,
  };
}

/** The blanks still empty, in the order the sentence reads them. */
export function missingBlanks(draft: RuleDraft): RuleBlank[] {
  const missing: RuleBlank[] = [];
  if (draft.classification === null) {
    missing.push("kind");
  }
  if (draft.sender === null) {
    missing.push("sender");
  }
  if (draft.action === null) {
    missing.push("action");
  }
  if (draft.action === "Assign" && draft.assigneeId === null) {
    missing.push("assignee");
  }

  return missing;
}

export function isComplete(draft: RuleDraft): boolean {
  return missingBlanks(draft).length === 0;
}

/** What a filled sentence sends; null while a blank is empty. */
export function ruleInput(draft: RuleDraft, enabled = true): TriageRuleInput | null {
  if (!isComplete(draft) || draft.classification === null || draft.sender === null) {
    return null;
  }
  const { sender } = draft;
  const action = draft.action as TriageRuleAction;

  return {
    classification: draft.classification,
    senderScope: sender.scope,
    sender: sender.scope === "Domain" || sender.scope === "Address" ? sender.value : null,
    mailboxId: draft.mailboxId,
    action,
    assigneeId: action === "Assign" ? draft.assigneeId : null,
    enabled,
  };
}

/** Whether a saved rule already says what the draft says, so an edit has nothing to save. */
export function sameRule(rule: TriageRule, draft: RuleDraft): boolean {
  const saved = draftFromRule(rule);

  return (
    saved.classification === draft.classification &&
    saved.mailboxId === draft.mailboxId &&
    saved.action === draft.action &&
    saved.assigneeId === draft.assigneeId &&
    (saved.sender === null || draft.sender === null
      ? saved.sender === draft.sender
      : senderKey(saved.sender) === senderKey(draft.sender))
  );
}

export type SenderOption = {
  key: string;
  choice: SenderChoice;
  /** Made from what was typed rather than from mail that arrived. */
  typed: boolean;
};

const FIXED: readonly SenderChoice[] = [
  { scope: "Anyone" },
  { scope: "AnyCustomer" },
  { scope: "AnyCarrier" },
];

function option(choice: SenderChoice, typed: boolean): SenderOption {
  return { key: senderKey(choice), choice, typed };
}

/**
 * Who the sender blank offers. With nothing typed: anyone, any customer, any
 * carrier, then every domain that has written in. Typing narrows the domains
 * and, when it is an address or a domain not yet seen, offers it as typed.
 */
export function senderOptions(domains: readonly string[], query: string): SenderOption[] {
  const typed = normalizeSender(query);
  const known = [...new Set(domains.map(normalizeSender).filter((domain) => domain !== ""))];
  if (typed === "") {
    return [
      ...FIXED.map((choice) => option(choice, false)),
      ...known.map((value) => option({ scope: "Domain", value }, false)),
    ];
  }

  const matching = known
    .filter((domain) => domain.includes(typed))
    .map((value) => option({ scope: "Domain", value }, false));
  if (ADDRESS.test(typed)) {
    return [option({ scope: "Address", value: typed }, true), ...matching];
  }
  if (DOMAIN.test(typed) && !known.includes(typed)) {
    return [option({ scope: "Domain", value: typed }, true), ...matching];
  }

  return matching;
}

/** A sender as a sentence names it in passing: a domain by its name, an address whole. */
export function senderShortName(scope: TriageSenderScope, sender: string): string {
  if (scope === "Domain") {
    return normalizeSender(sender).split(".")[0] ?? sender;
  }

  return sender;
}
