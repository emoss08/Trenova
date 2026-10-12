import type { CarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import { sortBasicMeasures } from "@/lib/carrier-intelligence";
import {
  carrierInsurancePolicyTypeChoices,
  carrierIntelRiskLevelChoices,
  carrierPaymentMethodChoices,
  carrierSafetyRatingChoices,
  carrierTypeChoices,
  choiceLabel,
} from "@/lib/choices";
import type {
  CarrierEligibility,
  CarrierIntel,
  CarrierSettlementRow,
  DeskCarrier,
} from "@/lib/graphql/carrier-card";
import { formatNumber } from "@trenova/shared/i18n/format";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { csaBasicLabel } from "@trenova/shared/lib/csa";
import { worstFirst, type DueCheck } from "../../record-card/card-due";
import type { CardCheckState, CardFact, CardStat, CardTone } from "../../record-card/record-card";
import {
  cardDate,
  cardDay,
  dueReading,
  minorMoney,
  money,
} from "../../record-card/record-card-values";

/** A policy expiring inside this many days is worth a look, as the server's own checks read it. */
export const INSURANCE_NOTICE_DAYS = 30;
/** A BASIC within this share of its threshold reads as close to an alert. */
const BASIC_CLOSE_SHARE = 0.8;

/** Whether the carrier can be tendered work, as the pill at the card's head says it. */
export function complianceStanding(
  carrier: Pick<DeskCarrier, "status" | "complianceStatus">,
  t: TranslateFn,
): { label: string; tone: CardTone; mark: "dot" | "check" | "pause" } {
  if (carrier.status !== "Active") {
    return { label: t("Inactive"), tone: "muted", mark: "dot" };
  }
  switch (carrier.complianceStatus) {
    case "Qualified":
      return { label: t("Qualified"), tone: "success", mark: "check" };
    case "Pending":
      return { label: t("Qualification pending"), tone: "progress", mark: "dot" };
    case "Disqualified":
      return { label: t("Disqualified"), tone: "warn", mark: "pause" };
    case "Expired":
      return { label: t("Qualification expired"), tone: "warn", mark: "pause" };
  }
}

/** An MC docket number without the "MC" a carrier may have typed into it. */
export function docketNumber(mcNumber: string): string {
  return mcNumber.trim().replace(/^MC[\s#:-]*/iu, "");
}

/** The identifiers a dispatcher reads a carrier by: its code, DOT, MC and SCAC. */
export function carrierKicker(
  carrier: Pick<DeskCarrier, "code" | "dotNumber" | "mcNumber" | "scac">,
  t: TranslateFn,
): string[] {
  return [
    carrier.code,
    carrier.dotNumber?.trim() ? t("DOT {0}", carrier.dotNumber.trim()) : "",
    carrier.mcNumber?.trim() ? t("MC {0}", docketNumber(carrier.mcNumber)) : "",
    carrier.scac?.trim() ? t("SCAC {0}", carrier.scac.trim()) : "",
  ].filter(Boolean);
}

/** The line under the carrier's name: the name it trades under, or what kind of carrier and where. */
export function carrierSubtitle(
  carrier: Pick<DeskCarrier, "dbaName" | "carrierType" | "city"> & {
    state?: { abbreviation: string } | null;
  },
  t: TranslateFn,
): string {
  const dba = carrier.dbaName?.trim();
  if (dba) return t("Doing business as {0}", dba);
  const home = [carrier.city?.trim() ?? "", carrier.state?.abbreviation ?? ""]
    .filter(Boolean)
    .join(", ");
  return [t("{0} carrier", choiceLabel(carrierTypeChoices, carrier.carrierType)), home]
    .filter(Boolean)
    .join(" · ");
}

/** The risk level intelligence last gave the carrier, and whether it reads as a concern. */
export function riskReading(
  level: string | null | undefined,
): { label: string; state: CardCheckState } | null {
  if (!level || level === "Unknown") return null;
  return {
    label: choiceLabel(carrierIntelRiskLevelChoices, level as never),
    state: level === "High" || level === "VeryHigh" ? "bad" : level === "Elevated" ? "warn" : "ok",
  };
}

/** Whether the carrier may be tendered a load, in one word, from the dispatch preview. */
export function eligibilityVerdict(
  eligibility: CarrierEligibility,
  t: TranslateFn,
): { label: string; state: CardCheckState } {
  if (eligibility.blockers.length > 0) return { label: t("Blocked"), state: "bad" };
  if (eligibility.warnings.length > 0) return { label: t("Needs an override"), state: "warn" };
  return { label: t("Clear to tender"), state: "ok" };
}

/**
 * What stands in the way of tendering to the carrier, worst first: what blocks
 * it, what a dispatcher must override, and what is only worth knowing. Each
 * says whether it comes from the carrier's own record or from intelligence.
 */
export function eligibilityChecks(eligibility: CarrierEligibility, t: TranslateFn): DueCheck[] {
  return worstFirst(
    eligibility.findings.map((finding, index) => {
      const source =
        finding.source === "Intelligence" ? t("From intelligence") : t("On the carrier's record");
      return {
        key: `${finding.code}-${index}`,
        state:
          finding.severity === "Blocker" ? "bad" : finding.severity === "Warning" ? "warn" : "info",
        title: finding.message,
        detail: finding.requiresOverride ? t("{0} · Needs an override", source) : source,
      };
    }),
  );
}

/**
 * The insurance the carrier keeps on file, worst first: each policy by its
 * expiry, read against the notice the server's own checks give, and one not
 * yet verified marked for a look.
 */
type InsurancePolicy = NonNullable<DeskCarrier["insurancePolicies"]>[number];

/** One policy read against now: its kind, how it stands, and how its title says so. */
function policyReading(policy: InsurancePolicy, now: number, t: TranslateFn) {
  const kind = choiceLabel(carrierInsurancePolicyTypeChoices, policy.policyType);
  const due = dueReading(policy.expirationDate, now, INSURANCE_NOTICE_DAYS);
  const unverified = !policy.isVerified && due.state === "ok";
  return {
    kind,
    due,
    unverified,
    state: (unverified ? "warn" : due.state) as CardCheckState,
    title:
      due.state === "bad"
        ? t("{0} expired", kind)
        : due.state === "warn"
          ? t("{0} expires in {1, plural, one {# day} other {# days}}", kind, due.days)
          : kind,
  };
}

/** One policy as a coverage tile draws it: its cover large, who writes it and until when. */
export type CoverageTile = {
  key: string;
  title: string;
  amount: string;
  insurer: string;
  until: string;
  state: CardCheckState;
  unverified: boolean;
};

/** The carrier's insurance as coverage tiles, worst first, each read as `insuranceChecks` reads it. */
export function coverageTiles(
  policies: readonly InsurancePolicy[],
  now: number,
  t: TranslateFn,
): CoverageTile[] {
  const rank: Record<CardCheckState, number> = { bad: 0, warn: 1, info: 2, ok: 3 };
  return policies
    .map((policy, index) => {
      const reading = policyReading(policy, now, t);
      return {
        index,
        tile: {
          key: policy.id,
          title: reading.title,
          amount: money(policy.coverageAmount),
          insurer: policy.providerName.trim(),
          until:
            reading.due.state === "bad"
              ? t("Since {0}", cardDate(policy.expirationDate))
              : t("Through {0}", cardDate(policy.expirationDate)),
          state: reading.state,
          unverified: reading.unverified,
        },
      };
    })
    .sort((a, b) => rank[a.tile.state] - rank[b.tile.state] || a.index - b.index)
    .map(({ tile }) => tile);
}

export function insuranceChecks(
  policies: NonNullable<DeskCarrier["insurancePolicies"]>,
  now: number,
  t: TranslateFn,
): DueCheck[] {
  return worstFirst(
    policies.map((policy) => {
      const { due, unverified, state, title } = policyReading(policy, now, t);
      return {
        key: policy.id,
        state,
        title,
        detail: [
          policy.providerName.trim(),
          money(policy.coverageAmount),
          due.state === "bad"
            ? t("Since {0}", cardDate(policy.expirationDate))
            : t("Through {0}", cardDate(policy.expirationDate)),
          unverified ? t("Not verified") : "",
        ]
          .filter(Boolean)
          .join(" · "),
      };
    }),
  );
}

type AuthorityGrant = NonNullable<
  NonNullable<NonNullable<CarrierIntel["profile"]["authority"]>["common"]>
>;

function grantStat(
  key: string,
  label: string,
  grant: AuthorityGrant | null | undefined,
  labels: CarrierIntelLabels,
  t: TranslateFn,
): CardStat {
  if (!grant) return { key, label, value: t("None") };
  const status = labels.authorityStatus[grant.status];
  return {
    key,
    label,
    value: grant.revocationPending
      ? t("{0}, revocation pending", status)
      : grant.pending
        ? t("{0}, pending", status)
        : status,
    state:
      grant.status === "Revoked" || grant.revocationPending ? "bad" : grant.pending ? "warn" : "ok",
  };
}

/** The carrier's operating authority and safety rating, as intelligence last read them. */
export function authorityStats(
  intel: CarrierIntel,
  labels: CarrierIntelLabels,
  t: TranslateFn,
): CardStat[] {
  const authority = intel.profile.authority ?? null;
  const safety = intel.profile.safety ?? null;
  const rating = safety?.rating?.trim();
  return [
    grantStat("common", t("Common authority"), authority?.common, labels, t),
    grantStat("contract", t("Contract authority"), authority?.contract, labels, t),
    grantStat("broker", t("Broker authority"), authority?.broker, labels, t),
    safety?.outOfServiceOrder
      ? { key: "safety", label: t("Safety"), value: t("Out-of-service order"), state: "bad" }
      : {
          key: "safety",
          label: t("Safety rating"),
          value: rating ? choiceLabel(carrierSafetyRatingChoices, rating as never) : t("Not rated"),
          state: rating === "Unsatisfactory" ? "bad" : rating === "Conditional" ? "warn" : "ok",
        },
  ];
}

/** One CSA BASIC as the card's meter reads it. */
export type BasicMeter = {
  key: string;
  label: string;
  reading: string;
  share: number;
  /** Where the threshold that sets an alert sits, 0 to 1; null when the BASIC has none. */
  thresholdShare: number | null;
  state: CardCheckState;
};

/**
 * The CSA BASICs in the FMCSA's order, each its percentile against the
 * threshold that sets an alert: past it or alerted is bad, close to it a
 * warning. A BASIC with too little data to rank reads as such and draws empty.
 */
export function basicMeters(
  basics: NonNullable<CarrierIntel["profile"]["basics"]>,
  t: TranslateFn,
): BasicMeter[] {
  return sortBasicMeasures(basics).map((basic) => {
    const percentile = basic.percentile ?? null;
    const threshold = basic.threshold ?? null;
    const closeAt = threshold !== null ? threshold * BASIC_CLOSE_SHARE : null;
    return {
      key: basic.basic,
      label: csaBasicLabel(basic.basic),
      reading:
        percentile === null
          ? t("Not enough data")
          : threshold !== null
            ? t("{0} of {1}", formatNumber(Math.round(percentile)), formatNumber(threshold))
            : formatNumber(Math.round(percentile)),
      share: percentile === null ? 0 : percentile / 100,
      thresholdShare: threshold === null ? null : threshold / 100,
      state:
        basic.alert || (percentile !== null && threshold !== null && percentile >= threshold)
          ? "bad"
          : percentile !== null && closeAt !== null && percentile >= closeAt
            ? "warn"
            : "ok",
    };
  });
}

/** A settlement as one row reads it: its pay period, when it pays and how many loads. */
export function settlementLine(row: CarrierSettlementRow, t: TranslateFn): string {
  return [
    t("{0} to {1}", cardDay(row.periodStart), cardDay(row.periodEnd)),
    t("Pays {0}", cardDate(row.payDate)),
    t("{0, plural, one {# shipment} other {# shipments}}", row.shipmentCount),
  ].join(" · ");
}

/** What a settlement pays, in its own currency. */
export function settlementAmount(row: CarrierSettlementRow): string {
  return minorMoney(row.netPayableMinor, row.currencyCode);
}

/** How to reach a contact: title, phone and email, whichever are on file. */
export function contactLine(contact: NonNullable<DeskCarrier["contacts"]>[number]): string {
  return [contact.title?.trim(), contact.phone?.trim(), contact.email?.trim()]
    .filter(Boolean)
    .join(" · ");
}

/** The carrier's contacts, the primary first, then by name. */
export function orderedContacts(
  contacts: NonNullable<DeskCarrier["contacts"]>,
): NonNullable<DeskCarrier["contacts"]> {
  return [...contacts].sort(
    (a, b) => Number(b.isPrimary) - Number(a.isPrimary) || a.name.localeCompare(b.name),
  );
}

/** The facts the card lists about the carrier, in the order the office reads them. */
export function carrierFacts(carrier: DeskCarrier, t: TranslateFn): CardFact[] {
  const address = [
    carrier.addressLine1?.trim(),
    carrier.city?.trim(),
    [carrier.state?.abbreviation ?? "", carrier.postalCode?.trim() ?? ""].filter(Boolean).join(" "),
  ]
    .filter(Boolean)
    .join(", ");
  return [
    { key: "type", label: t("Type"), value: choiceLabel(carrierTypeChoices, carrier.carrierType) },
    {
      key: "payment",
      label: t("Payment"),
      value: t(
        "{0}, net {1}",
        choiceLabel(carrierPaymentMethodChoices, carrier.paymentMethod),
        formatNumber(carrier.paymentTermDays),
      ),
    },
    {
      key: "remit",
      label: t("Remit to"),
      value: carrier.remitToName?.trim() || carrier.name,
    },
    { key: "w9", label: t("W-9"), value: carrier.w9OnFile ? t("On file") : t("Not on file") },
    { key: "phone", label: t("Phone"), value: carrier.phone?.trim() || t("None on file") },
    { key: "email", label: t("Email"), value: carrier.email?.trim() || t("None on file") },
    { key: "address", label: t("Address"), value: address || t("None on file") },
    {
      key: "qualified",
      label: t("Qualified"),
      value: carrier.qualifiedAt ? cardDate(carrier.qualifiedAt) : t("Not yet"),
    },
  ];
}
