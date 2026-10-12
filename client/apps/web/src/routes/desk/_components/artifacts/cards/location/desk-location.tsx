import { recordPath } from "@/config/record-links";
import { describeApiError } from "@/lib/api-error-message";
import {
  choiceLabel,
  detentionPolicyStatusChoices,
  facilityTypeChoices,
  locationCategoryTypeChoices,
  serviceFailureStatusChoices,
} from "@/lib/choices";
import type {
  DeskLocation as DeskLocationRecord,
  LocationDwell,
  LocationFailure,
  LocationPolicies,
} from "@/lib/graphql/location-card";
import { queries } from "@/lib/queries";
import { locationFailuresQuery } from "@/lib/queries/location";
import { keepPreviousData, useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import { useCardAccess } from "../../record-card/card-access";
import { CardShipments } from "../../record-card/card-shipments";
import { CardWhere, mapPoint } from "../../record-card/card-where";
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
  CardStats,
  CardStatus,
  CardStepper,
  CardUnreadable,
  RecordCard,
} from "../../record-card/record-card";
import { neighborsIn } from "../../record-card/record-card-values";
import { useRecordShipments } from "../../record-card/record-shipments";
import { MAP_SIZE } from "../../record-card/still-map";
import { useListedRecords, type RecordArtifactProps } from "../registry";
import {
  cityLine,
  DWELL_WINDOW_DAYS,
  dwellMoneyLine,
  dwellStats,
  dwellWindow,
  facilityTerms,
  failureLine,
  geofenceLine,
  geofenceOutline,
  locationFacts,
  locationStanding,
  locationZoom,
  policyFreeTime,
  policyScope,
  streetLine,
  waitText,
} from "./location-card";
import { LOCATION_SECTIONS } from "./location-sections";
import { DeskLocationSkeleton, DwellSkeleton } from "./location-skeleton";

/** The shipments the card reads at a time; the next page is read on request. */
const SHIPMENT_ROWS = 5;
/** The service failures the card reads at a time. */
const FAILURE_ROWS = 5;
/** The detention policies the card lists; the rest are counted. */
const POLICY_ROWS = 3;

/**
 * What the card is given: the location, and when it was opened from a table or
 * from a card that lists it, the rows in the order shown, how to step to
 * another, and the way back.
 */
export type LocationCardProps = {
  recordId: string;
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the location was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];

/**
 * One location as a dispatcher or a planner reads it, opened from a table's
 * row or looked up by the agent: where it is and the fence round it, how long
 * trucks wait there and what that has cost, the detention terms written for
 * it, the loads that stop there and the service failures at its docks. It
 * reads; changing the address, fence and category stays on the location's
 * page, one click away.
 *
 * The card steps through the rows it was opened from in the order shown.
 */
