import { getFragmentData, type FragmentType } from "@trenova/graphql/fragment-data";
import {
  AcceptDeskCaseRiskDocument,
  BringBackDeskCaseDocument,
  PutAwayDeskCaseDocument,
  AssignDeskCaseDocument,
  ChooseDeskCaseOptionDocument,
  CommitAgentDecisionNowDocument,
  DeskCaseDecisionFieldsFragmentDoc,
  DeskCaseDocument,
  DeskCaseExhibitFieldsFragmentDoc,
  DeskRecordFieldsFragmentDoc,
  ReleaseDeskCaseDocument,
  UndoAgentDecisionDocument,
  WatchtowerAgentFieldsFragmentDoc,
  type DeskCaseDecisionFieldsFragment,
  type DeskCaseDecisionPhase,
  type DeskCaseExhibitFieldsFragment,
  type DeskCaseQuery,
  type DeskCaseTone,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { ApprovalTarget } from "@/lib/graphql/agent-decisions";
import type { DeskRecord } from "@/lib/graphql/desk-attention";
import type { WatchtowerAgent } from "@/lib/graphql/watchtower";

export type { DeskCaseDecisionPhase, DeskCaseTone };

type RawCase = DeskCaseQuery["deskCase"];
type RawFinding = NonNullable<RawCase["finding"]>;

export type DeskCaseHead = RawCase["head"];
export type DeskCaseOption = RawCase["options"][number];
export type DeskCaseSegment = RawFinding["reading"][number];
export type DeskCaseTrustCell = RawFinding["trust"][number];

export type DeskCaseFinding = Omit<RawFinding, "agent"> & { agent: WatchtowerAgent | null };

export type DeskCaseDecision = Omit<DeskCaseDecisionFieldsFragment, "agent" | " $fragmentName"> & {
  agent: WatchtowerAgent | null;
};

type RawExhibit = Omit<DeskCaseExhibitFieldsFragment, " $fragmentName">;
export type DeskCaseMessage = Omit<NonNullable<RawExhibit["message"]>, "replyAgent"> & {
  replyAgent: WatchtowerAgent | null;
};
export type DeskCaseExhibit = Omit<RawExhibit, "message"> & { message: DeskCaseMessage | null };
export type DeskCaseRoute = NonNullable<DeskCaseExhibit["route"]>;

/**
 * What lets a person put an agent on a case nobody has worked: the condition
 * to hand off, its kind, and what the last agent on it said if it proposed
 * nothing.
 */
export type DeskCaseHandOff = Omit<NonNullable<RawCase["handOff"]>, "agent"> & {
  agent: WatchtowerAgent | null;
};

/** Who a case belongs to and how the reader stands to it. */
export type DeskCaseOwnership = RawCase["ownership"];

export type DeskCase = {
  record: DeskRecord;
  ownership: DeskCaseOwnership;
  /** Set while the case is put away: when it comes back if nothing new happens first. */
  away: { until: number } | null;
  /** The condition an agent can be asked to work; null when there is none to hand off. */
  handOff: DeskCaseHandOff | null;
  head: DeskCaseHead;
  finding: DeskCaseFinding | null;
  options: DeskCaseOption[];
  drafting: boolean;
  /** What the agent said when it finished without a plan for the chosen option; markdown. */
  draftNote: string;
  decision: DeskCaseDecision | null;
  exhibits: DeskCaseExhibit[];
  note: string;
  starters: string[];
  open: number;
  severity: RawCase["severity"];
};

type RequestOptions = { signal?: AbortSignal };

type AgentRef = FragmentType<typeof WatchtowerAgentFieldsFragmentDoc>;

function agentOf(ref: AgentRef | null | undefined): WatchtowerAgent | null {
  return ref ? getFragmentData(WatchtowerAgentFieldsFragmentDoc, ref) : null;
}

/** A record's case in one read, with its times told in the reader's zone. */
export async function fetchDeskCase(
  kind: string,
  id: string,
  timezone: string,
  options?: RequestOptions,
): Promise<DeskCase> {
  const data = await requestGraphQL({
    document: DeskCaseDocument,
    operationName: "DeskCase",
    variables: { kind, id, timezone },
    signal: options?.signal,
  });
  const result = data.deskCase;
  const decision = result.decision
    ? getFragmentData(DeskCaseDecisionFieldsFragmentDoc, result.decision)
    : null;

  return {
    record: getFragmentData(DeskRecordFieldsFragmentDoc, result.record),
    ownership: result.ownership,
    away: result.away ?? null,
    handOff: result.handOff ? { ...result.handOff, agent: agentOf(result.handOff.agent) } : null,
    head: result.head,
    finding: result.finding ? { ...result.finding, agent: agentOf(result.finding.agent) } : null,
    options: result.options,
    drafting: result.drafting,
    draftNote: result.draftNote,
    decision: decision ? { ...decision, agent: agentOf(decision.agent) } : null,
    exhibits: result.exhibits.map((ref) => {
      const exhibit = getFragmentData(DeskCaseExhibitFieldsFragmentDoc, ref);
      return {
        ...exhibit,
        message: exhibit.message
          ? { ...exhibit.message, replyAgent: agentOf(exhibit.message.replyAgent) }
          : null,
      };
    }),
    note: result.note,
    starters: result.starters,
    open: result.open,
    severity: result.severity,
  };
}

/** Chooses one of the options on a record's case; the agent drafts its plan when none exists. */
export async function chooseDeskCaseOption(kind: string, id: string, optionId: string) {
  const data = await requestGraphQL({
    document: ChooseDeskCaseOptionDocument,
    operationName: "ChooseDeskCaseOption",
    variables: { kind, id, optionId },
  });

  return data.chooseDeskCaseOption;
}

/** Marks the conditions about a record as needing no action, with the reason given. */
export async function acceptDeskCaseRisk(kind: string, id: string, reason: string) {
  const data = await requestGraphQL({
    document: AcceptDeskCaseRiskDocument,
    operationName: "AcceptDeskCaseRisk",
    variables: { kind, id, reason },
  });

  return data.acceptDeskCaseRisk;
}

/**
 * Puts a record's case away until a time: what is still open about it is
 * snoozed, so it leaves Needs you, and comes back then or when something new
 * is raised. Answers with how many conditions were put away.
 */
export async function putAwayDeskCase(kind: string, id: string, until: number) {
  const data = await requestGraphQL({
    document: PutAwayDeskCaseDocument,
    operationName: "PutAwayDeskCase",
    variables: { kind, id, until },
  });

  return data.putAwayDeskCase;
}

/** Brings a put-away case back in front before its time. */
export async function bringBackDeskCase(kind: string, id: string) {
  const data = await requestGraphQL({
    document: BringBackDeskCaseDocument,
    operationName: "BringBackDeskCase",
    variables: { kind, id },
  });

  return data.bringBackDeskCase;
}

/**
 * Gives a record's case to an owner. With no owner named the caller claims
 * it, or takes it over when they may assign cases.
 */
export async function assignDeskCase(kind: string, id: string, ownerId: string | null) {
  const data = await requestGraphQL({
    document: AssignDeskCaseDocument,
    operationName: "AssignDeskCase",
    variables: { kind, id, ownerId },
  });

  return data.assignDeskCase;
}

/** Leaves a record's case unowned, for the next person to claim. */
export async function releaseDeskCase(kind: string, id: string) {
  const data = await requestGraphQL({
    document: ReleaseDeskCaseDocument,
    operationName: "ReleaseDeskCase",
    variables: { kind, id },
  });

  return data.releaseDeskCase;
}

function targetVariables(target: ApprovalTarget) {
  return "planId" in target
    ? { proposalId: null, planId: target.planId }
    : { proposalId: target.proposalId, planId: null };
}

/** Takes back an approval made from the queue or a case while its undo window is open. */
export async function undoAgentDecision(target: ApprovalTarget) {
  const data = await requestGraphQL({
    document: UndoAgentDecisionDocument,
    operationName: "UndoAgentDecision",
    variables: targetVariables(target),
  });

  return data.undoAgentDecision;
}

/** Ends the undo window on an approval made from the queue or a case, so it runs now. */
export async function commitAgentDecisionNow(target: ApprovalTarget) {
  const data = await requestGraphQL({
    document: CommitAgentDecisionNowDocument,
    operationName: "CommitAgentDecisionNow",
    variables: targetVariables(target),
  });

  return data.commitAgentDecisionNow;
}
