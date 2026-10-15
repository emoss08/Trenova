import { recordPath } from "@/config/record-links";
import { describeApiError } from "@/lib/api-error-message";
import type {
  DeskWorker as DeskWorkerRecord,
  DeskWorkerHos,
  DeskWorkerStanding,
} from "@/lib/graphql/worker-card";
import { choiceLabel, driverTypeChoices, workerTypeChoices } from "@/lib/choices";
import { dutyStatusLabel } from "@/lib/hos";
import { worker as workerQueries } from "@/lib/queries/worker";
import { useQuery } from "@tanstack/react-query";
import {
  formatClockDurationMs,
  formatDurationMs,
  formatSecondsAgo,
} from "@trenova/shared/lib/date";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, toTitleCase } from "@trenova/shared/lib/utils";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import { useCardAccess } from "../../record-card/card-access";
import { CardDocuments } from "../../record-card/card-documents";
import { CardShipments } from "../../record-card/card-shipments";
import {
  CardAvatar,
  CardBar,
  CardBarHint,
  CardCheck,
  CardChecks,
  CardFacts,
  CardHead,
  CardIcon,
  CardOpenLink,
  CardSection,
  CardSectionError,
  CardStatus,
  CardStepper,
  CardUnreadable,
  RecordCard,
} from "../../record-card/record-card";
import { cardDate, neighborsIn } from "../../record-card/record-card-values";
import { useRecordShipments } from "../../record-card/record-shipments";
import { CardChecksSkeleton } from "../../skeletons/record-card-skeleton";
import { useListedRecords, type RecordArtifactProps } from "../registry";
import {
  concernState,
  dispatchStanding,
  hosClocks,
  licenseLine,
  otherCredentialChecks,
  qualificationChecks,
  standingConcerns,
  standingLabel,
  standingTone,
  workerFacts,
  workerName,
} from "./worker-card";
import { CredentialWallet, HosRings, WorkerReach } from "./worker-parts";
import { WORKER_SECTIONS } from "./worker-sections";
import { DeskWorkerSkeleton, HosClocksSkeleton, WalletSkeleton } from "./worker-skeleton";

/** The shipments the card reads at a time; the next page is read on request. */
const SHIPMENT_ROWS = 5;
/** The latest violations the card lists under the clocks. */
const VIOLATION_ROWS = 3;
const STANDING_WAIT = [[40, 56]] as const;
/** The facts the card draws as ways to reach the worker rather than in their record. */
const REACH_KEYS = new Set(["phone", "email"]);
const CREDENTIAL_WAIT = 2;

/**
 * What the card is given: the worker, and when they were opened from a table
 * or from a card that lists them, the rows in the order shown, how to step to
 * another, and the way back.
 */
export type WorkerCardProps = {
  recordId: string;
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the worker was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];

/**
 * One worker as a dispatcher or a safety manager reads them, opened from a
 * table's row or looked up by the agent: whether they can be put on a load,
 * how much of their day and week is left to drive, what stands against them,
 * the dates their qualification hangs on, the loads they ran, how to reach
 * them and their documents. It reads; changing their profile, credentials and
 * status stays on the worker's page, one click away.
 *
 * The card steps through the rows it was opened from in the order shown.
 * A step draws the next worker's skeleton until it is read, never the one it left.
 */
