import { recordPath } from "@/config/record-links";
import { useLaneScopeLabels } from "@/hooks/use-lane-scope-labels";
import { useRecordNames } from "@/hooks/use-record-names";
import { describeApiError } from "@/lib/api-error-message";
import { choiceLabel, ratePartyTypeChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { rateAgreementLanesQuery } from "@/lib/queries/rate-agreement";
import { agreementTypeLabel, agreementWindow } from "@/lib/rate-agreement";
import { keepPreviousData, useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { laneDisplayLabel, laneEndDisplay } from "@trenova/shared/lib/rate";
import { cn } from "@trenova/shared/lib/utils";
import type {
  RateAgreement,
  RateAgreementRule,
  RateAgreementVersion,
} from "@trenova/shared/types/rate";
import { useMemo, useState, type ReactNode } from "react";
import { Link } from "react-router";
import { CardActivity } from "../../record-card/card-activity";
import { useCardAccess } from "../../record-card/card-access";
import {
  CardBar,
  CardBarHint,
  CardFacts,
  CardHead,
  CardIcon,
  CardMeter,
  CardRecordRow,
  CardSection,
  CardSectionError,
  CardSectionWait,
  CardStatus,
  CardStepper,
  CardUnreadable,
  CardTimeline,
  RecordCard,
} from "../../record-card/record-card";
import { cardDay, cardMoment, money, neighborsIn } from "../../record-card/record-card-values";
import { useRecordActivity } from "../../record-card/record-activity";
import { CardMeterSkeleton } from "../../skeletons/record-card-skeleton";
import { useListedRecords, type RecordArtifactProps } from "../registry";
import {
  accessorialTerms,
  agreementReferences,
  agreementStage,
  breakLabel,
  fuelTerms,
  guardrails,
  lanePricing,
  laneTerms,
  pricingGroups,
} from "./rate-agreement-card";
import { ApprovalPath, TermPanel } from "./rate-agreement-term";
import { RATE_AGREEMENT_SECTIONS } from "./rate-agreement-sections";
import { DeskRateAgreementSkeleton } from "./rate-agreement-skeleton";

/**
 * The lanes the card reads at a time. An agreement can price tens of thousands,
 * so the card reads the most specific first and the next page on request.
 */
const LANE_PAGE = 8;
/** The weight breaks a lane lists before saying how many more it has. */
const BREAK_ROWS = 4;
/** The versions the card lists. */
const VERSION_ROWS = 4;
/** The ways of pricing the rate structure draws before folding the rest into one. */
export const PRICING_ROWS = 5;
/** The activity entries the card reads at a time. */
const ACTIVITY_ROWS = 5;

/**
 * What the card is given: the agreement, and when it was opened from a table
 * or from a card that lists it, the rows in the order shown, how to step to
 * another, and the way back.
 */
export type RateAgreementCardProps = {
  recordId: string;
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the agreement was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];
const NO_VERSIONS: readonly RateAgreementVersion[] = [];
const NO_LANES: readonly RateAgreementRule[] = [];

/**
 * One rate agreement as a pricing analyst or a biller reads it: whether it
 * prices freight now and how long it runs, the way it was approved, who it
 * prices, each lane with how it is priced and what bounds that price, the
 * charges and fuel it adds, the limits every price stays inside, and how its
 * terms have changed. It reads; amending lanes, reviewing and approving stay
 * on the agreement's page, one click away.
 *
 * The card steps through the rows it was opened from in the order shown.
 */
export function DeskRateAgreement({
  recordId,
  order = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: RateAgreementCardProps) {
  const t = useT();
  const query = useQuery({
    ...queries.rateAgreement.get(recordId),
    retry: false,
    placeholderData: keepPreviousData,
  });
  const agreement = query.data;

  if (!agreement) {
    return query.isError ? (
      <CardUnreadable
        kind={t("Rate agreement")}
        title={t("This rate agreement could not be read")}
        error={query.error}
        onBack={onBack}
        backLabel={backLabel}
      />
    ) : (
      <DeskRateAgreementSkeleton stepper={onSelect !== undefined && order.length > 1} />
    );
  }

  return (
    <AgreementCard
      key={agreement.id ?? recordId}
      agreement={agreement}
      recordId={agreement.id ?? recordId}
      order={order}
      onSelect={onSelect}
      onBack={onBack}
      backLabel={backLabel}
    />
  );
}

/**
 * The agreement as an artifact of its own, when the agent looked it up: the
 * same card, with nothing to step through and nowhere to go back to.
 */
export function DeskRateAgreementArtifact({ recordId }: RecordArtifactProps) {
  return <DeskRateAgreement recordId={recordId} />;
}

function AgreementCard({
  agreement,
  recordId,
  order,
  onSelect,
  onBack,
  backLabel,
}: Required<Pick<RateAgreementCardProps, "order" | "recordId">> &
  Omit<RateAgreementCardProps, "order" | "recordId"> & { agreement: RateAgreement }) {
  const t = useT();
  const access = useCardAccess(RATE_AGREEMENT_SECTIONS);
  const [now] = useState(() => Math.floor(Date.now() / 1000));
  const currency = agreement.currency;

  const { data: versions = NO_VERSIONS } = useQuery(queries.rateAgreement.versions(recordId));
  const pricing = useQuery({
    ...queries.rateAgreement.pricing(recordId),
    enabled: access.pricing,
    retry: false,
  });

  const laneSummary = useQuery({
    ...queries.rateAgreement.laneSummary(recordId),
    retry: false,
  });
  const activity = useRecordActivity({
    resourceId: recordId,
    pageSize: ACTIVITY_ROWS,
    enabled: access.activity,
  });

  const laneQuery = useInfiniteQuery({
    ...rateAgreementLanesQuery(recordId, LANE_PAGE),
    retry: false,
  });
  const lanes = useMemo(
    () => laneQuery.data?.pages.flatMap((page) => page.results) ?? NO_LANES,
    [laneQuery.data],
  );
  const firstLanes = laneQuery.data?.pages[0];
  const latestVersions = useMemo(
    () => [...versions].sort((a, b) => b.versionNumber - a.versionNumber).slice(0, VERSION_ROWS),
    [versions],
  );
  const refs = useMemo(
    () =>
      agreementReferences(
        agreement,
        lanes,
        latestVersions.map((version) => version.createdById),
      ),
    [agreement, lanes, latestVersions],
  );
  const templates = useRecordNames("FORMULA_TEMPLATE", refs.templates).names;
  const matrices = useRecordNames("RATE_MATRIX", refs.matrices).names;
  const charges = useRecordNames("ACCESSORIAL_CHARGE", refs.charges).names;
  const programs = useRecordNames("FUEL_SURCHARGE_PROGRAM", refs.programs).names;
  const customers = useRecordNames("CUSTOMER", refs.customers).names;
  const carriers = useRecordNames("CARRIER", refs.carriers).names;
  const users = useRecordNames("USER", refs.users).names;
  const resolvePlace = useLaneScopeLabels(lanes);

  const backToAgreement = t("Back to {0}", agreement.name);
  const listedCustomers = useListedRecords({
    entity: "customer",
    order: refs.customers,
    backLabel: backToAgreement,
  });
  const listedTemplates = useListedRecords({
    entity: "formula_template",
    order: refs.templates,
    backLabel: backToAgreement,
  });

  // A record opened from the card takes the card's place, and its way back
  // returns here.
  const opened = listedCustomers.opened ?? listedTemplates.opened;
  if (opened) {
    return opened;
  }

  const stage = agreementStage(agreement.status);
  const runs = agreementWindow(agreement, now, t);
  const neighbors = neighborsIn(order, recordId);
  const priced = access.pricing ? pricing.data : undefined;
  const laneCount = firstLanes?.count ?? null;
  const retired = firstLanes?.retired ?? 0;
  const agreementPath = recordPath("rate_agreement", recordId);

  return (
    <RecordCard className={cn("dk-ra", `dk-ra-${agreement.status.toLowerCase()}`)}>
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={[
          agreement.code,
          agreementTypeLabel(agreement.agreementType),
          t("{0} agreement", choiceLabel(ratePartyTypeChoices, agreement.partyType)),
          t("v{0}", agreement.currentVersionNumber),
        ]}
        stepper={
          onSelect &&
          order.length > 1 && (
            <CardStepper
              neighbors={neighbors}
              onStep={onSelect}
              previousLabel={t("Previous agreement")}
              nextLabel={t("Next agreement")}
            />
          )
        }
        title={agreement.name}
        subtitle={agreement.description.trim() || undefined}
        figure={
          laneCount === null ? (
            <>
              <b className="dk-ra-wait">—</b>
              <span>{t("lanes priced")}</span>
            </>
          ) : (
            <>
              <b key={laneCount}>{formatNumber(laneCount)}</b>
              <span>{t("{0, plural, one {lane priced} other {lanes priced}}", laneCount)}</span>
            </>
          )
        }
        meta={
          <>
            <CardStatus key={agreement.status} tone={stage.tone} mark={stage.mark}>
              {stage.label}
            </CardStatus>
            <CardFacts>
              {[
                agreement.status === "Active" ? (
                  <span key="runs" className={cn(runs.state && `dk-ra-${runs.state}`)}>
                    {runs.text}
                  </span>
                ) : (
                  ""
                ),
                priced && priced.loads > 0
                  ? t(
                      "{0, plural, one {Priced # load} other {Priced # loads}}, last {1}",
                      priced.loads,
                      cardDay(priced.lastRatedAt),
                    )
                  : priced
                    ? t("Has not priced a load yet")
                    : "",
                agreement.priority !== 0 ? t("Priority {0}", agreement.priority) : "",
                agreement.contractRef.trim() !== ""
                  ? t("Contract {0}", agreement.contractRef.trim())
                  : "",
              ]}
            </CardFacts>
          </>
        }
      />

      <CardSection title={t("Term")}>
        <TermPanel agreement={agreement} now={now} />
        <ApprovalPath agreement={agreement} users={users} />
      </CardSection>

      <CardSection title={agreement.partyType === "Carrier" ? t("Carrier") : t("Customer")}>
        <div className="dk-rec-list">
          {agreement.partyType === "Carrier" && agreement.carrierId ? (
            <CardRecordRow to={recordPath("carrier", agreement.carrierId)}>
              <PartyName name={carriers.get(agreement.carrierId)} />
            </CardRecordRow>
          ) : agreement.customerId ? (
            <PartyRow
              id={agreement.customerId}
              name={customers.get(agreement.customerId)}
              onOpen={listedCustomers.open}
            />
          ) : null}
          {agreement.billToCustomerId && agreement.billToCustomerId !== agreement.customerId && (
            <PartyRow
              id={agreement.billToCustomerId}
              name={customers.get(agreement.billToCustomerId)}
              note={t("Bill to")}
              onOpen={listedCustomers.open}
            />
          )}
        </div>
      </CardSection>

      <CardSection title={t("Rate structure")}>
        {laneSummary.isError ? (
          <CardSectionError>
            {describeApiError(laneSummary.error, t("How the lanes are priced could not be read."))}
          </CardSectionError>
        ) : !laneSummary.data ? (
          <div className="dk-ra-mix" aria-busy>
            <CardMeterSkeleton />
            <CardMeterSkeleton />
          </div>
        ) : laneSummary.data.count === 0 ? (
          <p className="dk-ra-note">
            {t("Once a lane prices freight, how each is priced shows here.")}
          </p>
        ) : (
          <div className="dk-ra-mix">
            {pricingGroups(laneSummary.data, { rows: PRICING_ROWS, currency }, t).map((group) => (
              <CardMeter
                key={group.key}
                label={group.label}
                reading={t("{0, plural, one {# lane} other {# lanes}}", group.lanes)}
                figure={group.range ?? undefined}
                share={group.share}
                state="ok"
              />
            ))}
          </div>
        )}
      </CardSection>

      <CardSection
        title={t("Lanes")}
        aside={
          laneCount !== null && (laneCount > lanes.length || retired > 0) ? (
            <span>
              {[
                laneCount > lanes.length
                  ? t("{0} of {1} shown", formatNumber(lanes.length), formatNumber(laneCount))
                  : "",
                retired > 0 ? t("{0, plural, one {# retired} other {# retired}}", retired) : "",
              ]
                .filter(Boolean)
                .join(" · ")}
            </span>
          ) : undefined
        }
      >
        {laneQuery.isError && lanes.length === 0 ? (
          <CardSectionError>
            {describeApiError(laneQuery.error, t("The lanes could not be read."))}
          </CardSectionError>
        ) : laneQuery.isPending ? (
          <CardSectionWait rows={3} />
        ) : lanes.length === 0 ? (
          <p className="dk-ra-note">{t("No lane prices freight on this agreement yet.")}</p>
        ) : (
          <>
            <div className="dk-ra-lanes">
              {lanes.map((lane, index) => (
                <LaneRow
                  key={lane.id ?? index}
                  lane={lane}
                  index={index}
                  currency={lane.currency || currency}
                  origin={laneEndDisplay(lane, "origin", resolvePlace)}
                  destination={laneEndDisplay(lane, "destination", resolvePlace)}
                  pricedBy={
                    <LanePrice
                      lane={lane}
                      templates={templates}
                      matrices={matrices}
                      onOpenTemplate={listedTemplates.open}
                    />
                  }
                />
              ))}
            </div>
            {laneQuery.hasNextPage && laneCount !== null && (
              <Button
                variant="quiet"
                size="xs"
                className="dk-ra-more-lanes"
                disabled={laneQuery.isFetchingNextPage}
                onClick={() => void laneQuery.fetchNextPage()}
              >
                {laneQuery.isFetchingNextPage
                  ? t("Reading more lanes…")
                  : t(
                      "{0, plural, one {Show # more lane} other {Show # more lanes}}",
                      Math.min(LANE_PAGE, laneCount - lanes.length),
                    )}
              </Button>
            )}
          </>
        )}
      </CardSection>

      {(agreement.accessorials.length > 0 || agreement.fuelBinding) && (
        <CardSection title={t("Charges")}>
          <div className="dk-ra-charges">
            {agreement.accessorials.map((accessorial, index) => (
              <div
                key={accessorial.id ?? `${accessorial.accessorialChargeId}-${index}`}
                className={cn("dk-ra-charge", accessorial.waived && "dk-off")}
              >
                <b>{charges.get(accessorial.accessorialChargeId) ?? "…"}</b>
                <span>{accessorialTerms(accessorial, currency, t).join(" · ")}</span>
              </div>
            ))}
            {agreement.fuelBinding && (
              <div className={cn("dk-ra-charge", agreement.fuelBinding.waived && "dk-off")}>
                <b>
                  {t(
                    "Fuel · {0}",
                    programs.get(agreement.fuelBinding.fuelSurchargeProgramId) ?? "…",
                  )}
                </b>
                <span>{fuelTerms(agreement.fuelBinding, currency, t).join(" · ")}</span>
              </div>
            )}
          </div>
        </CardSection>
      )}

      <CardSection title={t("Limits")}>
        <dl className="dk-ra-limits">
          {guardrails(agreement, t).map((row) => (
            <div key={row.key}>
              <dt>{row.label}</dt>
              <dd>{row.value}</dd>
            </div>
          ))}
        </dl>
      </CardSection>

      {access.activity && (
        <CardActivity
          activity={activity}
          emptyText={t("Nothing has been done to this agreement yet.")}
        />
      )}

      {latestVersions.length > 0 && (
        <CardSection title={t("Versions")}>
          <CardTimeline
            entries={[...latestVersions].reverse().map((version) => ({
              id: version.id ?? String(version.versionNumber),
              text: t("v{0} · {1}", version.versionNumber, version.changeMessage || t("No note")),
              meta: [
                version.createdById ? users.get(version.createdById) : "",
                cardMoment(version.createdAt),
              ]
                .filter(Boolean)
                .join(" · "),
            }))}
          />
        </CardSection>
      )}

      <CardBar>
        <CardBarHint>{t("Amend lanes, review and approve it on the agreement's page")}</CardBarHint>
        <Link className="dk-ax-btn dk-ink" to={agreementPath}>
          <CardIcon name="ext" size={13} />
          {t("Open agreement")}
        </Link>
      </CardBar>
    </RecordCard>
  );
}

/** The customer or carrier on a party row, while its name is read. */
function PartyName({ name, note }: { name: string | undefined; note?: string }) {
  return (
    <span className="dk-ra-party">
      <b>{name ?? "…"}</b>
      {note && <span>{note}</span>}
    </span>
  );
}

/** A customer the agreement names, opened in the card's place when the person may read it. */
function PartyRow({
  id,
  name,
  note,
  onOpen,
}: {
  id: string;
  name: string | undefined;
  note?: string;
  onOpen: ((customerId: string) => void) | undefined;
}) {
  return (
    <CardRecordRow to={recordPath("customer", id)} onOpen={onOpen && (() => onOpen(id))}>
      <PartyName name={name} note={note} />
    </CardRecordRow>
  );
}

/** How a lane is priced: the rate on its formula template, which opens in place, or its rate table. */
function LanePrice({
  lane,
  templates,
  matrices,
  onOpenTemplate,
}: {
  lane: RateAgreementRule;
  templates: ReadonlyMap<string, string>;
  matrices: ReadonlyMap<string, string>;
  onOpenTemplate: ((templateId: string) => void) | undefined;
}) {
  const t = useT();
  const pricing = lanePricing(lane);
  if (pricing.by === "matrix") {
    return <span>{t("Rate table {0}", matrices.get(pricing.matrixId) ?? "…")}</span>;
  }
  if (pricing.by === "none") {
    return <span className="dk-ra-off">{t("Not priced")}</span>;
  }
  const name = templates.get(pricing.templateId) ?? "…";
  return (
    <span>
      {pricing.rate !== null && <b>{pricing.rate}</b>}
      {onOpenTemplate ? (
        <Button
          variant="bare"
          size="bare"
          className="dk-ra-tpl"
          onClick={() => onOpenTemplate(pricing.templateId)}
        >
          {name}
        </Button>
      ) : (
        <span className="dk-ra-tpl">{name}</span>
      )}
    </span>
  );
}

/** One lane: its places, how it is priced, what bounds it, and its first weight breaks. */
function LaneRow({
  lane,
  index,
  currency,
  origin,
  destination,
  pricedBy,
}: {
  lane: RateAgreementRule;
  index: number;
  currency: string;
  origin: string;
  destination: string;
  pricedBy: ReactNode;
}) {
  const t = useT();
  const terms = laneTerms(lane, currency, t);
  const breaks = lane.breaks.slice(0, BREAK_ROWS);
  return (
    <div className="dk-ra-lane">
      <div className="dk-ra-lane-h">
        <span className="dk-ra-places">
          {lane.label && <em>{laneDisplayLabel(lane, index)}</em>}
          <b>
            {origin}
            <i aria-hidden>{lane.direction === "Bidirectional" ? " ⇄ " : " → "}</i>
            {destination}
          </b>
        </span>
        <span className="dk-ra-price">{pricedBy}</span>
      </div>
      {terms.length > 0 && (
        <ul className="dk-ra-tags">
          {terms.map((term) => (
            <li key={term}>{term}</li>
          ))}
        </ul>
      )}
      {breaks.length > 0 && (
        <dl className="dk-ra-breaks">
          {breaks.map((weightBreak, breakIndex) => (
            <div key={weightBreak.id ?? breakIndex}>
              <dt>{breakLabel(weightBreak, t)}</dt>
              <dd>{money(weightBreak.rate, currency)}</dd>
            </div>
          ))}
          {lane.breaks.length > breaks.length && (
            <div className="dk-ra-more">
              <dt>{t("{0} more", lane.breaks.length - breaks.length)}</dt>
            </div>
          )}
        </dl>
      )}
    </div>
  );
}
