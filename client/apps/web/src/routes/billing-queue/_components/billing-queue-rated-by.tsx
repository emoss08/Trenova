import { recordPath } from "@/config/record-links";
import type { BillingQueueRateAgreement } from "@/lib/graphql/billing-queue";
import { billingQueue } from "@/lib/queries/billing-queue";
import { LinkExternal01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { LockedReference } from "@trenova/shared/components/withheld-value";
import { useCanReadKind } from "@trenova/shared/hooks/use-can-read-kind";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { humanizeTariffName } from "@trenova/shared/lib/tariff-inputs";
import { formatTariffValue } from "@trenova/shared/lib/tariff-value";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type {
  BillingQueueItem,
  BillingQueueRatedBy as RatedBy,
} from "@trenova/shared/types/billing-queue";
import type { TariffWireValue } from "@trenova/shared/types/tariff";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Link } from "react-router";
import { BillingQueueRatedBySkeleton } from "./billing-queue-rated-by-skeleton";

const DATE = { month: "short", day: "numeric", year: "numeric" } as const;
const DATE_TIME = { month: "short", day: "numeric", hour: "numeric", minute: "2-digit" } as const;

/**
 * What priced the shipment, opening its terms in place: the rule that matched, the
 * contract's accessorials and what this load bills of them, and the fuel program. The
 * terms load when the card opens. Someone who cannot read the agreement or template sees
 * its name with a lock and no card, so its terms never reach the page.
 *
 * `trigger` is the name itself, as the charges header shows it, or a "See the terms"
 * button beside a summary that already names it.
 */
export function BillingQueueRatedBy({
  item,
  ratedBy,
  trigger = "name",
}: {
  item: BillingQueueItem;
  ratedBy: RatedBy;
  trigger?: "name" | "terms";
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const agreement = ratedBy.kind === "Agreement";
  const kind = agreement ? "rate_agreement" : "formula_template";
  const canRead = useCanReadKind(kind);
  const name = ratedByName(ratedBy, t);

  if (!canRead) {
    return trigger === "name" ? <LockedReference kind={kind}>{name}</LockedReference> : null;
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          trigger === "name" ? (
            <button
              type="button"
              className="text-brand ui-focus-ring rounded-xs font-medium hover:underline"
            />
          ) : (
            <Button type="button" size="xs" variant="outline" />
          )
        }
      >
        {trigger === "name" ? name : t("See the terms")}
      </PopoverTrigger>
      <PopoverContent align="end" className="w-105 p-0 text-xs">
        {open ? <RatedByCard item={item} ratedBy={ratedBy} /> : null}
      </PopoverContent>
    </Popover>
  );
}

/** The agreement by its code and name, or the template by its name. */
export function ratedByName(ratedBy: RatedBy, t: TranslateFn): string {
  return ratedBy.kind === "Agreement" && ratedBy.code
    ? t("{0} · {1}", ratedBy.code, ratedBy.name)
    : ratedBy.name;
}

