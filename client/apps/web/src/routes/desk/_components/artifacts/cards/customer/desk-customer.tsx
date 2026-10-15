import { recordPath } from "@/config/record-links";
import { customerStatementPath } from "@/lib/ar-links";
import { describeApiError } from "@/lib/api-error-message";
import { cityStateLabel } from "@/lib/place";
import type { ARCustomerProfile } from "@/lib/graphql/accounts-receivable";
import type {
  DeskCustomer as DeskCustomerRecord,
  DeskCustomerBilling,
  DeskCustomerRateAgreement,
} from "@/lib/graphql/customer-card";
import { ar } from "@/lib/queries/ar";
import { customer as customerQueries } from "@/lib/queries/customer";
import { agreementTypeLabel, agreementWindow, orderedAgreements } from "@/lib/rate-agreement";
import { useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, formatPercent } from "@trenova/shared/lib/utils";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import { useCardAccess } from "../../record-card/card-access";
import { CardShipments } from "../../record-card/card-shipments";
import { useRecordShipments } from "../../record-card/record-shipments";
import {
  CardBar,
  CardBarHint,
  CardFacts,
  CardHead,
  CardIcon,
  CardLink,
  CardMeter,
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
import {
  cardDay,
  cardMonth,
  minorMoney,
  money,
  neighborsIn,
} from "../../record-card/record-card-values";
import { useListedRecords, type RecordArtifactProps } from "../registry";
import {
  accountStanding,
  agingStretches,
  billingRules,
  billingTerms,
  collectionBars,
  creditUse,
  customerAddress,
  invoiceNeeds,
  payPace,
  type CreditUse,
} from "./customer-card";
import { CUSTOMER_SECTIONS } from "./customer-sections";
import { DeskCustomerSkeleton } from "./customer-skeleton";

/** The shipments the card reads at a time; the next page is read on request. */
const SHIPMENT_ROWS = 5;
/** The rate agreements the card lists. */
const AGREEMENT_ROWS = 4;
/** The months of payments the card charts. */
const COLLECTION_MONTHS = 6;

/**
 * What the card is given: the customer, and when it was opened from a table,
 * the table's rows in the order shown, how to step to another, and the way
 * back.
 */
export type CustomerCardProps = {
  recordId: string;
  order?: readonly string[];
  onSelect?: (recordId: string) => void;
  onBack?: () => void;
  /** What the way back says; the table, unless the customer was opened from another card. */
  backLabel?: string;
};

const NO_ROWS: readonly string[] = [];

/**
 * One customer as an account manager or a biller reads it, opened from a
 * table's row: who they are and whether their account is in good standing,
 * what they owe and how far past due it has aged, how they pay against their
 * terms, how they are billed and what each invoice needs, the freight they
 * have moved lately and the agreements that price it. A section drawn from
 * another kind of record shows only to someone who may read that kind.
 * Editing stays on the customer's page, one click away.
 *
 * The card steps through the table's rows in the order the table shows them.
 * A step draws the next customer's skeleton until it is read, never the one it left.
 */
export function DeskCustomer({
  recordId,
  order = NO_ROWS,
  onSelect,
  onBack,
  backLabel,
}: CustomerCardProps) {
  const t = useT();
  const query = useQuery({ ...customerQueries.card(recordId), retry: false });
  const customer = query.data;

  if (!customer) {
    return query.isError ? (
      <CardUnreadable
        kind={t("Customer")}
        title={t("This customer could not be read")}
        error={query.error}
        onBack={onBack}
        backLabel={backLabel}
      />
    ) : (
      <DeskCustomerSkeleton stepper={onSelect !== undefined && order.length > 1} />
    );
  }

  return (
    <CustomerCard
      key={customer.id}
      customer={customer}
      order={order}
      onSelect={onSelect}
      onBack={onBack}
      backLabel={backLabel}
    />
  );
}

/**
 * The customer as an artifact of its own, when the agent looked it up: the
 * same card, with nothing to step through and nowhere to go back to.
 */
export function DeskCustomerArtifact({ recordId }: RecordArtifactProps) {
  return <DeskCustomer recordId={recordId} />;
}

function CustomerCard({
  customer,
  order,
  onSelect,
  onBack,
  backLabel,
}: Required<Pick<CustomerCardProps, "order">> &
  Omit<CustomerCardProps, "order" | "recordId"> & { customer: DeskCustomerRecord }) {
  const t = useT();
  const access = useCardAccess(CUSTOMER_SECTIONS);
  const [now] = useState(() => Math.floor(Date.now() / 1000));
  const id = customer.id;
  const billing = customer.billingProfile ?? null;

  const receivables = useQuery({
    ...ar.customerProfile(id),
    enabled: access.receivables,
    retry: false,
  });
  const shipments = useRecordShipments({
    scope: { kind: "customer", id },
    pageSize: SHIPMENT_ROWS,
    enabled: access.shipments,
  });
  const agreements = useQuery({
    ...customerQueries.rateAgreements(id, AGREEMENT_ROWS),
    enabled: access.agreements,
    retry: false,
  });

  const agreementRows = useMemo(
    () => orderedAgreements(agreements.data?.rows ?? []),
    [agreements.data],
  );
  const agreementIds = useMemo(() => agreementRows.map((row) => row.id), [agreementRows]);
  const backToCustomer = t("Back to {0}", customer.name);
  const listedShipments = useListedRecords({
    entity: "shipment",
    order: shipments.ids,
    backLabel: backToCustomer,
  });
  const listedAgreements = useListedRecords({
    entity: "rate_agreement",
    order: agreementIds,
    backLabel: backToCustomer,
  });

  const profile = access.receivables ? receivables.data : undefined;
  const snapshot = profile?.snapshot;
  const standing = accountStanding(customer.status, billing?.creditStatus, t);
  const neighbors = neighborsIn(order, id);
  const place = cityStateLabel(customer.city, customer.state?.abbreviation);

  // A listed record opened from the card takes the card's place, and its way
  // back returns here.
  const opened = listedShipments.opened ?? listedAgreements.opened;
  if (opened) {
    return opened;
  }

  return (
    <RecordCard className={cn("dk-cu", `dk-cu-${standing.tone}`)}>
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={[
          customer.code,
          ...(place ? [place] : []),
          ...(customer.mcNumber ? [t("MC {0}", customer.mcNumber)] : []),
        ]}
        stepper={
          onSelect &&
          order.length > 1 && (
            <CardStepper
              neighbors={neighbors}
              onStep={onSelect}
              previousLabel={t("Previous customer")}
              nextLabel={t("Next customer")}
            />
          )
        }
        title={customer.name}
        subtitle={customerAddress(customer) || undefined}
        figure={
          access.receivables ? (
            <OpenFigure profile={profile} pending={receivables.isPending} />
          ) : null
        }
        meta={
          <>
            <CardStatus key={standing.label} tone={standing.tone} mark={standing.mark}>
              {standing.label}
            </CardStatus>
            <CardFacts>
              {[
                snapshot && snapshot.overdueMinor > 0
                  ? t("{0} past due", minorMoney(snapshot.overdueMinor))
                  : "",
                snapshot && snapshot.oldestDaysPastDue > 0
                  ? t(
                      "Oldest {0, plural, one {# day} other {# days}} late",
                      snapshot.oldestDaysPastDue,
                    )
                  : "",
                t("Customer since {0}", cardMonth(customer.createdAt)),
              ]}
            </CardFacts>
          </>
        }
      />

      {billing?.creditStatus === "Hold" && billing.creditHoldReason && (
        <div className="dk-cu-hold">
          <div className="dk-ax-warn">
            <CardIcon name="pause" size={13} />
            <span>{billing.creditHoldReason}</span>
          </div>
        </div>
      )}

      {access.receivables && (
        <CardSection
          title={t("Receivables")}
          aside={<CardLink to={customerStatementPath(id)}>{t("Statement")}</CardLink>}
        >
          <Receivables
            profile={profile}
            term={billing?.paymentTerm}
            now={now}
            pending={receivables.isPending}
            error={receivables.error}
          />
        </CardSection>
      )}

      <CardSection title={t("Billing")}>
        {billing ? (
          <Billing billing={billing} snapshot={snapshot} />
        ) : (
          <p className="dk-cu-note">{t("This customer has no billing profile yet.")}</p>
        )}
      </CardSection>

      {access.shipments && (
        <CardShipments
          title={t("Recent shipments")}
          shipments={shipments}
          onOpen={listedShipments.open}
          emptyText={t("No shipments yet.")}
        />
      )}

      {access.agreements && (
        <CardSection
          title={t("Rate agreements")}
          aside={
            agreements.data && agreements.data.total > agreements.data.rows.length ? (
              <span>{t("{0} in all", agreements.data.total)}</span>
            ) : undefined
          }
        >
          <RateAgreements
            rows={agreements.data ? agreementRows : undefined}
            onOpen={listedAgreements.open}
            now={now}
            pending={agreements.isPending}
            error={agreements.error}
          />
        </CardSection>
      )}

      <CardBar>
        <CardBarHint>{t("Change terms, credit and contacts on the customer's page")}</CardBarHint>
        <Link className="dk-ax-btn dk-ink" to={recordPath("customer", id)}>
          <CardIcon name="ext" size={13} />
          {t("Open customer")}
        </Link>
      </CardBar>
    </RecordCard>
  );
}

/** What the customer owes now, at the head of the card. */
function OpenFigure({
  profile,
  pending,
}: {
  profile: ARCustomerProfile | undefined;
  pending: boolean;
}) {
  const t = useT();
  const snapshot = profile?.snapshot;
  if (pending || !snapshot) {
    return (
      <>
        <b className="dk-cu-wait">—</b>
        <span>{t("Open balance")}</span>
      </>
    );
  }
  return (
    <>
      <b key={snapshot.totalOpenMinor}>{minorMoney(snapshot.totalOpenMinor)}</b>
      <span>
        {snapshot.openInvoiceCount > 0
          ? t(
              "{0, plural, one {open on # invoice} other {open on # invoices}}",
              snapshot.openInvoiceCount,
            )
          : t("nothing open")}
      </span>
    </>
  );
}

/**
 * What the customer owes and how it has aged, how they pay against their
 * terms, and what they have paid month by month.
 */
function Receivables({
  profile,
  term,
  now,
  pending,
  error,
}: {
  profile: ARCustomerProfile | undefined;
  term: DeskCustomerBilling["paymentTerm"] | undefined;
  now: number;
  pending: boolean;
  error: unknown;
}) {
  const t = useT();
  if (error && !profile) {
    return (
      <CardSectionError>
        {describeApiError(error, t("The receivables could not be read."))}
      </CardSectionError>
    );
  }
  if (pending || !profile) {
    return <CardSectionWait rows={3} />;
  }

  const { snapshot } = profile;
  const stretches = agingStretches(snapshot.buckets, t);
  const pace = payPace(snapshot.avgDaysToPay, term);
  const bars = collectionBars(snapshot.monthlyCollections, now, COLLECTION_MONTHS);
  const collected = bars.some((bar) => bar.minor > 0);

  return (
    <>
      {snapshot.totalOpenMinor > 0 ? (
        <>
          <div
            className="dk-cu-age"
            role="img"
            aria-label={stretches
              .filter((stretch) => stretch.minor > 0)
              .map((stretch) => `${stretch.label}: ${minorMoney(stretch.minor)}`)
              .join(", ")}
          >
            {stretches
              .filter((stretch) => stretch.minor > 0)
              .map((stretch) => (
                <i
                  key={stretch.key}
                  style={{ flexGrow: stretch.share, background: stretch.color }}
                />
              ))}
          </div>
          <dl className="dk-cu-buckets">
            {stretches.map((stretch) => (
              <div key={stretch.key} className={cn(stretch.minor === 0 && "dk-off")}>
                <dt>
                  <i style={{ background: stretch.color }} />
                  {stretch.label}
                </dt>
                <dd>{minorMoney(stretch.minor)}</dd>
              </div>
            ))}
          </dl>
        </>
      ) : (
        <p className="dk-cu-note">{t("Nothing is open. Every invoice is paid.")}</p>
      )}

      <CardStats
        stats={[
          {
            key: "pace",
            label: t("Pays in"),
            value:
              snapshot.avgDaysToPay > 0
                ? t("{0, plural, one {# day} other {# days}}", Math.round(snapshot.avgDaysToPay))
                : "—",
            state: pace ?? undefined,
          },
          {
            key: "dso",
            label: t("DSO"),
            value:
              profile.dsoDays > 0
                ? t("{0, plural, one {# day} other {# days}}", Math.round(profile.dsoDays))
                : "—",
          },
          {
            key: "last",
            label: t("Last payment"),
            value:
              snapshot.lastPaymentDate > 0
                ? t(
                    "{0} on {1}",
                    minorMoney(snapshot.lastPaymentMinor),
                    cardDay(snapshot.lastPaymentDate),
                  )
                : t("None yet"),
          },
          {
            key: "billed",
            label: t("Billed, 91 days"),
            value: minorMoney(snapshot.billedTrailing91Minor),
          },
        ]}
      />

      {snapshot.unappliedCashMinor > 0 && (
        <p className="dk-cu-note">
          {t(
            "{0} received and not yet applied to an invoice.",
            minorMoney(snapshot.unappliedCashMinor),
          )}
        </p>
      )}

      {collected && (
        <div className="dk-cu-paid">
          <span className="dk-cu-paid-t">{t("Collected by month")}</span>
          <ol aria-label={t("Collected by month")}>
            {bars.map((bar) => (
              <li key={bar.monthStart} title={`${bar.label}: ${minorMoney(bar.minor)}`}>
                <i style={{ height: `${Math.max(bar.height * 100, bar.minor > 0 ? 6 : 2)}%` }} />
                <span>{bar.label}</span>
              </li>
            ))}
          </ol>
        </div>
      )}
    </>
  );
}

/**
 * How the customer is billed: their credit against its limit, their terms,
 * what every invoice needs before it can go, and the rules that change it.
 */
function Billing({
  billing,
  snapshot,
}: {
  billing: DeskCustomerBilling;
  snapshot: ARCustomerProfile["snapshot"] | undefined;
}) {
  const t = useT();
  // Receivables count what is open now; the profile's balance is what it was
  // when last posted, so it stands in only for someone without receivables.
  const used = snapshot ? snapshot.totalOpenMinor / 100 : Number(billing.creditBalance ?? 0);
  const limit = snapshot
    ? snapshot.hasCreditLimit
      ? snapshot.creditLimitMinor / 100
      : null
    : billing.creditLimit === null || billing.creditLimit === undefined
      ? null
      : Number(billing.creditLimit);
  const credit = creditUse(used, limit);
  const needs = invoiceNeeds(billing, t);
  const rules = billingRules(billing, t);

  return (
    <>
      {credit && limit !== null ? (
        <CreditMeter credit={credit} used={used} limit={limit} currency={billing.billingCurrency} />
      ) : (
        <p className="dk-cu-note">{t("No credit limit is set.")}</p>
      )}

      <CardStats className="dk-cu-terms" stats={billingTerms(billing, t)} />

      <div className="dk-cu-needs">
        <span>{t("Every invoice needs")}</span>
        {needs.length > 0 ? (
          <ul>
            {needs.map((need) => (
              <li key={need}>{need}</li>
            ))}
          </ul>
        ) : (
          <em>{t("Nothing beyond the shipment")}</em>
        )}
      </div>

      {rules.length > 0 && (
        <p className="dk-cu-rules">
          <CardFacts>{rules}</CardFacts>
        </p>
      )}

      {billing.billingNotes.trim() !== "" && (
        <blockquote className="dk-cu-memo">{billing.billingNotes}</blockquote>
      )}
    </>
  );
}

/** Credit used against the limit, as a meter that turns as the limit nears. */
function CreditMeter({
  credit,
  used,
  limit,
  currency,
}: {
  credit: CreditUse;
  used: number;
  limit: number;
  currency: string;
}) {
  const t = useT();
  return (
    <CardMeter
      label={t("Credit used")}
      reading={t("{0} of {1}", money(used, currency), money(limit, currency))}
      figure={formatPercent(Math.round(credit.share * 100), 0)}
      share={credit.share}
      state={credit.state}
    />
  );
}

/** The agreements that price the customer's freight, the ones in force first. */
function RateAgreements({
  rows,
  onOpen,
  now,
  pending,
  error,
}: {
  /** The agreements, those in force first. */
  rows: DeskCustomerRateAgreement[] | undefined;
  /** Opens an agreement in the card's place; absent, each row links to the agreement's page. */
  onOpen: ((agreementId: string) => void) | undefined;
  now: number;
  pending: boolean;
  error: unknown;
}) {
  const t = useT();
  if (error && !rows) {
    return (
      <CardSectionError>
        {describeApiError(error, t("The rate agreements could not be read."))}
      </CardSectionError>
    );
  }
  if (pending || !rows) {
    return <CardSectionWait rows={2} />;
  }
  if (rows.length === 0) {
    return (
      <p className="dk-cu-note">
        {t("No rate agreement. Freight is priced by its own rating method.")}
      </p>
    );
  }
  return (
    <div className="dk-rec-list">
      {rows.map((agreement) => {
        const runs = agreementWindow(agreement, now, t);
        return (
          <CardRecordRow
            key={agreement.id}
            to={recordPath("rate_agreement", agreement.id)}
            onOpen={onOpen && (() => onOpen(agreement.id))}
          >
            <span className="dk-rec-main">
              <b>{agreement.name}</b>
              <span className="dk-rec-sub">
                {[agreement.code, agreementTypeLabel(agreement.agreementType)].join(" · ")}
              </span>
            </span>
            <span className={cn("dk-cu-window", runs.state && `dk-${runs.state}`)}>
              {runs.text}
            </span>
          </CardRecordRow>
        );
      })}
    </div>
  );
}
