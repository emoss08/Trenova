import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type {
  EvidenceKind,
  EvidenceSource,
  NoticeDeliveryStatus,
  NoticeKind,
} from "@trenova/shared/types/detention";

/** What a detention notice was, as the desk and the Desk's card name it. */
export function noticeKindLabel(kind: NoticeKind, t: TranslateFn): string {
  switch (kind) {
    case "Warning":
      return t("Warning notice");
    case "Started":
      return t("Started notice");
    case "Update":
      return t("Update notice");
    case "Final":
      return t("Final notice");
  }
}

/** Where a detention notice's delivery stands. */
export function deliveryStatusLabel(status: NoticeDeliveryStatus, t: TranslateFn): string {
  switch (status) {
    case "Queued":
      return t("Queued");
    case "Sent":
      return t("Sent");
    case "Delivered":
      return t("Delivered");
    case "Opened":
      return t("Opened");
    case "Bounced":
      return t("Bounced");
    case "Failed":
      return t("Failed");
  }
}

/** What a piece of detention evidence records. */
export function evidenceKindLabel(kind: EvidenceKind, t: TranslateFn): string {
  switch (kind) {
    case "Arrival":
      return t("Arrival");
    case "Departure":
      return t("Departure");
    case "Appointment":
      return t("Appointment");
    case "Notice":
      return t("Notice");
    case "Document":
      return t("Document");
    case "StatusChange":
      return t("Status change");
    case "Recalculated":
      return t("Recalculated");
    case "Waiver":
      return t("Waiver");
    case "Dispute":
      return t("Dispute");
  }
}

/** Where a piece of detention evidence came from. */
export function evidenceSourceLabel(source: EvidenceSource, t: TranslateFn): string {
  switch (source) {
    case "Telematics":
      return t("Telematics");
    case "Geofence":
      return t("Geofence");
    case "DriverApp":
      return t("Worker app");
    case "EDI":
      return t("EDI");
    case "Document":
      return t("Document");
    case "Manual":
      return t("Entered by hand");
    case "System":
      return t("System");
  }
}
