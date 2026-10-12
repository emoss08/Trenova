import { recordPath } from "@/config/record-links";
import { describeApiError } from "@/lib/api-error-message";
import { ShipmentStatusBadge } from "@trenova/shared/components/status-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  CardMore,
  CardRecordRow,
  CardSection,
  CardSectionError,
  CardSectionWait,
} from "./record-card";
import { cardDay, money } from "./record-card-values";
import { shipmentLane, type RecordShipments } from "./record-shipments";

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
}: {
  title: string;
  shipments: RecordShipments;
  onOpen: ((shipmentId: string) => void) | undefined;
  /** What the section says when the record has no shipments. */
  emptyText: string;
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
      <ShipmentList shipments={shipments} onOpen={onOpen} emptyText={emptyText} />
    </CardSection>
  );
}

function ShipmentList({
  shipments,
  onOpen,
  emptyText,
}: {
  shipments: RecordShipments;
  onOpen: ((shipmentId: string) => void) | undefined;
  emptyText: string;
}) {
  const t = useT();
  const { rows, error, pending } = shipments;
  if (error && !rows) {
    return (
      <CardSectionError>
        {describeApiError(error, t("The shipments could not be read."))}
      </CardSectionError>
    );
  }
  if (pending || !rows) {
    return <CardSectionWait rows={3} />;
  }
  if (rows.length === 0) {
    return <p className="dk-rec-note">{emptyText}</p>;
  }
  return (
    <>
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
                <span className="dk-rec-lane">
                  {lane.origin ?? t("No pickup")}
                  <i aria-hidden> → </i>
                  {lane.destination ?? t("No delivery")}
                </span>
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
      {shipments.hasMore && (
        <CardMore
          shown={
            shipments.total !== null
              ? t("{0} of {1} shown", rows.length, shipments.total)
              : t("{0} shown", rows.length)
          }
          onMore={shipments.loadMore}
          loading={shipments.loadingMore}
          label={t("Show more")}
          loadingLabel={t("Reading more shipments…")}
        />
      )}
      {shipments.error !== null && shipments.error !== undefined && (
        <CardSectionError>
          {describeApiError(shipments.error, t("The next shipments could not be read."))}
        </CardSectionError>
      )}
    </>
  );
}
