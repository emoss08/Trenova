import { shipmentStatusText } from "@trenova/shared/components/status-badge";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { shipmentStatusSchema } from "@trenova/shared/types/shipment";
import type { PaletteRecord } from "./palette-model";
import type { StatusState } from "./palette-parts";

export interface RecordStatus {
  label: string;
  state: StatusState;
}

/** A shipment's status as the palette draws it: the words the app uses, a dot only where it needs someone. */
export function shipmentStatus(status: string | undefined, t: TranslateFn): RecordStatus | null {
  const parsed = shipmentStatusSchema.safeParse(status);
  if (!parsed.success) {
    return null;
  }
  const state: StatusState =
    parsed.data === "Delayed"
      ? "warn"
      : parsed.data === "Completed" || parsed.data === "ReadyToInvoice"
        ? "ok"
        : "neutral";
  return { label: shipmentStatusText(parsed.data, t), state };
}

/** A customer is only labelled when it is not active; an active one is the ordinary case. */
export function customerStatus(record: PaletteRecord, t: TranslateFn): RecordStatus | null {
  const status = record.metadata.status;
  if (!status || status === "Active" || status === "Unknown") {
    return null;
  }
  return { label: status === "Inactive" ? t("Inactive") : status, state: "warn" };
}

/**
 * Where a worker stands right now: on a load, free to dispatch, or out of
 * service. Each reading comes from the search hit, so a row never waits on a
 * second request to say it.
 */
export function workerStatus(record: PaletteRecord, t: TranslateFn): RecordStatus | null {
  const meta = record.metadata;
  if (meta.status && meta.status !== "Active" && meta.status !== "Unknown") {
    return { label: meta.status === "Inactive" ? t("Inactive") : meta.status, state: "neutral" };
  }
  if (meta.currentLoadStatus === "InTransit" || meta.currentLoadStatus === "PartiallyCompleted") {
    return { label: t("Driving"), state: "neutral" };
  }
  if (meta.currentLoadStatus === "Delayed") {
    return { label: t("Driving"), state: "warn" };
  }
  if (meta.currentLoadId) {
    return { label: t("Assigned"), state: "neutral" };
  }
  if (meta.availableForDispatch === "true") {
    return { label: t("Available"), state: "ok" };
  }
  if (meta.availableForDispatch === "false") {
    return { label: t("Unavailable"), state: "neutral" };
  }
  return null;
}

export function recordStatus(record: PaletteRecord, t: TranslateFn): RecordStatus | null {
  switch (record.entityType) {
    case "shipment":
      return shipmentStatus(record.metadata.status, t);
    case "customer":
      return customerStatus(record, t);
    case "worker":
      return workerStatus(record, t);
    case "document":
      return null;
  }
}
