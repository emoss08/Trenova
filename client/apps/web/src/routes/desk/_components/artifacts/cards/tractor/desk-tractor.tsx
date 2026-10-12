import { recordPath } from "@/config/record-links";
import { describeApiError } from "@/lib/api-error-message";
import type {
  DeskTractor as DeskTractorRecord,
  DeskTractorInspection,
  DeskTractorPosition,
} from "@/lib/graphql/tractor-card";
import { queries } from "@/lib/queries";
import { tractorInspectionsQuery } from "@/lib/queries/tractor";
import { keepPreviousData, useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, formatPercent, metersToMiles } from "@trenova/shared/lib/utils";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import { useCardAccess } from "../../record-card/card-access";
import { CardDocuments } from "../../record-card/card-documents";
import { CardShipments } from "../../record-card/card-shipments";
import {
  CardBar,
  CardBarHint,
  CardFactGrid,
  CardFacts,
  CardHead,
  CardIcon,
  CardMore,
  CardRecordRow,
  CardSection,
  CardSectionError,
  CardSectionWait,
  CardStatus,
  CardStepper,
  CardUnreadable,
  RecordCard,
} from "../../record-card/record-card";
import { cardDate, neighborsIn } from "../../record-card/record-card-values";
import { useRecordShipments } from "../../record-card/record-shipments";
import { useListedRecords, type RecordArtifactProps } from "../registry";
import {
  inspectionKind,
  inspectionReading,
  positionReading,
  tractorChecks,
  tractorFacts,
  tractorPeople,
} from "./tractor-card";
import { CardDue } from "../../record-card/card-due";
import { CardWhere, mapPoint } from "../../record-card/card-where";
import { equipmentName, equipmentStanding } from "../../record-card/equipment";
import { TRACTOR_SECTIONS } from "./tractor-sections";
import { DeskTractorSkeleton } from "./tractor-skeleton";

/** The shipments the card reads at a time; the next page is read on request. */
const SHIPMENT_ROWS = 5;
/** The inspections the card reads at a time. */
const INSPECTION_ROWS = 5;

/**
 * What the card is given: the tractor, and when it was opened from a table or
 * from a card that lists it, the rows in the order shown, how to step to
 * another, and the way back.
 */
export type TractorCardProps = {
  recordId: string;
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the tractor was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];
const NO_INSPECTIONS: readonly DeskTractorInspection[] = [];

/**
 * One tractor as a dispatcher or a fleet manager reads it, opened from a
 * table's row or looked up by the agent: whether it can run and where it is
 * now, what falls due on it, who drives and owns it, the loads it pulled, its
 * drivers' inspections and what it is. It reads; changing its status, drivers
 * and registration stays on the tractor's page, one click away.
 *
 * The card steps through the rows it was opened from in the order shown.
 */