export function DeskLocation({
  recordId,
  order = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: LocationCardProps) {
  const t = useT();
  const query = useQuery({
    ...queries.location.card(recordId),
    retry: false,
    placeholderData: keepPreviousData,
  });
  const location = query.data;

  if (!location) {
    return query.isError ? (
      <CardUnreadable
        kind={t("Location")}
        title={t("This location could not be read")}
        error={query.error}
        onBack={onBack}
        backLabel={backLabel}
      />
    ) : (
      <DeskLocationSkeleton stepper={onSelect !== undefined && order.length > 1} />
    );
  }

  return (
    <LocationCard
      key={location.id}
      location={location}
      order={order}
      onSelect={onSelect}
      onBack={onBack}
      backLabel={backLabel}
    />
  );
}

/**
 * The location as an artifact of its own, when the agent looked it up: the
 * same card, with nothing to step through and nowhere to go back to.
 */
export function DeskLocationArtifact({ recordId }: RecordArtifactProps) {
  return <DeskLocation recordId={recordId} />;
}

function LocationCard({
  location,
  order,
  onSelect,
  onBack,
  backLabel,
}: Required<Pick<LocationCardProps, "order">> &
  Omit<LocationCardProps, "order" | "recordId"> & { location: DeskLocationRecord }) {
  const t = useT();
  const access = useCardAccess(LOCATION_SECTIONS);
  const [span] = useState(() => dwellWindow(Math.floor(Date.now() / 1000)));
  const id = location.id;

  const dwell = useQuery({
    ...queries.location.dwell(id, span),
    retry: false,
    enabled: access.detention,
  });
  const policies = useQuery({
    ...queries.location.policies(id, POLICY_ROWS),
    retry: false,
    enabled: access.detention,
  });
  const shipments = useRecordShipments({
    scope: { kind: "location", id },
    pageSize: SHIPMENT_ROWS,
    enabled: access.shipments,
  });
  const failures = useInfiniteQuery({
    ...locationFailuresQuery(id, FAILURE_ROWS),
    retry: false,
    enabled: access.failures,
  });

  const standing = locationStanding(location.status, t);
  const neighbors = neighborsIn(order, id);
  const category = location.locationCategory ?? null;
  const point = mapPoint(location.latitude, location.longitude);
  const outline = geofenceOutline(location);
  const zoom = point ? locationZoom(point, outline, MAP_SIZE) : undefined;
  const street = streetLine(location);
  const city = cityLine(location);
  const failureRows = useMemo(
    () => failures.data?.pages.flatMap((page) => page.rows) ?? [],
    [failures.data],
  );
  const failureIds = useMemo(() => failureRows.map((row) => row.id), [failureRows]);

  const backToLocation = t("Back to {0}", location.name);
  const listedShipments = useListedRecords({
    entity: "shipment",
    order: shipments.ids,
    backLabel: backToLocation,
  });
  const listedFailures = useListedRecords({
    entity: "service_failure",
    order: failureIds,
    backLabel: backToLocation,
  });

  // A listed record opened from the card takes the card's place, and its way
  // back returns here.
  const opened = listedShipments.opened ?? listedFailures.opened;
  if (opened) {
    return opened;
  }

  return (
    <RecordCard className={cn("dk-lo", `dk-lo-${standing.tone}`)}>
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={[
          location.code,
          ...(category ? [choiceLabel(locationCategoryTypeChoices, category.type)] : []),
          ...(category?.facilityType
            ? [choiceLabel(facilityTypeChoices, category.facilityType)]
            : []),
        ]}
        stepper={
          onSelect &&
          order.length > 1 && (
            <CardStepper
              neighbors={neighbors}
              onStep={onSelect}
              previousLabel={t("Previous location")}
              nextLabel={t("Next location")}
            />
          )
        }
        title={location.name}
        subtitle={[street, city].filter(Boolean).join(", ") || t("No address on file")}
        figure={
          access.detention ? (
            <DwellFigure dwell={dwell.data} pending={dwell.isPending} />
          ) : undefined
        }
        meta={
          <>
            <CardStatus key={standing.label} tone={standing.tone}>
              {standing.label}
            </CardStatus>
            <CardFacts>{[...facilityTerms(category, t), location.timezone.trim()]}</CardFacts>
          </>
        }
      />

      <CardSection title={t("Where it is")}>
        <CardWhere
          point={point}
          zoom={zoom}
          outline={outline}
          place={street || null}
          placeMissing={t("No street address on file")}
          line={geofenceLine(location, t)}
          lineOff={!outline}
          facts={[city, point ? "" : t("Not geocoded")]}
          mapLabel={t("Map of {0}", location.name)}
          noMapLabel={t("This location has not been geocoded")}
        />
      </CardSection>

      {access.detention && (
        <CardSection title={t("Dwell, last {0} days", DWELL_WINDOW_DAYS)}>
          <Dwell dwell={dwell.data} pending={dwell.isPending} error={dwell.error} />
        </CardSection>
      )}

      {access.detention && (
        <CardSection
          title={t("Detention terms")}
          aside={
            policies.data && policies.data.total > policies.data.rows.length ? (
              <span>
                {t(
                  "{0} of {1, plural, one {# policy} other {# policies}}",
                  policies.data.rows.length,
                  policies.data.total,
                )}
              </span>
            ) : undefined
          }
        >
          <Policies policies={policies.data} pending={policies.isPending} error={policies.error} />
        </CardSection>
      )}

      {access.shipments && (
        <CardShipments
          title={t("Recent shipments")}
          shipments={shipments}
          onOpen={listedShipments.open}
          emptyText={t("No load has stopped here yet.")}
        />
      )}

      {access.failures && (
        <CardSection
          title={t("Service failures")}
          aside={
            failures.data?.pages[0]?.total ? (
              <span className="dk-warn">
                {t("{0, plural, one {# failure} other {# failures}}", failures.data.pages[0].total)}
              </span>
            ) : undefined
          }
        >
          <Failures
            rows={failures.data ? failureRows : undefined}
            total={failures.data?.pages[0]?.total ?? null}
            pending={failures.isPending}
            error={failures.error}
            hasMore={failures.hasNextPage}
            loadingMore={failures.isFetchingNextPage}
            onMore={() => void failures.fetchNextPage()}
            onOpen={listedFailures.open}
          />
        </CardSection>
      )}

      <CardSection title={t("Details")}>
        <CardFactGrid facts={locationFacts(location, t)} />
      </CardSection>

      <CardBar>
        <CardBarHint>
          {t("Change the address, fence and category on the location's page")}
        </CardBarHint>
        <Link className="dk-ax-btn dk-ink" to={recordPath("location", id)}>
          <CardIcon name="ext" size={13} />
          {t("Open location")}
        </Link>
      </CardBar>
    </RecordCard>
  );
}

/** The average wait at the location, at the head of the card. */
function DwellFigure({
  dwell,
  pending,
}: {
  dwell: LocationDwell | null | undefined;
  pending: boolean;
}) {
  const t = useT();
  if (pending || !dwell) {
    return (
      <>
        <b className="dk-lo-wait">—</b>
        <span>{pending ? t("Average dwell") : t("No stops scored yet")}</span>
      </>
    );
  }
  return (
    <>
      <b key={dwell.avgDwellMinutes}>{waitText(dwell.avgDwellMinutes)}</b>
      <span>{t("average dwell")}</span>
    </>
  );
}

/** How long trucks waited at the location, and what the detention came to. */
function Dwell({
  dwell,
  pending,
  error,
}: {
  dwell: LocationDwell | null | undefined;
  pending: boolean;
  error: unknown;
}) {
  const t = useT();
  if (error && dwell === undefined) {
    return (
      <CardSectionError>
        {describeApiError(error, t("The dwell could not be read."))}
      </CardSectionError>
    );
  }
  if (pending || dwell === undefined) {
    return <DwellSkeleton />;
  }
  if (dwell === null) {
    return (
      <p className="dk-rec-note">
        {t("No stop here has been scored for detention in the last {0} days.", DWELL_WINDOW_DAYS)}
      </p>
    );
  }
  return (
    <>
      <CardStats stats={dwellStats(dwell, t)} />
      <p className="dk-rec-note dk-lo-money">{dwellMoneyLine(dwell, t)}</p>
    </>
  );
}

/** The detention policies written for the location, highest priority first. */
function Policies({
  policies,
  pending,
  error,
}: {
  policies: LocationPolicies | undefined;
  pending: boolean;
  error: unknown;
}) {
  const t = useT();
  if (error && !policies) {
    return (
      <CardSectionError>
        {describeApiError(error, t("The detention terms could not be read."))}
      </CardSectionError>
    );
  }
  if (pending || !policies) {
    return <CardSectionWait rows={1} />;
  }
  if (policies.rows.length === 0) {
    return (
      <p className="dk-rec-note">
        {t("No policy is written for this location; the customer's or the organization's applies.")}
      </p>
    );
  }
  return (
    <ul className="dk-rec-list dk-lo-terms">
      {policies.rows.map((policy) => (
        <li key={policy.id} className={cn(policy.status !== "Active" && "dk-off")}>
          <span className="dk-rec-main">
            <b>{policy.name}</b>
            <span className="dk-rec-sub">{policyScope(policy, t)}</span>
          </span>
          <span className="dk-lo-free">
            {policy.status === "Active"
              ? policyFreeTime(policy, t)
              : choiceLabel(detentionPolicyStatusChoices, policy.status)}
          </span>
        </li>
      ))}
    </ul>
  );
}

/** The service failures at the location's docks, newest first, a page at a time. */
function Failures({
  rows,
  total,
  pending,
  error,
  hasMore,
  loadingMore,
  onMore,
  onOpen,
}: {
  rows: readonly LocationFailure[] | undefined;
  total: number | null;
  pending: boolean;
  error: unknown;
  hasMore: boolean;
  loadingMore: boolean;
  onMore: () => void;
  onOpen: ((failureId: string) => void) | undefined;
}) {
  const t = useT();
  if (error && !rows) {
    return (
      <CardSectionError>
        {describeApiError(error, t("The service failures could not be read."))}
      </CardSectionError>
    );
  }
  if (pending || !rows) {
    return <CardSectionWait rows={2} />;
  }
  if (rows.length === 0) {
    return <p className="dk-rec-note">{t("No service failure has been recorded here.")}</p>;
  }
  return (
    <>
      <div className="dk-rec-list">
        {rows.map((failure) => (
          <CardRecordRow
            key={failure.id}
            to={recordPath("service_failure", failure.id)}
            onOpen={onOpen && (() => onOpen(failure.id))}
          >
            <span className="dk-rec-main">
              <b>{failure.number}</b>
              <span className="dk-rec-sub">{failureLine(failure, t)}</span>
            </span>
            <span
              className={cn(
                "dk-lo-fst",
                failure.status === "Open" && "dk-warn",
                failure.status === "Resolved" && "dk-ok",
              )}
            >
              {choiceLabel(serviceFailureStatusChoices, failure.status)}
            </span>
          </CardRecordRow>
        ))}
      </div>
      {hasMore && (
        <CardMore
          shown={
            total !== null ? t("{0} of {1} shown", rows.length, total) : t("{0} shown", rows.length)
          }
          onMore={onMore}
          loading={loadingMore}
          label={t("Show more")}
          loadingLabel={t("Reading more service failures…")}
        />
      )}
      {error !== null && error !== undefined && (
        <CardSectionError>
          {describeApiError(error, t("The next service failures could not be read."))}
        </CardSectionError>
      )}
    </>
  );
}
