import { recordPath } from "@/config/record-links";
import { ShipmentStatusBadge } from "@trenova/shared/components/status-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { CardPagedList } from "./card-paged-list";
import { CardRecordRow, CardSection } from "./record-card";
import { cardDay, cardTime, money } from "./record-card-values";
import type { RecordShipment } from "@/lib/graphql/record-shipments";
import { shipmentLane, type RecordShipments, type ShipmentLane } from "./record-shipments";

/**
 * How a card draws its shipments, for what it reads them for: `rows`, a box a
 * load; `table`, a ruled table to scan many loads at once; `board`, a
 * departures board led by when each load picks up; `route`, an order's legs
 * one after another down a line.
 */
export type ShipmentLayout = "rows" | "table" | "board" | "route";

/**
 * A record's shipments as a card lists them, newest first: each load's PRO,
 * lane, status, pickup day and charge, how many there are in all beside the
 * title, and the next page on request. A row opens the shipment's card in the
 * listing card's place when the kind has one here (`onOpen`, from
 * `useListedRecords`), and links to the shipment's page when it does not.
 */
export function CardShipments({
  title,
  shipments,
  onOpen,
  emptyText,
  layout = "rows",
  figureOf,
}: {
  title: string;
  shipments: RecordShipments;
  onOpen: ((shipmentId: string) => void) | undefined;
  /** What the section says when the record has no shipments. */
  emptyText: string;
  layout?: ShipmentLayout;
  /**
   * On a route, what the listing record makes of each load in place of the
   * load's own charge, such as what a settlement pays for it; null leaves the
   * load's charge.
   */
  figureOf?: (shipmentId: string) => string | null;
}) {
  const t = useT();
  return (
    <CardSection
      title={title}
      aside={
        shipments.total !== null && shipments.total > 0 ? (
          <span>{t("{0, plural, one {# shipment} other {# shipments}}", shipments.total)}</span>
        ) : undefined
      }
    >
      <ShipmentList
        shipments={shipments}
        onOpen={onOpen}
        emptyText={emptyText}
        layout={layout}
        figureOf={figureOf}
      />
    </CardSection>
  );
}

function ShipmentList({
  shipments,
  onOpen,
  emptyText,
  layout,
  figureOf,
}: {
  shipments: RecordShipments;
  onOpen: ((shipmentId: string) => void) | undefined;
  emptyText: string;
  layout: ShipmentLayout;
  figureOf: ((shipmentId: string) => string | null) | undefined;
}) {
  const t = useT();
  return (
    <CardPagedList
      list={shipments}
      emptyText={emptyText}
      errorText={t("The shipments could not be read.")}
      nextErrorText={t("The next shipments could not be read.")}
      loadingLabel={t("Reading more shipments…")}
    >
      {(rows) => {
        switch (layout) {
          case "table":
            return <ShipmentTable rows={rows} onOpen={onOpen} />;
          case "board":
            return <ShipmentBoard rows={rows} onOpen={onOpen} />;
          case "route":
            return <ShipmentRoute rows={rows} onOpen={onOpen} figureOf={figureOf} />;
          default:
            return <ShipmentRows rows={rows} onOpen={onOpen} />;
        }
      }}
    </CardPagedList>
  );
}

type ShipmentRowsProps = {
  rows: readonly RecordShipment[];
  onOpen: ((shipmentId: string) => void) | undefined;
};

/** A load's lane, from its first pickup to its last delivery. */
function Lane({ lane }: { lane: ShipmentLane }) {
  const t = useT();
  return (
    <span className="dk-rec-lane">
      {lane.origin ?? t("No pickup")}
      <i aria-hidden> → </i>
      {lane.destination ?? t("No delivery")}
    </span>
  );
}

/** A box a load: its PRO over its lane, its status over its day and charge. */
function ShipmentRows({ rows, onOpen }: ShipmentRowsProps) {
  return (
    <div className="dk-rec-list">
      {rows.map((shipment) => {
        const lane = shipmentLane(shipment);
        return (
          <CardRecordRow
            key={shipment.id}
            to={recordPath("shipment", shipment.id)}
            onOpen={onOpen && (() => onOpen(shipment.id))}
          >
            <span className="dk-rec-main">
              <b>{shipment.proNumber}</b>
              <Lane lane={lane} />
            </span>
            <span className="dk-rec-side">
              <ShipmentStatusBadge status={shipment.status} />
              <span className="dk-rec-sub">
                {[cardDay(lane.pickupAt), money(shipment.totalChargeAmount)]
                  .filter(Boolean)
                  .join(" · ")}
              </span>
            </span>
          </CardRecordRow>
        );
      })}
    </div>
  );
}

