import { recordPath } from "@/config/record-links";
import type { DeskTrailer as DeskTrailerRecord } from "@/lib/graphql/trailer-card";
import { queries } from "@/lib/queries";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import { useCardAccess } from "../../record-card/card-access";
import { CardDocuments } from "../../record-card/card-documents";
import { CardShipments } from "../../record-card/card-shipments";
import { CardWhere, mapPoint } from "../../record-card/card-where";
import { equipmentName, equipmentStanding } from "../../record-card/equipment";
import {
  CardAvatar,
  CardBar,
  CardBarHint,
  CardFacts,
  CardHead,
  CardIcon,
  CardOpenLink,
  CardRecordRow,
  CardSection,
  CardStatus,
  CardStepper,
  CardUnreadable,
  needsLook,
  RecordCard,
} from "../../record-card/record-card";
import { neighborsIn, personName } from "../../record-card/record-card-values";
import { useRecordShipments } from "../../record-card/record-shipments";
import { useListedRecords, type RecordArtifactProps } from "../registry";
import { calendarMarks, maxLoadText, trailerChecks, trailerFacts } from "./trailer-card";
import { DataPlate, ServiceCalendar } from "./trailer-parts";
import { TRAILER_SECTIONS } from "./trailer-sections";
import { DeskTrailerSkeleton } from "./trailer-skeleton";

/** The shipments the card reads at a time; the next page is read on request. */
const SHIPMENT_ROWS = 5;

/**
 * What the card is given: the trailer, and when it was opened from a table or
 * from a card that lists it, the rows in the order shown, how to step to
 * another, and the way back.
 */
export type TrailerCardProps = {
  recordId: string;
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the trailer was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];

/**
 * One trailer as a dispatcher or a fleet manager reads it, opened from a
 * table's row or looked up by the agent: whether it can be loaded and where
 * its last move left it, its annual inspection, registration and lease, who
 * owns it, the loads it carried, what it is and its documents. It reads;
 * changing its status, registration and inspection stays on the trailer's
 * page, one click away.
 *
 * The card steps through the rows it was opened from in the order shown.
 */
