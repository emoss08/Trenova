import type { AssistantPageContext, CaseSubjectType } from "@/types/assistant";
import { limitUtf8Bytes } from "@trenova/shared/lib/utils";

/** The bound the server holds a page's title to, in UTF-8 bytes (agentpagecontext.PageContext). */
const MAX_TITLE = 200;

// i18n-ignore: record kinds mapped to the server's case subject enum, never shown to a person
const CASE_SUBJECTS: Readonly<Record<string, CaseSubjectType>> = {
  shipment: "Shipment",
  invoice: "Invoice",
  invoice_dispute: "InvoiceDispute",
};

// i18n-ignore: record kinds mapped to the server's subject enum, never shown to a person
const THREAD_SUBJECTS: Readonly<Record<string, string>> = { ...CASE_SUBJECTS, worker: "Worker" };

/**
 * The subject a conversation asked on a record's case is bound to, so the
 * case finds its conversation again and the rail lists the two as one. Wider
 * than `caseSubjectOf`: a worker's conversation is about the worker though no
 * checklist case can be.
 */
export function threadSubjectOf(kind: string): string | null {
  return THREAD_SUBJECTS[kind] ?? null;
}

const SUBJECT_KINDS: Readonly<Record<string, string>> = Object.fromEntries(
  Object.entries(THREAD_SUBJECTS).map(([kind, subject]) => [subject, kind]),
);

/**
 * The record a conversation is the case conversation of, as the kind and id
 * its case opens on; null for a conversation about no record. It is the
 * reverse of `threadSubjectOf`, so a conversation asked on a case is found
 * again as that case wherever it is listed.
 */
export function caseOfThread(thread: {
  subjectType?: string | null;
  subjectId?: string | null;
}): { kind: string; id: string } | null {
  const kind = thread.subjectType ? SUBJECT_KINDS[thread.subjectType] : undefined;

  return kind && thread.subjectId ? { kind, id: thread.subjectId } : null;
}

/** The case a record's conversation binds to, when its kind is one a case can be about. */
export function caseSubjectOf(kind: string): CaseSubjectType | null {
  return CASE_SUBJECTS[kind] ?? null;
}

/**
 * The record as the page a margin conversation is about, so every question
 * asked beside its case carries the record the person is looking at.
 */
export function casePageContext(
  path: string,
  kind: string,
  id: string,
  title: string,
): AssistantPageContext {
  return {
    path,
    entityType: kind,
    entityId: id,
    title: limitUtf8Bytes(title, MAX_TITLE),
    view: null,
    draft: null,
  };
}

/**
 * Names the case as the region a view transition moves (desk-motion.css),
 * unless motion is reduced by the system or by the Desk's own setting.
 */
// i18n-ignore: class names, never shown to a person
export const CASE_TRANSITION_CLASS =
  "[view-transition-name:dk-case] motion-reduce:[view-transition-name:none] in-[.dk-calm]:[view-transition-name:none]";

/**
 * The conversation a record's case continues: the person's conversation about
 * the record with the agent working its case. A conversation another agent
 * had about the same record is a different conversation, and asking the case
 * a question never lands in it. With no agent on the case, any conversation
 * about the record will do.
 */
export function caseThreadOf<
  T extends { subjectType: string; subjectId?: string | null; agentDefinitionId: string },
>(threads: readonly T[], kind: string, id: string, agentId: string | null | undefined): T | null {
  const subjectType = threadSubjectOf(kind);
  if (subjectType === null) {
    return null;
  }

  return (
    threads.find(
      (thread) =>
        thread.subjectType === subjectType &&
        thread.subjectId === id &&
        (!agentId || thread.agentDefinitionId === agentId),
    ) ?? null
  );
}