/** A ruled table a load to a line, under its column heads, to scan many at once. */
function ShipmentTable({ rows, onOpen }: ShipmentRowsProps) {
  const t = useT();
  return (
    <div className="dk-rec-tbl">
      <div className="dk-rec-tbl-h" aria-hidden>
        <span>{t("PRO")}</span>
        <span>{t("Lane")}</span>
        <span>{t("Pickup")}</span>
        <span>{t("Charge")}</span>
        <span>{t("Status")}</span>
      </div>
      {rows.map((shipment) => {
        const lane = shipmentLane(shipment);
        return (
          <CardRecordRow
            key={shipment.id}
            className="dk-rec-tbl-r"
            to={recordPath("shipment", shipment.id)}
            onOpen={onOpen && (() => onOpen(shipment.id))}
          >
            <b>{shipment.proNumber}</b>
            <Lane lane={lane} />
            <span className="dk-rec-tbl-n">{cardDay(lane.pickupAt) || "—"}</span>
            <span className="dk-rec-tbl-n">{money(shipment.totalChargeAmount)}</span>
            <ShipmentStatusBadge status={shipment.status} />
          </CardRecordRow>
        );
      })}
    </div>
  );
}

/** A departures board: when each load picks up, large, then where it goes and how it stands. */
function ShipmentBoard({ rows, onOpen }: ShipmentRowsProps) {
  const t = useT();
  return (
    <div className="dk-rec-board">
      {rows.map((shipment) => {
        const lane = shipmentLane(shipment);
        return (
          <CardRecordRow
            key={shipment.id}
            className="dk-rec-board-r"
            to={recordPath("shipment", shipment.id)}
            onOpen={onOpen && (() => onOpen(shipment.id))}
          >
            <span className="dk-rec-board-at">
              <b>{lane.pickupAt ? cardTime(lane.pickupAt) : "—"}</b>
              <span>{lane.pickupAt ? cardDay(lane.pickupAt) : t("No pickup")}</span>
            </span>
            <span className="dk-rec-main">
              <b>{shipment.proNumber}</b>
              <Lane lane={lane} />
            </span>
            <ShipmentStatusBadge status={shipment.status} />
          </CardRecordRow>
        );
      })}
    </div>
  );
}

/** An order's legs one after another down a line, each from its pickup to its delivery. */
function ShipmentRoute({
  rows,
  onOpen,
  figureOf,
}: ShipmentRowsProps & { figureOf: ((shipmentId: string) => string | null) | undefined }) {
  const t = useT();
  return (
    <ol className="dk-rec-route">
      {rows.map((shipment, index) => {
        const lane = shipmentLane(shipment);
        return (
          <li key={shipment.id}>
            <span className="dk-rec-route-n" aria-hidden>
              {index + 1}
            </span>
            <CardRecordRow
              className="dk-rec-route-r"
              to={recordPath("shipment", shipment.id)}
              onOpen={onOpen && (() => onOpen(shipment.id))}
            >
              <span className="dk-rec-route-ends">
                <span>
                  <i aria-hidden />
                  {lane.origin ?? t("No pickup")}
                </span>
                <span>
                  <i aria-hidden />
                  {lane.destination ?? t("No delivery")}
                </span>
              </span>
              <span className="dk-rec-route-side">
                <b>{shipment.proNumber}</b>
                <span className="dk-rec-sub">
                  {[
                    cardDay(lane.pickupAt),
                    figureOf?.(shipment.id) ?? money(shipment.totalChargeAmount),
                  ]
                    .filter(Boolean)
                    .join(" · ")}
                </span>
              </span>
              <ShipmentStatusBadge status={shipment.status} />
            </CardRecordRow>
          </li>
        );
      })}
    </ol>
  );
}
