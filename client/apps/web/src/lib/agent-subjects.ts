import { AgentSubjectRecordKind } from "@trenova/shared/types/generated/record-kinds";
import { isRecordEntityType, recordPath } from "@/config/record-links";

/**
 * Where the record an agent run or exception is about opens, or null when
 * that kind of record has no page of its own (a document, an insight, a bank
 * receipt are read where the agent's work is). The subject's kind comes from
 * the server (`AgentSubjectRecordKind`) and its address from the record-link
 * registry, so a subject whose kind gains a page links to it with no change
 * here.
 */
export function agentSubjectPath(subjectType: string, subjectId: string): string | null {
  if (subjectId === "" || !Object.hasOwn(AgentSubjectRecordKind, subjectType)) {
    return null;
  }
  const kind: string = AgentSubjectRecordKind[subjectType as keyof typeof AgentSubjectRecordKind];

  return isRecordEntityType(kind) ? recordPath(kind, subjectId) : null;
}