function RatedByCard({ item, ratedBy }: { item: BillingQueueItem; ratedBy: RatedBy }) {
  const t = useT();
  const agreement = ratedBy.kind === "Agreement";
  const shipmentId = item.shipmentId ?? "";
  const shipment = useQuery({ ...billingQueue.shipment(shipmentId), enabled: Boolean(shipmentId) });
  const contract = useQuery({ ...billingQueue.rateAgreement(ratedBy.id), enabled: agreement });
  const template = useQuery({ ...billingQueue.formulaTemplate(ratedBy.id), enabled: !agreement });
  const terms = agreement ? contract : template;

  if ((shipmentId && shipment.isPending) || terms.isPending) {
    return <BillingQueueRatedBySkeleton />;
  }
  if (terms.isError || !terms.data) {
    return (
      <p className="text-foreground-muted px-3.5 py-4" role="alert">
        {t("The terms couldn't be loaded. Close this and try again.")}
      </p>
    );
  }

  const rating = shipment.data?.ratingDetail ?? null;
  const inputs = Object.entries((rating?.inputs as Record<string, unknown> | null) ?? {});
  const path = agreement
    ? recordPath("rate_agreement", ratedBy.id)
    : recordPath("formula_template", ratedBy.id);

  return (
    <div className="flex flex-col" data-testid="billing-queue-rated-by">
      {agreement && contract.data ? (
        <CardHeader
          title={ratedByName(ratedBy, t)}
          badge={agreementStatusLabel(contract.data.status, t)}
          active={contract.data.status === "Active"}
          subtitle={
            contract.data.effectiveTo
              ? t(
                  "{0} – {1} · rate agreement",
                  formatUnixInUserTimezone(contract.data.effectiveFrom, DATE),
                  formatUnixInUserTimezone(contract.data.effectiveTo, DATE),
                )
              : t(
                  "From {0} · rate agreement",
                  formatUnixInUserTimezone(contract.data.effectiveFrom, DATE),
                )
          }
        />
      ) : (
        <CardHeader
          title={ratedBy.name}
          badge={t("Formula template")}
          active={false}
          subtitle={
            template.data?.description ||
            t("No contract covers this lane, so the shipment rated on its template.")
          }
        />
      )}

      <div className="flex flex-col gap-3 px-3.5 py-3">
        <section className="flex flex-col gap-1.5">
          <h4 className="text-foreground-subtle text-xs font-medium">
            {agreement ? t("The rule that matched") : t("What it charged")}
          </h4>
          {rating && (rating.ruleLabel || inputs.length > 0) ? (
            <DescriptionList layout="inline" className="text-xs">
              {agreement && rating.ruleLabel ? (
                <DescriptionItem label={t("Rule")} valueClassName="text-xs">
                  {rating.ruleLabel}
                </DescriptionItem>
              ) : null}
              {inputs.map(([inputName, value]) => (
                <DescriptionItem
                  key={inputName}
                  label={humanizeTariffName(inputName)}
                  valueClassName="font-mono text-xs"
                >
                  {formatTariffValue(value as TariffWireValue, t)}
                </DescriptionItem>
              ))}
            </DescriptionList>
          ) : (
            <p className="text-foreground-muted">{t("The rating recorded no terms.")}</p>
          )}
        </section>

        {agreement && contract.data ? (
          <ContractAccessorials item={item} contract={contract.data} />
        ) : null}

        {agreement && contract.data ? (
          <section className="flex flex-col gap-1">
            <h4 className="text-foreground-subtle text-xs font-medium">{t("Fuel")}</h4>
            <p className="text-foreground">
              {contract.data.fuelBinding?.waived
                ? t("Waived on this contract")
                : contract.data.fuelBinding?.programName ||
                  t("This contract names no fuel program")}
            </p>
          </section>
        ) : null}
      </div>

      <div className="border-border flex items-center gap-2 border-t px-3.5 py-2.5">
        <span className="text-foreground-muted flex-1">
          {rating?.ratedAt
            ? t("Rated {0}", formatUnixInUserTimezone(rating.ratedAt, DATE_TIME))
            : null}
        </span>
        <Link
          to={path}
          className="text-brand ui-focus-ring inline-flex items-center gap-1 rounded-xs font-medium hover:underline"
        >
          {agreement ? t("Open agreement") : t("Open template")}
          <LinkExternal01Icon className="size-3" />
        </Link>
      </div>
    </div>
  );
}

function CardHeader({
  title,
  badge,
  active,
  subtitle,
}: {
  title: string;
  badge: string;
  active: boolean;
  subtitle: string;
}) {
  return (
    <div className="border-border flex flex-col gap-1 border-b px-3.5 py-3">
      <div className="flex items-center gap-2">
        <span className="text-foreground text-sm font-semibold">{title}</span>
        <span
          className={cn(
            "rounded-xs px-1.5 text-2xs font-medium",
            active ? "bg-success-subtle text-success-foreground" : "bg-muted text-foreground-muted",
          )}
        >
          {badge}
        </span>
      </div>
      <span className="text-foreground-muted">{subtitle}</span>
    </div>
  );
}

