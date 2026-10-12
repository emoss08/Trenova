import { useCarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import { recordPath } from "@/config/record-links";
import { describeApiError } from "@/lib/api-error-message";
import { carrierIntelProviderLabel } from "@/lib/carrier-intelligence";
import { carrierSettlementStatusChoices, choiceLabel } from "@/lib/choices";
import type {
  CarrierEligibility,
  CarrierIntel,
  CarrierSettlementRow,
  CarrierSettlementsPage,
  DeskCarrier as DeskCarrierRecord,
} from "@/lib/graphql/carrier-card";
import { queries } from "@/lib/queries";
import { carrierSettlementsQuery } from "@/lib/queries/carrier";
import { keepPreviousData, useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import { useCardAccess } from "../../record-card/card-access";
import { CardPagedList } from "../../record-card/card-paged-list";
import { CardShipments } from "../../record-card/card-shipments";
import {
  CardBar,
  CardBarHint,
  CardFactGrid,
  CardFacts,
  CardHead,
  CardIcon,
  CardRecordRow,
  CardSection,
  CardSectionError,
  CardStatus,
  CardStepper,
  CardUnreadable,
  needsLook,
  RecordCard,
} from "../../record-card/record-card";
import { usePagedRows, type PagedRows } from "../../record-card/paged-rows";
import { neighborsIn } from "../../record-card/record-card-values";
import { useRecordShipments } from "../../record-card/record-shipments";
import { useListedRecords, type RecordArtifactProps } from "../registry";
import {
  authorityStats,
  basicMeters,
  carrierFacts,
  carrierKicker,
  carrierSubtitle,
  complianceStanding,
  contactLine,
  coverageTiles,
  eligibilityChecks,
  eligibilityVerdict,
  insuranceChecks,
  orderedContacts,
  riskReading,
  settlementAmount,
  settlementLine,
} from "./carrier-card";
import {
  AuthorityPills,
  BasicChart,
  ContactCard,
  CoverageTiles,
  VerdictBanner,
} from "./carrier-parts";
import { CARRIER_SECTIONS } from "./carrier-sections";
import { CarrierIntelSkeleton, DeskCarrierSkeleton, VerdictSkeleton } from "./carrier-skeleton";

/** The shipments the card reads at a time; the next page is read on request. */
const SHIPMENT_ROWS = 5;
/** The settlements the card reads at a time. */
const SETTLEMENT_ROWS = 5;

/**
 * What the card is given: the carrier, and when it was opened from a table or
 * from a card that lists it, the rows in the order shown, how to step to
 * another, and the way back.
 */
export type CarrierCardProps = {
  recordId: string;
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the carrier was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];

/**
 * One carrier as a carrier manager or a dispatcher reads it, opened from a
 * table's row or looked up by the agent: whether it can be tendered a load and
 * why not, the insurance it keeps on file, its authority, safety rating and CSA
 * BASICs, the loads it hauled, what it was paid and who to call. It reads;
 * qualifying, vetting and paying the carrier stay on its page, one click away.
 *
 * The card steps through the rows it was opened from in the order shown.
 */
export function DeskCarrier({
  recordId,
  order = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: CarrierCardProps) {
  const t = useT();
  const query = useQuery({
    ...queries.carrier.card(recordId),
    retry: false,
    placeholderData: keepPreviousData,
  });
  const carrier = query.data;

  if (!carrier) {
    return query.isError ? (
      <CardUnreadable
        kind={t("Carrier")}
        title={t("This carrier could not be read")}
        error={query.error}
        onBack={onBack}
        backLabel={backLabel}
      />
    ) : (
      <DeskCarrierSkeleton stepper={onSelect !== undefined && order.length > 1} />
    );
  }

  return (
    <CarrierCard
      key={carrier.id}
      carrier={carrier}
      order={order}
      onSelect={onSelect}
      onBack={onBack}
      backLabel={backLabel}
    />
  );
}

/**
 * The carrier as an artifact of its own, when the agent looked it up: the same
 * card, with nothing to step through and nowhere to go back to.
 */
export function DeskCarrierArtifact({ recordId }: RecordArtifactProps) {
  return <DeskCarrier recordId={recordId} />;
}

function CarrierCard({
  carrier,
  order,
  onSelect,
  onBack,
  backLabel,
}: Required<Pick<CarrierCardProps, "order">> &
  Omit<CarrierCardProps, "order" | "recordId"> & { carrier: DeskCarrierRecord }) {
  const t = useT();
  const access = useCardAccess(CARRIER_SECTIONS);
  const [now] = useState(() => Math.floor(Date.now() / 1000));
  const id = carrier.id;

  const eligibility = useQuery({
    ...queries.carrier.eligibility(id),
    retry: false,
    enabled: access.eligibility,
  });
  const intel = useQuery({
    ...queries.carrier.intel(id),
    retry: false,
    enabled: access.intelligence,
  });
  const shipments = useRecordShipments({
    scope: { kind: "carrier", id },
    pageSize: SHIPMENT_ROWS,
    enabled: access.shipments,
  });
  const settlements = usePagedRows<CarrierSettlementRow, CarrierSettlementsPage>(
    useInfiniteQuery({
      ...carrierSettlementsQuery(id, SETTLEMENT_ROWS),
      retry: false,
      enabled: access.settlements,
    }),
  );

  const standing = complianceStanding(carrier, t);
  const risk = riskReading(carrier.intelRiskLevel);
  const neighbors = neighborsIn(order, id);
  const policies = carrier.insurancePolicies ?? [];
  const insuranceOpen = insuranceChecks(policies, now, t).filter((check) =>
    needsLook(check.state),
  ).length;
  const contacts = orderedContacts(carrier.contacts ?? []);
  const settlementIds = useMemo(
    () => settlements.rows?.map((row) => row.id) ?? NO_ROWS,
    [settlements.rows],
  );

  const backToCarrier = t("Back to {0}", carrier.name);
  const listedShipments = useListedRecords({
    entity: "shipment",
    order: shipments.ids,
    backLabel: backToCarrier,
  });
  const listedSettlements = useListedRecords({
    entity: "carrier_settlement",
    order: settlementIds,
    backLabel: backToCarrier,
  });

  // A listed record opened from the card takes the card's place, and its way
  // back returns here.
  const opened = listedShipments.opened ?? listedSettlements.opened;
  if (opened) {
    return opened;
  }

  return (
    <RecordCard className={cn("dk-ca", `dk-ca-${standing.tone}`)}>
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={carrierKicker(carrier, t)}
        stepper={
          onSelect &&
          order.length > 1 && (
            <CardStepper
              neighbors={neighbors}
              onStep={onSelect}
              previousLabel={t("Previous carrier")}
              nextLabel={t("Next carrier")}
            />
          )
        }
        title={carrier.name}
        subtitle={carrierSubtitle(carrier, t)}
        figure={
          <>
            <b className={cn(!risk && "dk-ca-wait", risk && `dk-${risk.state}`)}>
              {risk?.label ?? "—"}
            </b>
            <span>{risk ? t("risk") : t("Not vetted")}</span>
          </>
        }
        meta={
          <>
            <CardStatus key={standing.label} tone={standing.tone} mark={standing.mark}>
              {standing.label}
            </CardStatus>
            <CardFacts>
              {[
                carrier.disqualifiedReason?.trim() ?? "",
                carrier.intelReviewRequired ? t("Review required") : "",
              ]}
            </CardFacts>
          </>
        }
      />

      {access.eligibility && (
        <Eligibility
          eligibility={eligibility.data}
          pending={eligibility.isPending}
          error={eligibility.error}
        />
      )}

      <CardSection
        title={t("Insurance")}
        aside={
          policies.length > 0 ? (
            insuranceOpen > 0 ? (
              <span className="dk-warn">
                {t("{0, plural, one {# needs a look} other {# need a look}}", insuranceOpen)}
              </span>
            ) : (
              <span className="dk-ok">{t("Nothing due")}</span>
            )
          ) : undefined
        }
      >
        {policies.length > 0 ? (
          <CoverageTiles tiles={coverageTiles(policies, now, t)} />
        ) : (
          <p className="dk-rec-note">{t("No insurance policy is on file.")}</p>
        )}
      </CardSection>

      {access.intelligence && (
        <AuthorityAndSafety
          intel={intel.data}
          pending={intel.isPending}
          error={intel.error}
          now={now}
        />
      )}

      {access.shipments && (
        <CardShipments
          title={t("Recent shipments")}
          shipments={shipments}
          onOpen={listedShipments.open}
          emptyText={t("This carrier has not hauled a load yet.")}
        />
      )}

      {access.settlements && (
        <CardSection
          title={t("Settlements")}
          aside={
            settlements.total ? (
              <span>
                {t("{0, plural, one {# settlement} other {# settlements}}", settlements.total)}
              </span>
            ) : undefined
          }
        >
          <Settlements settlements={settlements} onOpen={listedSettlements.open} />
        </CardSection>
      )}

      <CardSection title={t("Contacts")}>
        {contacts.length > 0 ? (
          <ul className="dk-ca-contacts">
            {contacts.map((contact) => (
              <ContactCard
                key={contact.id}
                name={contact.name}
                line={contactLine(contact) || t("No way to reach them on file")}
                role={[
                  contact.isPrimary ? t("Primary") : "",
                  contact.receivesRateConfirmations ? t("Gets rate confirmations") : "",
                ]
                  .filter(Boolean)
                  .join(" · ")}
              />
            ))}
          </ul>
        ) : (
          <p className="dk-rec-note">{t("No contact is on file.")}</p>
        )}
      </CardSection>

      <CardSection title={t("Details")}>
        <CardFactGrid facts={carrierFacts(carrier, t)} />
      </CardSection>

      <CardBar>
        <CardBarHint>{t("Qualify, vet and pay the carrier on its page")}</CardBarHint>
        <Link className="dk-ax-btn dk-ink" to={recordPath("carrier", id)}>
          <CardIcon name="ext" size={13} />
          {t("Open carrier")}
        </Link>
      </CardBar>
    </RecordCard>
  );
}

/** Whether the carrier can be tendered a load now, and what stands in the way. */
function Eligibility({
  eligibility,
  pending,
  error,
}: {
  eligibility: CarrierEligibility | undefined;
  pending: boolean;
  error: unknown;
}) {
  const t = useT();
  const verdict = eligibility ? eligibilityVerdict(eligibility, t) : null;
  const checks = eligibility ? eligibilityChecks(eligibility, t) : [];
  return (
    <CardSection title={t("Can we tender to them")}>
      {error && !eligibility ? (
        <CardSectionError>
          {describeApiError(error, t("Whether the carrier can be tendered could not be read."))}
        </CardSectionError>
      ) : pending || !eligibility ? (
        <VerdictSkeleton />
      ) : (
        verdict && <VerdictBanner verdict={verdict} checks={checks} />
      )}
    </CardSection>
  );
}

/** The carrier's authority, safety rating and CSA BASICs, as intelligence last read them. */
function AuthorityAndSafety({
  intel,
  pending,
  error,
  now,
}: {
  intel: CarrierIntel | null | undefined;
  pending: boolean;
  error: unknown;
  now: number;
}) {
  const t = useT();
  const labels = useCarrierIntelLabels();
  const meters = intel?.profile.basics ? basicMeters(intel.profile.basics, t) : [];
  return (
    <CardSection
      title={t("Authority and safety")}
      aside={
        intel ? (
          <span className="dk-ca-src">
            {t(
              "{0}, {1}",
              carrierIntelProviderLabel(intel.provider),
              formatSecondsAgo(Math.max(0, now - intel.effectiveAsOf)),
            )}
          </span>
        ) : undefined
      }
    >
      {error && intel === undefined ? (
        <CardSectionError>
          {describeApiError(error, t("The carrier's intelligence could not be read."))}
        </CardSectionError>
      ) : pending || intel === undefined ? (
        <CarrierIntelSkeleton />
      ) : intel === null ? (
        <p className="dk-rec-note">
          {t(
            "This carrier has not been vetted. Vet it on its page to read its authority and safety.",
          )}
        </p>
      ) : intel.notFound ? (
        <p className="dk-rec-note dk-warn">
          {t("The FMCSA has no record of this carrier's DOT number.")}
        </p>
      ) : (
        <>
          <AuthorityPills stats={authorityStats(intel, labels, t)} />
          {meters.length > 0 ? (
            <BasicChart meters={meters} />
          ) : (
            <p className="dk-rec-note dk-ca-nobasics">
              {t("No CSA BASIC has been measured for this carrier.")}
            </p>
          )}
        </>
      )}
    </CardSection>
  );
}

/** What the carrier was paid, newest first, a page at a time. */
function Settlements({
  settlements,
  onOpen,
}: {
  settlements: PagedRows<CarrierSettlementRow>;
  onOpen: ((settlementId: string) => void) | undefined;
}) {
  const t = useT();
  return (
    <CardPagedList
      list={settlements}
      waitRows={2}
      emptyText={t("This carrier has not been settled yet.")}
      errorText={t("The settlements could not be read.")}
      nextErrorText={t("The next settlements could not be read.")}
      loadingLabel={t("Reading more settlements…")}
    >
      {(rows) => (
        <div className="dk-rec-list">
          {rows.map((row) => (
            <CardRecordRow
              key={row.id}
              to={recordPath("carrier_settlement", row.id)}
              onOpen={onOpen && (() => onOpen(row.id))}
            >
              <span className="dk-rec-main">
                <b>{row.settlementNumber}</b>
                <span className="dk-rec-sub">{settlementLine(row, t)}</span>
              </span>
              <span className="dk-rec-side">
                <span className="dk-ca-amt">{settlementAmount(row)}</span>
                <span
                  className={cn(
                    "dk-rec-sub",
                    row.status === "Paid" && "dk-ok",
                    row.status === "Voided" && "dk-off",
                  )}
                >
                  {choiceLabel(carrierSettlementStatusChoices, row.status)}
                </span>
              </span>
            </CardRecordRow>
          ))}
        </div>
      )}
    </CardPagedList>
  );
}
