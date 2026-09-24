import type { AgentSubjectType } from "@trenova/graphql/generated/graphql";
import { recordPath, type RecordEntityType } from "@/config/record-links";

/**
 * The record-link entity each agent subject opens as, for the subjects that
 * have a page of their own. The rest (a document, an insight, a bank receipt)
 * are read where the agent's work is, not on a record page.
 */
const SUBJECT_RECORDS: Partial<Record<AgentSubjectType, RecordEntityType>> = {
  AssistantThread: "assistant_thread",
  BillingQueueItem: "billing_queue_item",
  Dashboard: "dashboard",
  EDIInboundFile: "edi_inbound_file",
  Report: "report",
  Shipment: "shipment",
  Worker: "worker",
};

/**
 * Where the record an agent run or exception is about opens, or null when
 * that kind of record has no page of its own. Built from the record-link
 * registry, like every other record's address.
 */
export function agentSubjectPath(subjectType: string, subjectId: string): string | null {
  const entity = SUBJECT_RECORDS[subjectType as AgentSubjectType];
  if (entity === undefined || subjectId === "") {
    return null;
  }

  return recordPath(entity, subjectId);
}