export function DeskTrailer({
  recordId,
  order = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: TrailerCardProps) {
  const t = useT();
  const query = useQuery({
    ...queries.trailer.card(recordId),
    retry: false,
    placeholderData: keepPreviousData,
  });
  const trailer = query.data;

  if (!trailer) {
    return query.isError ? (
      <CardUnreadable
        kind={t("Trailer")}
        title={t("This trailer could not be read")}
        error={query.error}
        onBack={onBack}
        backLabel={backLabel}
      />
    ) : (
      <DeskTrailerSkeleton stepper={onSelect !== undefined && order.length > 1} />
    );
  }

  return (
    <TrailerCard
      key={trailer.id}
      trailer={trailer}
      order={order}
      onSelect={onSelect}
      onBack={onBack}
      backLabel={backLabel}
    />
  );
}

/**
 * The trailer as an artifact of its own, when the agent looked it up: the same
 * card, with nothing to step through and nowhere to go back to.
 */
export function DeskTrailerArtifact({ recordId }: RecordArtifactProps) {
  return <DeskTrailer recordId={recordId} />;
}

function TrailerCard({
  trailer,
  order,
  onSelect,
  onBack,
  backLabel,
}: Required<Pick<TrailerCardProps, "order">> &
  Omit<TrailerCardProps, "order" | "recordId"> & { trailer: DeskTrailerRecord }) {
  const t = useT();
  const access = useCardAccess(TRAILER_SECTIONS);
  const [now] = useState(() => Math.floor(Date.now() / 1000));
  const id = trailer.id;

  const shipments = useRecordShipments({
    scope: { kind: "trailer", id },
    pageSize: SHIPMENT_ROWS,
    enabled: access.shipments,
  });

  const checks = trailerChecks(trailer, now, t);
  const openChecks = checks.filter((check) => needsLook(check.state)).length;
  const standing = equipmentStanding(trailer.status, t);
  const neighbors = neighborsIn(order, id);
  const load = maxLoadText(trailer.maxLoadWeight, t);
  const owner = trailer.ownershipType === "OwnerOperator" ? (trailer.ownerWorker ?? null) : null;
  const lastKnown = trailer.lastKnownLocation ?? null;

  const backToTrailer = t("Back to {0}", trailer.code);
  const listedShipments = useListedRecords({
    entity: "shipment",
    order: shipments.ids,
    backLabel: backToTrailer,
  });
  const ownerIds = useMemo(() => (owner ? [owner.id] : NO_ROWS), [owner]);
  const listedOwner = useListedRecords({
    entity: "worker",
    order: ownerIds,
    backLabel: backToTrailer,
  });
  const placeIds = useMemo(() => (lastKnown ? [lastKnown.id] : NO_ROWS), [lastKnown]);
  const listedPlace = useListedRecords({
    entity: "location",
    order: placeIds,
    backLabel: backToTrailer,
  });

  // A listed record opened from the card takes the card's place, and its way
  // back returns here.
  const opened = listedShipments.opened ?? listedOwner.opened ?? listedPlace.opened;
  if (opened) {
    return opened;
  }

  return (
    <RecordCard className={cn("dk-tl", `dk-tl-${standing.tone}`)}>
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={[
          trailer.code,
          ...(trailer.equipmentType ? [trailer.equipmentType.code] : []),
          ...(trailer.fleetCode ? [t("Fleet {0}", trailer.fleetCode.code)] : []),
        ]}
        stepper={
          onSelect &&
          order.length > 1 && (
            <CardStepper
              neighbors={neighbors}
              onStep={onSelect}
              previousLabel={t("Previous trailer")}
              nextLabel={t("Next trailer")}
            />
          )
        }
        title={equipmentName(trailer)}
        subtitle={trailer.vin.trim() ? t("VIN {0}", trailer.vin.trim()) : t("No VIN on file")}
        figure={
          <>
            <b className={cn(!load && "dk-tl-wait")}>{load ?? "—"}</b>
            <span>{load ? t("maximum load") : t("No maximum load on file")}</span>
          </>
        }
        meta={
          <>
            <CardStatus key={standing.label} tone={standing.tone} mark={standing.mark}>
              {standing.label}
            </CardStatus>
            <CardFacts>
              {[
                trailer.equipmentType?.description ?? "",
                trailer.ownershipType === "OwnerOperator"
                  ? t("Owner-operator")
                  : trailer.ownershipType === "Leased"
                    ? t("Leased")
                    : "",
                trailer.licensePlateNumber.trim()
                  ? t("Plate {0}", trailer.licensePlateNumber.trim())
                  : "",
              ]}
            </CardFacts>
          </>
        }
      />

      <CardSection
        title={t("Where it is")}
        aside={
          lastKnown ? (
            <CardOpenLink
              to={recordPath("location", lastKnown.id)}
              onOpen={listedPlace.open && (() => listedPlace.open?.(lastKnown.id))}
            >
              {t("Open location")}
            </CardOpenLink>
          ) : undefined
        }
      >
        <CardWhere
          point={mapPoint(lastKnown?.latitude, lastKnown?.longitude)}
          place={lastKnown?.name ?? null}
          placeMissing={t("No known location yet")}
          line={
            lastKnown
              ? t("Where its last move left it")
              : t("It has not been on a move or located by hand")
          }
          lineOff={!lastKnown}
          facts={
            lastKnown?.latitude !== null &&
            lastKnown?.latitude !== undefined &&
            lastKnown.longitude !== null &&
            lastKnown.longitude !== undefined
              ? [t("{0}, {1}", lastKnown.latitude.toFixed(4), lastKnown.longitude.toFixed(4))]
              : lastKnown
                ? [t("Not geocoded")]
                : []
          }
          mapLabel={lastKnown ? t("Map of {0}", lastKnown.name) : t("Map")}
          noMapLabel={lastKnown ? t("This location has not been geocoded") : t("No known location")}
        />
      </CardSection>

      <CardSection
        title={t("Service calendar")}
        aside={
          openChecks > 0 ? (
            <span className="dk-warn">
              {t("{0, plural, one {# needs a look} other {# need a look}}", openChecks)}
            </span>
          ) : (
            <span className="dk-ok">{t("Nothing due")}</span>
          )
        }
      >
        <ServiceCalendar marks={calendarMarks(trailer, now, t)} checks={checks} now={now} />
      </CardSection>

      {owner && (
        <CardSection title={t("Owner")}>
          <div className="dk-rec-list">
            <CardRecordRow
              to={recordPath("worker", owner.id)}
              onOpen={listedOwner.open && (() => listedOwner.open?.(owner.id))}
            >
              <span className="dk-tl-owner">
                <CardAvatar name={personName(owner) || t("Unnamed worker")} />
                <span className="dk-rec-main">
                  <b>{personName(owner) || t("Unnamed worker")}</b>
                  <span className="dk-rec-sub">{t("Owner-operator")}</span>
                </span>
              </span>
              <CardIcon name="forward" size={12} />
            </CardRecordRow>
          </div>
        </CardSection>
      )}

      {access.shipments && (
        <CardShipments
          title={t("Recent shipments")}
          shipments={shipments}
          onOpen={listedShipments.open}
          emptyText={t("This trailer has not carried a load yet.")}
        />
      )}

      <CardSection title={t("Data plate")}>
        <DataPlate facts={trailerFacts(trailer, t)} />
      </CardSection>

      <CardDocuments resourceType="trailer" resourceId={id} manageTo={recordPath("trailer", id)} />

      <CardBar>
        <CardBarHint>
          {t("Change status, registration and inspection on the trailer's page")}
        </CardBarHint>
        <Link className="dk-ax-btn dk-ink" to={recordPath("trailer", id)}>
          <CardIcon name="ext" size={13} />
          {t("Open trailer")}
        </Link>
      </CardBar>
    </RecordCard>
  );
}