export function DeskWorker({
  recordId,
  order = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: WorkerCardProps) {
  const t = useT();
  const query = useQuery({ ...workerQueries.card(recordId), retry: false });
  const worker = query.data;

  if (!worker) {
    return query.isError ? (
      <CardUnreadable
        kind={t("Worker")}
        title={t("This worker could not be read")}
        error={query.error}
        onBack={onBack}
        backLabel={backLabel}
      />
    ) : (
      <DeskWorkerSkeleton stepper={onSelect !== undefined && order.length > 1} />
    );
  }

  return (
    <WorkerCard
      key={worker.id}
      worker={worker}
      order={order}
      onSelect={onSelect}
      onBack={onBack}
      backLabel={backLabel}
    />
  );
}

/**
 * The worker as an artifact of their own, when the agent looked them up: the
 * same card, with nothing to step through and nowhere to go back to.
 */
export function DeskWorkerArtifact({ recordId }: RecordArtifactProps) {
  return <DeskWorker recordId={recordId} />;
}

function WorkerCard({
  worker,
  order,
  onSelect,
  onBack,
  backLabel,
}: Required<Pick<WorkerCardProps, "order">> &
  Omit<WorkerCardProps, "order" | "recordId"> & { worker: DeskWorkerRecord }) {
  const t = useT();
  const access = useCardAccess(WORKER_SECTIONS);
  const [now] = useState(() => Math.floor(Date.now() / 1000));
  const id = worker.id;

  const hos = useQuery({ ...workerQueries.hos(id, VIOLATION_ROWS), retry: false });
  const standing = useQuery({ ...workerQueries.standing(id), retry: false });
  const credentials = useQuery({
    ...workerQueries.credentials(id),
    retry: false,
    enabled: access.credentials,
  });
  const shipments = useRecordShipments({
    scope: { kind: "worker", id },
    pageSize: SHIPMENT_ROWS,
    enabled: access.shipments,
  });

  const name = workerName(worker, t);
  const dispatch = dispatchStanding(worker, t);
  const neighbors = neighborsIn(order, id);
  const profile = worker.profile ?? null;
  const tractorId = hos.data?.state?.currentTractorId ?? null;

  const backToWorker = t("Back to {0}", name);
  const listedShipments = useListedRecords({
    entity: "shipment",
    order: shipments.ids,
    backLabel: backToWorker,
  });
  const tractorIds = useMemo(() => (tractorId ? [tractorId] : NO_ROWS), [tractorId]);
  const listedTractor = useListedRecords({
    entity: "tractor",
    order: tractorIds,
    backLabel: backToWorker,
  });

  // A listed record opened from the card takes the card's place, and its way
  // back returns here.
  const opened = listedShipments.opened ?? listedTractor.opened;
  if (opened) {
    return opened;
  }

  return (
    <RecordCard className={cn("dk-wk", `dk-wk-${dispatch.tone}`)}>
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={[
          choiceLabel(workerTypeChoices, worker.type),
          ...(worker.driverType ? [choiceLabel(driverTypeChoices, worker.driverType)] : []),
          ...(worker.fleetCode ? [t("Fleet {0}", worker.fleetCode.code)] : []),
        ]}
        stepper={
          onSelect &&
          order.length > 1 && (
            <CardStepper
              neighbors={neighbors}
              onStep={onSelect}
              previousLabel={t("Previous worker")}
              nextLabel={t("Next worker")}
            />
          )
        }
        lead={<CardAvatar name={name} size="lg" />}
        title={name}
        subtitle={profile ? licenseLine(profile, t) : t("No driver profile on file")}
        figure={<DriveFigure hos={hos.data} pending={hos.isPending} />}
        meta={
          <>
            <CardStatus key={dispatch.label} tone={dispatch.tone} mark={dispatch.mark}>
              {dispatch.label}
            </CardStatus>
            <CardFacts>
              {[
                worker.manager ? t("Managed by {0}", worker.manager.name) : "",
                [worker.city.trim(), worker.state?.abbreviation ?? ""].filter(Boolean).join(", "),
              ]}
            </CardFacts>
            {dispatch.reason && <p className="dk-wk-why">{dispatch.reason}</p>}
          </>
        }
      />

      <CardSection
        title={t("Hours of service")}
        aside={
          tractorId ? (
            <CardOpenLink
              to={recordPath("tractor", tractorId)}
              onOpen={listedTractor.open && (() => listedTractor.open?.(tractorId))}
            >
              {t("Open tractor")}
            </CardOpenLink>
          ) : undefined
        }
      >
        <HoursOfService hos={hos.data} pending={hos.isPending} error={hos.error} now={now} />
      </CardSection>

      <StandingSection
        standing={standing.data}
        pending={standing.isPending}
        error={standing.error}
      />

      {profile && (
        <CredentialWallet
          title={t("Qualifications")}
          checks={qualificationChecks(profile, now, t)}
        />
      )}

      {access.credentials && (
        <OtherCredentials
          rows={credentials.data}
          pending={credentials.isPending}
          error={credentials.error}
        />
      )}

      {access.shipments && (
        <CardShipments
          title={t("Recent shipments")}
          shipments={shipments}
          onOpen={listedShipments.open}
          emptyText={t("This worker has not run a load yet.")}
          layout="table"
        />
      )}

      <CardSection title={t("Reach and record")}>
        <WorkerReach
          phone={worker.phoneNumber.trim()}
          email={worker.email.trim()}
          facts={workerFacts(worker, t).filter((fact) => !REACH_KEYS.has(fact.key))}
        />
      </CardSection>

      <CardDocuments resourceType="worker" resourceId={id} manageTo={recordPath("worker", id)} />

      <CardBar>
        <CardBarHint>
          {t("Change the profile, credentials and status on the worker's page")}
        </CardBarHint>
        <Link className="dk-ax-btn dk-ink" to={recordPath("worker", id)}>
          <CardIcon name="ext" size={13} />
          {t("Open worker")}
        </Link>
      </CardBar>
    </RecordCard>
  );
}

/** The drive time the driver has left, at the head of the card. */
function DriveFigure({ hos, pending }: { hos: DeskWorkerHos | undefined; pending: boolean }) {
  const t = useT();
  const state = hos?.state ?? null;
  if (pending || !state) {
    return (
      <>
        <b className="dk-wk-wait">—</b>
        <span>{pending ? t("Drive time") : t("No hours reported")}</span>
      </>
    );
  }
  const left = Math.max(0, state.driveRemainingMs);
  return (
    <>
      <b key={left} className={cn(left <= 0 && "dk-bad")}>
        {formatClockDurationMs(left)}
      </b>
      <span>{t("drive time left")}</span>
    </>
  );
}

/**
 * The driver's clocks as telematics last reported them, each a meter of what
 * is left of its limit, their duty status and when it was reported, and their
 * latest violations.
 */
function HoursOfService({
  hos,
  pending,
  error,
  now,
}: {
  hos: DeskWorkerHos | undefined;
  pending: boolean;
  error: unknown;
  now: number;
}) {
  const t = useT();
  if (error && !hos) {
    return (
      <CardSectionError>
        {describeApiError(error, t("The hours of service could not be read."))}
      </CardSectionError>
    );
  }
  if (pending || !hos) {
    return <HosClocksSkeleton />;
  }
  const state = hos.state;
  return (
    <>
      {state ? (
        <>
          <p className="dk-wk-duty">
            <b>{state.dutyStatus ? dutyStatusLabel(state.dutyStatus) : t("Duty status unknown")}</b>
            <span>
              {[
                t("Reported {0}", formatSecondsAgo(Math.max(0, now - state.recordedAt))),
                state.rulesetCycle?.trim() ?? "",
              ]
                .filter(Boolean)
                .join(" · ")}
            </span>
          </p>
          <HosRings clocks={hosClocks(state, t)} />
        </>
      ) : (
        <p className="dk-rec-note">{t("Telematics has not reported this worker's hours.")}</p>
      )}
      {hos.violations.length > 0 && (
        <ol className="dk-wk-viol" aria-label={t("Latest violations")}>
          {hos.violations.map((violation) => (
            <li key={`${violation.violationType}-${violation.violationStartAt}`}>
              <span className="dk-rec-main">
                <b>{toTitleCase(violation.violationType)}</b>
                <span className="dk-rec-sub">
                  {violation.description?.trim() || t("No description from telematics")}
                </span>
              </span>
              <span className="dk-wk-viol-at">
                {t(
                  "{0} · {1}",
                  cardDate(violation.violationStartAt),
                  formatDurationMs(violation.durationMs),
                )}
              </span>
            </li>
          ))}
        </ol>
      )}
    </>
  );
}

/**
 * Whether the worker is in good standing, and what stands against them beyond
 * what the card shows elsewhere: their training, safety record, discipline,
 * open checklist and reviews.
 */
function StandingSection({
  standing,
  pending,
  error,
}: {
  standing: DeskWorkerStanding | undefined;
  pending: boolean;
  error: unknown;
}) {
  const t = useT();
  const concerns = standing ? standingConcerns(standing.concerns) : [];
  return (
    <CardSection
      title={t("Standing")}
      aside={
        standing ? (
          <span className={cn(standingTone(standing.standing) === "success" ? "dk-ok" : "dk-warn")}>
            {standingLabel(standing.standing, t)}
          </span>
        ) : undefined
      }
    >
      {error && !standing ? (
        <CardSectionError>
          {describeApiError(error, t("The worker's standing could not be read."))}
        </CardSectionError>
      ) : pending || !standing ? (
        <CardChecksSkeleton rows={STANDING_WAIT} />
      ) : concerns.length > 0 ? (
        <CardChecks>
          {concerns.map((concern) => (
            <CardCheck
              key={concern.code}
              state={concernState(concern.severity)}
              title={concern.headline}
              detail={concern.detail}
            />
          ))}
        </CardChecks>
      ) : (
        <p className="dk-wk-clear">
          <CardIcon name="check" size={12} stroke={2.6} />
          <b>{t("Nothing else stands against them")}</b>
          <span>{t("Training, safety, discipline and reviews are in order")}</span>
        </p>
      )}
    </CardSection>
  );
}

/** The worker's credentials beyond their license and medical dates, worst first. */
function OtherCredentials({
  rows,
  pending,
  error,
}: {
  rows: Parameters<typeof otherCredentialChecks>[0] | undefined;
  pending: boolean;
  error: unknown;
}) {
  const t = useT();
  if (error && !rows) {
    return (
      <CardSection title={t("Other credentials")}>
        <CardSectionError>
          {describeApiError(error, t("The credentials could not be read."))}
        </CardSectionError>
      </CardSection>
    );
  }
  if (pending || !rows) {
    return (
      <CardSection title={t("Other credentials")}>
        <WalletSkeleton tiles={CREDENTIAL_WAIT} />
      </CardSection>
    );
  }
  const checks = otherCredentialChecks(rows, t);
  if (checks.length === 0) {
    return (
      <CardSection title={t("Other credentials")}>
        <p className="dk-rec-note">{t("No credentials beyond the license and medical card.")}</p>
      </CardSection>
    );
  }
  return <CredentialWallet title={t("Other credentials")} checks={checks} />;
}
