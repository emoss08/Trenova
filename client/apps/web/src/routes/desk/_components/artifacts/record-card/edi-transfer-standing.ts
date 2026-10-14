import type { TransferDirection } from "@/lib/edi/transfer-sides";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { EDITransfer } from "@trenova/shared/types/edi";
import type { CardStatusMark, CardTone } from "./record-card";

/** Where the tender stands, as the pill at the card's head says it. */
export type TransferStanding = {
  label: string;
  tone: CardTone;
  mark: CardStatusMark;
};

const SUPERSEDED = "Superseded by";

/** Whether a canceled tender was replaced by a change its partner sent. */
export function superseded(transfer: EDITransfer): boolean {
  return transfer.status === "Canceled" && (transfer.failureReason ?? "").startsWith(SUPERSEDED);
}

/** Where the tender stands, in the words of the side reading it. */
export function transferStanding(
  transfer: EDITransfer,
  direction: TransferDirection,
  t: TranslateFn,
): TransferStanding {
  switch (transfer.status) {
    case "Submitted":
      return { label: t("Submitted"), tone: "progress", mark: "dot" };
    case "MappingRequired":
      return { label: t("Needs mapping"), tone: "warn", mark: "pause" };
    case "PendingApproval":
      return {
        label: direction === "inbound" ? t("Awaiting your decision") : t("Awaiting their decision"),
        tone: "progress",
        mark: "dot",
      };
    case "Processing":
      return { label: t("Building the shipment"), tone: "progress", mark: "dot" };
    case "Approved":
      return { label: t("Accepted"), tone: "success", mark: "check" };
    case "Rejected":
      return { label: t("Declined"), tone: "muted", mark: "dot" };
    case "Expired":
      return { label: t("Expired"), tone: "muted", mark: "dot" };
    case "Canceled":
      return {
        label: superseded(transfer) ? t("Replaced by a change") : t("Canceled"),
        tone: "muted",
        mark: "dot",
      };
    case "Failed":
      return { label: t("Failed"), tone: "danger", mark: "x" };
  }
}
