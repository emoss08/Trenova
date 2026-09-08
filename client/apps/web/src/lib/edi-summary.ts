type StatusCount = { status: string; count: number };

type SummaryWindow = {
  deliveryStatusCounts: readonly StatusCount[];
  ackStatusCounts: readonly StatusCount[];
  inboundFileStatusCounts: readonly StatusCount[];
  inboundTransferStatusCounts: readonly StatusCount[];
  overdueAckCount: number;
  attentionItems: readonly unknown[];
};

function total(counts: readonly StatusCount[]): number {
  return counts.reduce((sum, entry) => sum + entry.count, 0);
}

/**
 * Whether anything at all moved through EDI in the summary's window: a
 * delivery in any state, an acknowledgment, an inbound file or transfer, an
 * overdue ack, or a failure on the list. A window with traffic and no
 * problems is a healthy dashboard of zeros; a window with no traffic has
 * nothing to count, and every zero on it would say the same thing.
 */
export function ediWindowHasTraffic(summary: SummaryWindow): boolean {
  return (
    total(summary.deliveryStatusCounts) > 0 ||
    total(summary.ackStatusCounts) > 0 ||
    total(summary.inboundFileStatusCounts) > 0 ||
    total(summary.inboundTransferStatusCounts) > 0 ||
    summary.overdueAckCount > 0 ||
    summary.attentionItems.length > 0
  );
}
