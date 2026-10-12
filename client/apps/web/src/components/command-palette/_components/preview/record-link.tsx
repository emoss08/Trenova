import { getNameInitials } from "@trenova/shared/lib/utils";
import { PALETTE_ENTITY_ICONS } from "../../palette-entities";
import type { PaletteRecord } from "../../palette-model";
import { RelatedRow } from "./preview-frame";

function recordSub(record: PaletteRecord): string {
  const meta = record.metadata;
  switch (record.entityType) {
    case "shipment":
      return [meta.origin, meta.destination].filter(Boolean).join(" → ") || meta.customerName || "";
    case "customer":
      return meta.city ?? record.subtitle ?? "";
    case "worker":
      return meta.city ?? record.subtitle ?? "";
    case "document":
      return meta.attachedLabel ?? "";
  }
}

/** A record named in a preview; the line under it is the record's own unless one is given. */
export function RecordLink({
  record,
  sub,
  note,
  onOpen,
}: {
  record: PaletteRecord;
  sub?: string;
  note?: React.ReactNode;
  onOpen: (record: PaletteRecord) => void;
}) {
  return (
    <RelatedRow
      icon={PALETTE_ENTITY_ICONS[record.entityType]}
      initials={record.entityType === "worker" ? getNameInitials(record.title) : undefined}
      title={record.title}
      mono={record.entityType === "shipment" || record.entityType === "document"}
      sub={sub || recordSub(record) || undefined}
      note={note}
      onOpen={() => onOpen(record)}
    />
  );
}
