import { recordPath } from "@/config/record-links";
import type { DeskServiceFailure as DeskServiceFailureRecord } from "@/lib/graphql/service-failure-card";
import { choiceLabel, serviceFailureSourceChoices, serviceFailureTypeChoices } from "@/lib/choices";
import { serviceFailure } from "@/lib/queries/service-failure";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo } from "react";
import { Link } from "react-router";
import { CardActivity } from "../../record-card/card-activity";
import { useCardAccess } from "../../record-card/card-access";
import { CardFailures } from "../../record-card/card-failures";
import { CardPath } from "../../record-card/card-path";
import { CardWhere, mapPoint } from "../../record-card/card-where";
import {
  CardBar,
  CardBarHint,
  CardFacts,
  CardHead,
  CardIcon,
  CardRecordRow,
  CardSection,
  CardStatus,
  CardStepper,
  CardUnreadable,
  RecordCard,
} from "../../record-card/record-card";
import {
  cardDate,
  cardMoment,
  cityLine,
  neighborsIn,
  waitText,
} from "../../record-card/record-card-values";
import { useRecordActivity } from "../../record-card/record-activity";
import { useRecordFailures, withoutFailure } from "../../record-card/record-failures";
import { useListedRecords, type RecordArtifactProps } from "../registry";
import {
  ediCodes,
  failureCause,
  failurePlace,
  failureStanding,
  isUnsettled,
  missClock,
  reviewSteps,
  stopLine,
} from "./service-failure-card";
import { EdiCodes, MissClock } from "./service-failure-parts";
import { SERVICE_FAILURE_SECTIONS } from "./service-failure-sections";
import { DeskServiceFailureSkeleton } from "./service-failure-skeleton";

/** The load's other failures the card reads at a time; the next page is read on request. */
const FAILURE_ROWS = 5;
/** The activity entries the card reads at a time. */
const ACTIVITY_ROWS = 5;

/**
 * What the card is given: the failure, and when it was opened from a table or
 * from a card that lists it, the rows in the order shown, how to step to
 * another, and the way back.
 */
export type ServiceFailureCardProps = {
  recordId: string;
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the failure was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];

/**
 * One service failure as a customer service rep or an operations manager
 * reads it, opened from a table's row or looked up by the agent: where and how
 * late the truck was, against what it was due, on which load for which
 * customer, why, who has reviewed and settled it, and what else went wrong on
 * the same load. It reads; reviewing, resolving and voiding stay on the
 * failure's page, one click away.
 *
 * The card steps through the rows it was opened from in the order shown.
 */