export function DeskTractor({
  recordId,
  order = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: TractorCardProps) {
  const t = useT();
  const query = useQuery({
    ...queries.tractor.card(recordId),
    retry: false,
    placeholderData: keepPreviousData,
  });
  const tractor = query.data;

  if (!tractor) {
    return query.isError ? (
      <CardUnreadable
        kind={t("Tractor")}
        title={t("This tractor could not be read")}
        error={query.error}
        onBack={onBack}
        backLabel={backLabel}
      />
    ) : (
      <DeskTractorSkeleton stepper={onSelect !== undefined && order.length > 1} />
    );
  }

  return (
    <TractorCard
      key={tractor.id}
      tractor={tractor}
      order={order}
      onSelect={onSelect}
      onBack={onBack}
      backLabel={backLabel}
    />
  );
}

/**
 * The tractor as an artifact of its own, when the agent looked it up: the same
 * card, with nothing to step through and nowhere to go back to.
 */
export function DeskTractorArtifact({ recordId }: RecordArtifactProps) {
  return <DeskTractor recordId={recordId} />;
}

function TractorCard({
  tractor,
  order,
  onSelect,
  onBack,
  backLabel,
}: Required<Pick<TractorCardProps, "order">> &
  Omit<TractorCardProps, "order" | "recordId"> & { tractor: DeskTractorRecord }) {
  const t = useT();
  const access = useCardAccess(TRACTOR_SECTIONS);
  const [now] = useState(() => Math.floor(Date.now() / 1000));
  const id = tractor.id;

  const position = useQuery({ ...queries.tractor.position(id), retry: false });
  const inspections = useInfiniteQuery({
    ...tractorInspectionsQuery(id, INSPECTION_ROWS),
    retry: false,
  });
  const shipments = useRecordShipments({
    scope: { kind: "tractor", id },
    pageSize: SHIPMENT_ROWS,
    enabled: access.shipments,
  });

  const inspectionRows = useMemo(
    () => inspections.data?.pages.flatMap((page) => page.rows) ?? NO_INSPECTIONS,
    [inspections.data],
  );
  // Undefined while the inspections are read, null once they are known to be none.
  const latest = inspections.data ? (inspectionRows[0] ?? null) : undefined;
  const checks = tractorChecks(tractor, latest, now, t);
  const people = tractorPeople(tractor, t);
  const peopleIds = useMemo(() => people.map((person) => person.id), [people]);
  const standing = equipmentStanding(tractor.status, t);
  const neighbors = neighborsIn(order, id);
  const name = equipmentName(tractor);

  const backToTractor = t("Back to {0}", tractor.code);
  const listedShipments = useListedRecords({
    entity: "shipment",
    order: shipments.ids,
    backLabel: backToTractor,
  });
  const listedPeople = useListedRecords({
    entity: "worker",
    order: peopleIds,
    backLabel: backToTractor,
  });
  const lastKnown = tractor.lastKnownLocation;
  const lastKnownIds = useMemo(() => (lastKnown ? [lastKnown.id] : NO_ROWS), [lastKnown]);
  const listedPlace = useListedRecords({
    entity: "location",
    order: lastKnownIds,
    backLabel: backToTractor,
  });

  // A listed record opened from the card takes the card's place, and its way
  // back returns here.
  const opened = listedShipments.opened ?? listedPeople.opened ?? listedPlace.opened;
  if (opened) {
    return opened;
  }

  return (
    <RecordCard className={cn("dk-tr", `dk-tr-${standing.tone}`)}>
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={[
          tractor.code,
          ...(tractor.equipmentType ? [tractor.equipmentType.code] : []),
          ...(tractor.fleetCode ? [t("Fleet {0}", tractor.fleetCode.code)] : []),
        ]}
        stepper={
          onSelect &&
          order.length > 1 && (
            <CardStepper
              neighbors={neighbors}
              onStep={onSelect}
              previousLabel={t("Previous tractor")}
              nextLabel={t("Next tractor")}
            />
          )
        }
        title={name}
        subtitle={tractor.vin.trim() ? t("VIN {0}", tractor.vin.trim()) : t("No VIN on file")}
        figure={<OdometerFigure position={position.data} pending={position.isPending} />}
        meta={
          <>
            <CardStatus key={standing.label} tone={standing.tone} mark={standing.mark}>
              {standing.label}
            </CardStatus>
            <CardFacts>
              {[
                people[0] ? t("Driven by {0}", people[0].name) : "",
                tractor.ownershipType === "OwnerOperator"
                  ? t("Owner-operator")
                  : tractor.ownershipType === "Leased"
                    ? t("Leased")
                    : "",
                tractor.equipmentType?.description ?? "",
              ]}
            </CardFacts>
          </>
        }
      />

      <CardSection
        title={t("Where it is")}
        aside={
          position.data ? (
            <span className={cn(positionReading(position.data, now, t).live && "dk-ok")}>
              {t("Reported {0}", formatSecondsAgo(now - position.data.recordedAt))}
            </span>
          ) : undefined
        }
      >
        <Whereabouts
          position={position.data}
          pending={position.isPending}
          error={position.error}
          now={now}
          lastKnown={lastKnown}
          onOpenPlace={listedPlace.open}
        />
      </CardSection>

      <CardDue
        title={t("Due")}
        checks={checks}
        waiting={
          latest === undefined && inspections.isPending
            ? t("Reading the last inspection…")
            : undefined
        }
      />

      <CardSection title={t("Drivers")}>
        {people.length > 0 ? (
          <div className="dk-rec-list">
            {people.map((person) => (
              <CardRecordRow
                key={person.key}
                to={recordPath("worker", person.id)}
                onOpen={listedPeople.open && (() => listedPeople.open?.(person.id))}
              >
                <span className="dk-rec-main">
                  <b>{person.name}</b>
                  <span className="dk-rec-sub">{person.role}</span>
                </span>
                <CardIcon name="forward" size={12} />
              </CardRecordRow>
            ))}
          </div>
        ) : (
          <p className="dk-rec-note">{t("No driver is assigned to this tractor.")}</p>
        )}
      </CardSection>

      {access.shipments && (
        <CardShipments
          title={t("Recent shipments")}
          shipments={shipments}
          onOpen={listedShipments.open}
          emptyText={t("This tractor has not pulled a load yet.")}
        />
      )}

      <CardSection title={t("Inspections")}>
        <Inspections
          rows={inspections.data ? inspectionRows : undefined}
          pending={inspections.isPending}
          error={inspections.error}
          hasMore={inspections.hasNextPage}
          loadingMore={inspections.isFetchingNextPage}
          onMore={() => void inspections.fetchNextPage()}
        />
      </CardSection>

      <CardSection title={t("Details")}>
        <CardFactGrid facts={tractorFacts(tractor, t)} />
      </CardSection>

      <CardDocuments resourceType="tractor" resourceId={id} manageTo={recordPath("tractor", id)} />

      <CardBar>
        <CardBarHint>
          {t("Change status, drivers and registration on the tractor's page")}
        </CardBarHint>
        <Link className="dk-ax-btn dk-ink" to={recordPath("tractor", id)}>
          <CardIcon name="ext" size={13} />
          {t("Open tractor")}
        </Link>
      </CardBar>
    </RecordCard>
  );
}

/** The miles on the tractor's clock at its last report, at the head of the card. */
function OdometerFigure({
  position,
  pending,
}: {
  position: DeskTractorPosition | null | undefined;
  pending: boolean;
}) {
  const t = useT();
  const meters = position?.odometerMeters;
  if (pending || meters === null || meters === undefined) {
    return (
      <>
        <b className="dk-tr-wait">—</b>
        <span>{pending ? t("Odometer") : t("No odometer reading")}</span>
      </>
    );
  }
  return (
    <>
      <b key={meters}>{t("{0} mi", formatNumber(Math.round(metersToMiles(meters))))}</b>
      <span>{t("on the odometer")}</span>
    </>
  );
}

/**
 * Where the tractor is: its last GPS report on a small still map, how it was
 * moving and when it said so, and the location its last move left it at in
 * Trenova's own records. With no GPS the map shows that location, muted.
 */
function Whereabouts({
  position,
  pending,
  error,
  now,
  lastKnown,
  onOpenPlace,
}: {
  position: DeskTractorPosition | null | undefined;
  pending: boolean;
  error: unknown;
  now: number;
  lastKnown: DeskTractorRecord["lastKnownLocation"];
  /** Opens the last known location in the card's place; absent, it links to its page. */
  onOpenPlace: ((locationId: string) => void) | undefined;
}) {
  const t = useT();
  const reading = position ? positionReading(position, now, t) : null;
  const place = reading ? reading.place : (lastKnown?.name ?? null);
  const point = position
    ? mapPoint(position.latitude, position.longitude)
    : mapPoint(lastKnown?.latitude, lastKnown?.longitude);
  return (
    <CardWhere
      point={point}
      live={reading?.live ?? false}
      pending={pending}
      place={place}
      placeMissing={
        position ? t("Somewhere without an address") : t("No GPS report from this tractor")
      }
      line={
        reading
          ? reading.motion
          : lastKnown
            ? t("Placed by its last move, not by GPS")
            : t("No known location yet")
      }
      lineOff={!reading}
      facts={
        position
          ? [
              position.fuelPercent !== null && position.fuelPercent !== undefined
                ? t("Fuel {0}", formatPercent(position.fuelPercent, 0))
                : "",
              t("{0}, {1}", position.latitude.toFixed(4), position.longitude.toFixed(4)),
            ]
          : []
      }
      mapLabel={place ? t("Map of {0}", place) : t("Map of the tractor's last reported position")}
      noMapLabel={t("No position reported")}
      error={error}
      errorText={t("The tractor's position could not be read.")}
    >
      {lastKnown && position && (
        <div className="dk-rec-list dk-rec-left">
          <CardRecordRow
            to={recordPath("location", lastKnown.id)}
            onOpen={onOpenPlace && (() => onOpenPlace(lastKnown.id))}
          >
            <span className="dk-rec-main">
              <span className="dk-rec-sub">{t("Last move left it at")}</span>
              <b>{lastKnown.name}</b>
            </span>
            <CardIcon name="forward" size={12} />
          </CardRecordRow>
        </div>
      )}
    </CardWhere>
  );
}

/** The drivers' inspections of the tractor, newest first, a page at a time. */
function Inspections({
  rows,
  pending,
  error,
  hasMore,
  loadingMore,
  onMore,
}: {
  rows: readonly DeskTractorInspection[] | undefined;
  pending: boolean;
  error: unknown;
  hasMore: boolean;
  loadingMore: boolean;
  onMore: () => void;
}) {
  const t = useT();
  if (error && !rows) {
    return (
      <CardSectionError>
        {describeApiError(error, t("The inspections could not be read."))}
      </CardSectionError>
    );
  }
  if (pending || !rows) {
    return <CardSectionWait rows={3} />;
  }
  if (rows.length === 0) {
    return (
      <p className="dk-rec-note">{t("No driver has inspected this tractor through telematics.")}</p>
    );
  }
  return (
    <>
      <ol className="dk-rec-list dk-tr-insp">
        {rows.map((inspection) => {
          const reading = inspectionReading(inspection, t);
          const location = inspection.location?.trim();
          return (
            <li key={inspection.id}>
              <span className="dk-rec-main">
                <b>
                  {t(
                    "{0} · {1}",
                    inspectionKind(inspection.inspectionType, t),
                    cardDate(inspection.startedAt),
                  )}
                </b>
                <span className="dk-rec-sub">
                  {[
                    inspection.workerName?.trim() || t("Driver not matched"),
                    location ?? "",
                    inspection.odometerMeters
                      ? t(
                          "{0} mi",
                          formatNumber(Math.round(metersToMiles(inspection.odometerMeters))),
                        )
                      : "",
                  ]
                    .filter(Boolean)
                    .join(" · ")}
                </span>
              </span>
              <span className={cn("dk-tr-verdict", `dk-${reading.state}`)}>{reading.text}</span>
            </li>
          );
        })}
      </ol>
      {hasMore && (
        <CardMore
          shown={t("{0} shown", rows.length)}
          onMore={onMore}
          loading={loadingMore}
          label={t("Show older")}
          loadingLabel={t("Reading older inspections…")}
        />
      )}
      {error !== null && error !== undefined && (
        <CardSectionError>
          {describeApiError(error, t("The older inspections could not be read."))}
        </CardSectionError>
      )}
    </>
  );
}
