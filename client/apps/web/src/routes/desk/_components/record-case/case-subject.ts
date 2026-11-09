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