export function DeskServiceFailure({
  recordId,
  order: rowOrder = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: ServiceFailureCardProps) {
  const t = useT();
  const query = useQuery({
    ...serviceFailure.card(recordId),
    retry: false,
    placeholderData: keepPreviousData,
  });
  const record = query.data;

  if (!record) {
    return query.isError ? (
      <CardUnreadable
        kind={t("Service failure")}
        title={t("This service failure could not be read")}
        error={query.error}
        onBack={onBack}
        backLabel={backLabel}
      />
    ) : (
      <DeskServiceFailureSkeleton stepper={onSelect !== undefined && rowOrder.length > 1} />
    );
  }

  return (
    <ServiceFailureCard
      key={record.id}
      record={record}
      rowOrder={rowOrder}
      onSelect={onSelect}
      onBack={onBack}
      backLabel={backLabel}
    />
  );
}

/**
 * The failure as an artifact of its own, when the agent looked it up: the same
 * card, with nothing to step through and nowhere to go back to.
 */
export function DeskServiceFailureArtifact({ recordId }: RecordArtifactProps) {
  return <DeskServiceFailure recordId={recordId} />;
}

function ServiceFailureCard({
  record,
  rowOrder,
  onSelect,
  onBack,
  backLabel,
}: Omit<ServiceFailureCardProps, "order" | "recordId"> & {
  record: DeskServiceFailureRecord;
  rowOrder: readonly string[];
}) {
  const t = useT();
  const access = useCardAccess(SERVICE_FAILURE_SECTIONS);
  const id = record.id;
  const load = record.shipment ?? null;
  const customer = load?.customer ?? null;
  const location = record.stop?.location ?? null;

  const others = useRecordFailures({
    scope: { kind: "shipment", id: record.shipmentId },
    pageSize: FAILURE_ROWS,
    enabled: true,
  });
  const otherFailures = withoutFailure(others, id);
  const activity = useRecordActivity({
    resourceId: id,
    pageSize: ACTIVITY_ROWS,
    enabled: access.activity,
  });

  const standing = failureStanding(record.status);
  const neighbors = neighborsIn(rowOrder, id);
  const place = failurePlace(record, t);
  const town = location ? cityLine(location) : "";
  const point = mapPoint(location?.latitude, location?.longitude);
  const cause = failureCause(record);
  const unsettled = isUnsettled(record.status);

  const backToFailure = t("Back to {0}", record.number);
  const loadIds = useMemo(() => (load ? [load.id] : NO_ROWS), [load]);
  const customerIds = useMemo(() => (customer ? [customer.id] : NO_ROWS), [customer]);
  const otherIds = useMemo(
    () => otherFailures.rows?.map((row) => row.id) ?? NO_ROWS,
    [otherFailures.rows],
  );
  const listedLoad = useListedRecords({
    entity: "shipment",
    order: loadIds,
    backLabel: backToFailure,
  });
  const listedCustomer = useListedRecords({
    entity: "customer",
    order: customerIds,
    backLabel: backToFailure,
  });
  const listedFailures = useListedRecords({
    entity: "service_failure",
    order: otherIds,
    backLabel: backToFailure,
  });

  // A listed record opened from the card takes the card's place, and its way
  // back returns here.
  const opened = listedLoad.opened ?? listedCustomer.opened ?? listedFailures.opened;
  if (opened) {
    return opened;
  }

  return (
    <RecordCard className={cn("dk-svf", `dk-svf-${record.status.toLowerCase()}`)}>
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={[
          record.number,
          choiceLabel(serviceFailureTypeChoices, record.type),
          choiceLabel(serviceFailureSourceChoices, record.source),
        ]}
        stepper={
          onSelect &&
          rowOrder.length > 1 && (
            <CardStepper
              neighbors={neighbors}
              onStep={onSelect}
              previousLabel={t("Previous failure")}
              nextLabel={t("Next failure")}
            />
          )
        }
        title={place}
        subtitle={town && town !== place ? town : undefined}
        figure={
          <>
            <b className="dk-svf-late">{waitText(record.lateMinutes)}</b>
            <span>{t("late")}</span>
          </>
        }
        meta={
          <>
            <CardStatus key={record.status} tone={standing.tone}>
              {standing.label}
            </CardStatus>
            <CardFacts>
              {[
                stopLine(record, t),
                load ? t("Shipment {0}", load.proNumber) : "",
                t("Recorded {0}", cardDate(record.detectedAt)),
              ]}
            </CardFacts>
          </>
        }
      />

      <CardSection title={t("What happened")}>
        <MissClock clock={missClock(record, t)} />
      </CardSection>

      <CardSection title={t("Where")}>
        <CardWhere
          point={point}
          place={location?.addressLine1.trim() || null}
          placeMissing={t("No street address on file")}
          line={stopLine(record, t)}
          facts={[town, location && !point ? t("Not geocoded") : ""]}
          mapLabel={t("Map of {0}", place)}
          noMapLabel={t("This stop's location has not been geocoded")}
        />
      </CardSection>

      <CardSection title={t("Load")}>
        {load ? (
          <div className="dk-rec-list">
            <CardRecordRow
              to={recordPath("shipment", load.id)}
              onOpen={listedLoad.open && (() => listedLoad.open?.(load.id))}
            >
              <span className="dk-rec-main">
                <b>{load.proNumber}</b>
                <span className="dk-rec-sub">
                  {load.bol?.trim() ? t("BOL {0}", load.bol.trim()) : t("Shipment")}
                </span>
              </span>
            </CardRecordRow>
            {customer && (
              <CardRecordRow
                to={recordPath("customer", customer.id)}
                onOpen={listedCustomer.open && (() => listedCustomer.open?.(customer.id))}
              >
                <span className="dk-rec-main">
                  <b>{customer.name}</b>
                  <span className="dk-rec-sub">{t("Customer · {0}", customer.code)}</span>
                </span>
              </CardRecordRow>
            )}
          </div>
        ) : (
          <p className="dk-rec-note">{t("The load this failure was recorded on is gone.")}</p>
        )}
      </CardSection>

      <CardSection title={t("Cause")}>
        <div className="dk-svf-cause">
          {cause ? (
            <>
              <div className="dk-svf-cause-h">
                <b>{cause.name}</b>
                <span className="dk-svf-tag">{cause.category}</span>
                <code>{cause.code}</code>
              </div>
              {cause.description && <p>{cause.description}</p>}
            </>
          ) : (
            <p className={cn("dk-rec-note", unsettled && "dk-svf-missing")}>
              {unsettled ? t("No reason has been given yet.") : t("No reason was given.")}
            </p>
          )}
          {record.notes.trim() && (
            <figure className="dk-svf-note">
              <figcaption>{t("Note")}</figcaption>
              <blockquote>{record.notes.trim()}</blockquote>
            </figure>
          )}
          {record.internalNotes.trim() && (
            <figure className="dk-svf-note dk-internal">
              <figcaption>{t("Internal note, kept off the customer's record")}</figcaption>
              <blockquote>{record.internalNotes.trim()}</blockquote>
            </figure>
          )}
        </div>
      </CardSection>

      <CardSection title={t("Review")}>
        <CardPath label={t("Review")} steps={reviewSteps(record, t)} />
      </CardSection>

      <CardSection
        title={t("Other failures on this load")}
        aside={
          otherFailures.total ? (
            <span className="dk-warn">
              {t("{0, plural, one {# other} other {# others}}", otherFailures.total)}
            </span>
          ) : undefined
        }
      >
        <CardFailures
          failures={otherFailures}
          scope="shipment"
          onOpen={listedFailures.open}
          emptyText={t("Nothing else has gone wrong on this load.")}
        />
      </CardSection>

      {access.activity && (
        <CardActivity
          activity={activity}
          emptyText={t("Nothing has been done to this failure yet.")}
        />
      )}

      <CardSection
        title={t("Sent on the 214")}
        aside={<span>{t("Last changed {0}", cardMoment(record.updatedAt))}</span>}
      >
        <EdiCodes codes={ediCodes(record, t)} />
      </CardSection>

      <CardBar>
        <CardBarHint>{t("Review, resolve or void it on its page")}</CardBarHint>
        <Link className="dk-ax-btn dk-ink" to={recordPath("service_failure", id)}>
          <CardIcon name="ext" size={13} />
          {t("Open service failure")}
        </Link>
      </CardBar>
    </RecordCard>
  );
}
