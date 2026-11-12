import { isPreviewConflict } from "@/components/assistant/proposal-preview/preview-gate";
import {
  visibleWarnings,
  wouldFailReasons,
  type WouldFailReason,
} from "@/components/assistant/proposal-preview/preview-warnings";
import { approvalFacts } from "@/components/desk-chat/conversation/approval-facts";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { BatchPreviewEntry } from "./batch-review-model";
import type { CardGate } from "./decision-card";
import type { DecisionPreviews } from "./use-decision-previews";
import type { PendingProposalNode } from "./use-pending-decisions";

/** How much a change asks of the person before they decide it. */
export type DecisionFlag = "danger" | "warning" | null;

export function isHeld(node: PendingProposalNode): boolean {
  return node.hold !== null && node.hold !== undefined;
}

/** Why the change would be refused if approved as it stands; empty when it would run. */
export function refusalsOf(entry: BatchPreviewEntry | undefined): WouldFailReason[] {
  return (entry?.preview?.warnings ?? []).flatMap(wouldFailReasons);
}

/**
 * What a row says about its change before it is opened: that it cannot run
 * as it stands, or that there is something to read first (a warning, a
 * record that moved, a preview that could not be read).
 */
export function flagOf(entry: BatchPreviewEntry | undefined): DecisionFlag {
  if (entry === undefined) {
    return null;
  }
  if (entry.preview === null) {
    return "warning";
  }
  if (refusalsOf(entry).length > 0) {
    return "danger";
  }
  if (entry.preview.stale) {
    return "warning";
  }

  return visibleWarnings(entry.preview.warnings, { inPlan: false }).length > 0 ? "warning" : null;
}

/** Whether a change can be approved without its digest: an internal change whose preview failed. */
function unreviewedApprovable(
  node: PendingProposalNode,
  entry: BatchPreviewEntry | undefined,
  failed: boolean,
): boolean {
  return (failed || (entry !== undefined && entry.preview === null)) && !node.reviewRequired;
}

/** What the bar needs to decide the open change alone. */
export function currentGateOf(
  node: PendingProposalNode,
  previews: DecisionPreviews,
  t: TranslateFn,
): CardGate | null {
  if (isHeld(node)) {
    return null;
  }
  const entry = previews.entry(node.id);
  const preview = entry?.preview ?? null;
  const facts = approvalFacts(preview ?? undefined, t);

  return {
    approvable:
      preview !== null
        ? !preview.stale
        : unreviewedApprovable(node, entry, previews.failed(node.id)),
    digest: preview?.digest,
    refused: facts.refused.length,
    count: facts.count,
    handleError: (error) => {
      if (!isPreviewConflict(error)) {
        return false;
      }
      previews.refetch(node.id);
      return true;
    },
  };
}

/** What the ticked changes come to, as the dock at the foot of the list reads it. */
export type BulkGate = {
  count: number;
  /** Ticked changes an approval carries: read, current and able to run. */
  approveIds: string[];
  /** The digest of each preview the review shows, for those it read. */
  digests: { proposalId: string; digest: string }[];
  /** Every ticked change; a decline carries them all. */
  declineIds: string[];
  /** Ticked changes with something to read first. */
  flagged: number;
  /** Ticked changes that cannot run as they stand, or whose record moved; an approval leaves them out. */
  blocked: number;
  /** Some ticked change's preview is still being read. */
  reading: boolean;
};

export function bulkGateOf(
  nodes: readonly PendingProposalNode[],
  previews: DecisionPreviews,
): BulkGate {
  const gate: BulkGate = {
    count: nodes.length,
    approveIds: [],
    digests: [],
    declineIds: nodes.map((node) => node.id),
    flagged: 0,
    blocked: 0,
    reading: false,
  };
  for (const node of nodes) {
    const entry = previews.entry(node.id);
    const preview = entry?.preview ?? null;
    const flag = flagOf(entry);
    if (previews.reading(node.id)) {
      gate.reading = true;
      continue;
    }
    if (preview !== null && !preview.stale && flag !== "danger") {
      gate.approveIds.push(node.id);
      gate.digests.push({ proposalId: node.id, digest: preview.digest });
      if (flag === "warning") {
        gate.flagged++;
      }
    } else if (unreviewedApprovable(node, entry, previews.failed(node.id))) {
      gate.approveIds.push(node.id);
      gate.flagged++;
    } else {
      gate.blocked++;
    }
  }

  return gate;
}
