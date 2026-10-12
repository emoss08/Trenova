import {
  journeyStops,
  laneOf,
} from "@/routes/desk/_components/artifacts/cards/shipment/shipment-card";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQuery } from "@tanstack/react-query";
import { documentKind, workerRoleText } from "../../palette-fields";
import type { PaletteRecord } from "../../palette-model";
import { shipmentStatus } from "../../palette-status";
import { RecordLink } from "./record-link";
import {
  customerPreviewQuery,
  documentPreviewQuery,
  shipmentPreviewQuery,
  workerPreviewQuery,
} from "./preview-queries";

/**
 * How many of the records an answer used are drawn with their detail. Each
 * reads the preview a hover would open, so the first few are warm by the time
 * the person points at them; the rest are listed by name only.
 */
export const SOURCE_DETAILS_SHOWN = 8;

type SourceProps = {
  record: PaletteRecord;
  onOpen: (record: PaletteRecord) => void;
};

function ShipmentSource({ record, onOpen }: SourceProps) {
  const t = useT();
  const { data } = useQuery(shipmentPreviewQuery(record.id));
  const lane = data ? laneOf(journeyStops(data.moves)) : null;
  const sub = lane
    ? [lane.from?.label, lane.to?.label].filter(Boolean).join(" → ")
    : (data?.customer?.name ?? "");
  return (
    <RecordLink
      record={record}
      sub={sub}
      note={data ? shipmentStatus(data.status, t)?.label : undefined}
      onOpen={onOpen}
    />
  );
}

function CustomerSource({ record, onOpen }: SourceProps) {
  const { data } = useQuery(customerPreviewQuery(record.id));
  const place = data ? [data.city, data.state?.abbreviation].filter(Boolean).join(", ") : "";
  return <RecordLink record={record} sub={place} onOpen={onOpen} />;
}

function WorkerSource({ record, onOpen }: SourceProps) {
  const { data } = useQuery(workerPreviewQuery(record.id));
  const role = data ? workerRoleText(data.type ?? undefined, data.driverType ?? undefined) : "";
  return <RecordLink record={record} sub={role} onOpen={onOpen} />;
}

function DocumentSource({ record, onOpen }: SourceProps) {
  const { data } = useQuery(documentPreviewQuery(record.id));
  return (
    <RecordLink
      record={record}
      sub={data?.documentType?.name || documentKind(record)}
      onOpen={onOpen}
    />
  );
}

/** A record the answer was built from, with the line that tells it apart. */
export function SourceRecord({ record, detailed, onOpen }: SourceProps & { detailed: boolean }) {
  if (!detailed) {
    return <RecordLink record={record} onOpen={onOpen} />;
  }
  switch (record.entityType) {
    case "shipment":
      return <ShipmentSource record={record} onOpen={onOpen} />;
    case "customer":
      return <CustomerSource record={record} onOpen={onOpen} />;
    case "worker":
      return <WorkerSource record={record} onOpen={onOpen} />;
    case "document":
      return <DocumentSource record={record} onOpen={onOpen} />;
  }
}