/**
 * The contract's accessorial schedule, each marked with what this load does with it:
 * billed, suggested by a receipt on the shipment, waived, or not billed.
 */
function ContractAccessorials({
  item,
  contract,
}: {
  item: BillingQueueItem;
  contract: BillingQueueRateAgreement;
}) {
  const t = useT();
  const accessorials = contract.accessorials ?? [];
  if (accessorials.length === 0) return null;

  const billed = new Set(
    (item.shipment?.additionalCharges ?? []).map((charge) => charge.accessorialChargeId),
  );
  const suggested = new Set(
    (item.review?.issues ?? [])
      .filter((issue) => !issue.resolutionKey)
      .flatMap((issue) => issue.options)
      .filter((option) => option.effect.kind === "add")
      .map((option) => option.effect.accessorialChargeId),
  );
  const marked = accessorials.map((accessorial) => {
    const state = accessorial.waived
      ? "waived"
      : billed.has(accessorial.accessorialChargeId)
        ? "billed"
        : suggested.has(accessorial.accessorialChargeId)
          ? "suggested"
          : "none";
    return { accessorial, state } as const;
  });
  const order = { billed: 0, suggested: 1, none: 2, waived: 3 } as const;
  marked.sort((a, b) => order[a.state] - order[b.state]);

  return (
    <section className="flex flex-col gap-1.5">
      <h4 className="text-foreground-subtle text-xs font-medium">
        {t("Accessorials on this contract")}
      </h4>
      <ul className="flex flex-col gap-1.5">
        {marked.map(({ accessorial, state }) => (
          <li key={accessorial.id} className="flex items-baseline gap-2">
            <span className="min-w-0 flex-1">
              <span className={cn("text-foreground", state === "billed" && "font-medium")}>
                {accessorial.chargeDescription || accessorial.chargeCode}
              </span>{" "}
              <span className="text-foreground-muted">· {accessorialTerms(accessorial, t)}</span>
            </span>
            <span
              className={cn(
                "rounded-xs px-1.25 text-2xs font-medium whitespace-nowrap",
                state === "billed"
                  ? "bg-success-subtle text-success-foreground"
                  : state === "suggested"
                    ? "bg-warning-subtle text-warning-foreground"
                    : "bg-muted text-foreground-muted",
              )}
            >
              {state === "billed"
                ? t("Billed")
                : state === "suggested"
                  ? t("Suggested")
                  : state === "waived"
                    ? t("Waived")
                    : t("Not billed")}
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}

type ContractAccessorial = NonNullable<BillingQueueRateAgreement["accessorials"]>[number];

/** An accessorial's price on the contract, said the way a biller reads a rate sheet. */
export function accessorialTerms(accessorial: ContractAccessorial, t: TranslateFn): string {
  const amount = Number(accessorial.amount);
  const money = formatCurrency(amount);
  const free = accessorial.freeUnits ?? 0;
  if (accessorial.method === "Percentage") {
    return t("{0}% of line haul", amount);
  }
  if (accessorial.method !== "PerUnit") {
    return money;
  }
  switch (accessorial.rateUnit) {
    case "Hour":
      return free > 0
        ? t("{0} an hour after {1, plural, one {# free hour} other {# free hours}}", money, free)
        : t("{0} an hour", money);
    case "Day":
      return free > 0
        ? t("{0} a day after {1, plural, one {# free day} other {# free days}}", money, free)
        : t("{0} a day", money);
    case "Stop":
      return free > 0
        ? t("{0} a stop after {1, plural, one {# free stop} other {# free stops}}", money, free)
        : t("{0} a stop", money);
    case "Mile":
      return t("{0} a mile", money);
    default:
      return money;
  }
}

function agreementStatusLabel(status: string, t: TranslateFn): string {
  switch (status) {
    case "Active":
      return t("Active");
    case "Draft":
      return t("Draft");
    case "InReview":
      return t("In review");
    case "Suspended":
      return t("Suspended");
    case "Expired":
      return t("Expired");
    case "Archived":
      return t("Archived");
    default:
      return status;
  }
}
