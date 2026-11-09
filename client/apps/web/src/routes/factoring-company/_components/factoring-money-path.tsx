import { HoverGlideGroup } from "@/components/hover-glide";
import type { FactoringCompanyOverview } from "@/lib/graphql/factoring-company-page";
import { factoringWorkspacePath, type FactoringTab } from "@/routes/factoring/use-factoring-state";
import {
  ArrowNarrowRightIcon,
  CheckCircleIcon,
  ChevronRightIcon,
  CoinsStacked01Icon,
  CornerDownRightIcon,
  FileCheck02Icon,
  Send01Icon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { formatMinor } from "@trenova/shared/lib/money";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { Link } from "react-router";
import type { FactoringCompanyTab } from "../use-factoring-company-state";

type Stage = {
  key: string;
  icon: IconComponent;
  label: string;
  value: string;
  count: string;
  note: ReactNode;
  tone?: "warning" | "success";
  to?: string;
  onOpen?: () => void;
  hint: string;
};

function StageCell({ stage }: { stage: Stage }) {
  const body = (
    <>
      <span className="flex items-center gap-1.5 text-xs font-medium text-foreground-subtle">
        <stage.icon className="size-3" />
        {stage.label}
      </span>
      <span className="text-xl leading-tight font-semibold tabular-nums">{stage.value}</span>
      <span className="text-xs text-muted-foreground tabular-nums">{stage.count}</span>
      <span
        className={cn(
          "mt-1 text-xs",
          stage.tone === "warning"
            ? "text-warning-foreground"
            : stage.tone === "success"
              ? "text-success-foreground"
              : "text-muted-foreground",
        )}
      >
        {stage.note}
      </span>
      <ArrowNarrowRightIcon className="absolute top-3 right-3 size-3.5 text-foreground-faint opacity-0 transition-opacity group-hover:opacity-100" />
    </>
  );
  const className =
    "group relative flex min-w-0 flex-col gap-0.5 rounded-md ui-focus-ring px-3 py-3 text-left";

  if (stage.to) {
    return (
      <Link to={stage.to} data-glide-target title={stage.hint} className={className}>
        {body}
      </Link>
    );
  }
  return (
    <button
      type="button"
      data-glide-target
      title={stage.hint}
      onClick={stage.onOpen}
      className={className}
    >
      {body}
    </button>
  );
}

/**
 * Where the factor's invoices are on their way to cash: ready to send, with the factor, funded
 * with reserve held, and released, with the chargebacks that branch off. Each stage opens the
 * factoring workspace narrowed to this factor, or the ledger for what has come back.
 */
export function FactoringMoneyPath({
  overview,
  factoringCompanyId,
  onTab,
}: {
  overview: FactoringCompanyOverview;
  factoringCompanyId: string;
  onTab: (tab: FactoringCompanyTab) => void;
}) {
  const t = useT();
  const amounts = overview.amountsIncluded;
  const currency = overview.currencyCode;
  const money = (minor: number) => formatMinor(minor, currency);
  const invoices = (count: number) => t("{0, plural, one {# invoice} other {# invoices}}", count);
  const workspace = (view: FactoringTab) => factoringWorkspacePath(view, factoringCompanyId);
  const headline = (totals: { count: number; amountMinor: number }) =>
    amounts ? money(totals.amountMinor) : invoices(totals.count);
  const awaiting = overview.awaiting;

  const stages: Stage[] = [
    {
      key: "ready",
      icon: FileCheck02Icon,
      label: t("Ready to submit"),
      value: headline(overview.ready),
      count: amounts ? invoices(overview.ready.count) : "",
      note:
        overview.readyLate.count > 0
          ? t(
              "{0, plural, one {# has} other {# have}} been ready for over a day",
              overview.readyLate.count,
            )
          : overview.ready.count > 0
            ? t("All ready since today")
            : t("Nothing waiting"),
      tone: overview.readyLate.count > 0 ? "warning" : undefined,
      to: workspace("ready"),
      hint: t("Opens the factoring workspace with this factor's ready invoices"),
    },
    {
      key: "submitted",
      icon: Send01Icon,
      label: t("With the factor"),
      value: headline(overview.submitted),
      count:
        awaiting && amounts
          ? t(
              "{0} · {1, plural, one {# schedule} other {# schedules}} waiting",
              invoices(overview.submitted.count),
              overview.awaitingCount,
            )
          : awaiting
            ? t("{0, plural, one {# schedule} other {# schedules}} waiting", overview.awaitingCount)
            : amounts
              ? invoices(overview.submitted.count)
              : "",
      note: awaiting
        ? t(
            "{0} sent {1}, funding due {2}",
            awaiting.number,
            formatUnixDate(awaiting.sentAt),
            formatUnixDate(awaiting.dueAt),
          )
        : t("No schedule waiting on funding"),
      to: workspace("submitted"),
      hint: t("Opens the factoring workspace with this factor's submitted invoices"),
    },
    {
      key: "funded",
      icon: CoinsStacked01Icon,
      label: t("Funded, reserve held"),
      value: headline(overview.funded),
      count: amounts
        ? t(
            "{0} · {1} reserve held",
            invoices(overview.funded.count),
            money(overview.reserveHeld.amountMinor),
          )
        : "",
      note: overview.oldestFundedOn
        ? t("Oldest funded {0}", formatUnixDate(overview.oldestFundedOn))
        : t("Nothing funded and outstanding"),
      to: workspace("funded"),
      hint: t("Opens the factoring workspace with this factor's funded invoices"),
    },
    {
      key: "released",
      icon: CheckCircleIcon,
      label: t("Released, last {0} days", overview.movedWindowDays),
      value: amounts ? money(overview.released.amountMinor) : invoices(overview.released.count),
      count: amounts ? t("{0} · back to you", invoices(overview.released.count)) : "",
      note:
        overview.released.count > 0
          ? t("Reserve released as customers paid")
          : t("Nothing released in the window"),
      tone: overview.released.count > 0 ? "success" : undefined,
      onOpen: () => onTab("ledger"),
      hint: t("Opens the ledger"),
    },
  ];

  return (
    <section aria-label={t("Where the invoices are")} className="rounded-lg border bg-card">
      <header className="flex min-h-9 items-center justify-between gap-2 border-b px-3 py-1.5">
        <h3 className="text-sm font-semibold">{t("Where the invoices are")}</h3>
        <span className="hidden text-xs text-muted-foreground sm:inline">
          {amounts
            ? t("Open balance and face value, in {0}", currency)
            : t("Invoice counts; amounts need factoring reserve access")}
        </span>
      </header>
      <HoverGlideGroup className="rounded-b-lg bg-card p-1.5">
        <ol className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-[1fr_auto_1fr_auto_1.2fr_auto_1fr]">
          {stages.map((stage, index) => (
            <li key={stage.key} className="contents">
              <StageCell stage={stage} />
              {index < stages.length - 1 ? (
                <span aria-hidden className="hidden items-center text-foreground-faint lg:flex">
                  <ChevronRightIcon className="size-4" />
                </span>
              ) : null}
            </li>
          ))}
        </ol>
      </HoverGlideGroup>
      <div className="grid border-t lg:grid-cols-[1fr_auto_1fr_auto_1.2fr_auto_1fr]">
        <p className="col-span-3 hidden items-center px-3 py-2.5 text-xs text-foreground-subtle lg:flex">
          {t("A rejected invoice goes back to ready, with the factor's reason on it.")}
        </p>
        <div className="col-span-1 flex flex-col sm:flex-row lg:col-span-4">
          <Link
            to={workspace("funded")}
            className="flex flex-1 items-start gap-2 ui-focus-ring px-3 py-2.5 text-xs transition-colors hover:bg-surface-hover lg:border-l"
          >
            <CornerDownRightIcon
              className={cn(
                "mt-0.5 size-3.5 shrink-0",
                overview.chargebacksDue.count > 0
                  ? "text-danger-foreground"
                  : "text-foreground-subtle",
              )}
            />
            <span className="flex flex-col gap-0.5">
              <span
                className={cn(
                  "font-medium tabular-nums",
                  overview.chargebacksDue.count > 0 && "text-danger-foreground",
                )}
              >
                {amounts
                  ? t(
                      "{0} · {1}",
                      invoices(overview.chargebacksDue.count),
                      money(overview.chargebacksDue.amountMinor),
                    )
                  : invoices(overview.chargebacksDue.count)}
              </span>
              <span className="text-muted-foreground">
                {overview.chargebackDeadline
                  ? t(
                      "Chargeback window closes by {0}. You repay these if the factor hasn't collected.",
                      formatUnixDate(overview.chargebackDeadline),
                    )
                  : t("No recourse invoice reaches its chargeback window this week")}
              </span>
            </span>
          </Link>
          <button
            type="button"
            onClick={() => onTab("ledger")}
            className="flex flex-1 items-start gap-2 border-t ui-focus-ring px-3 py-2.5 text-left text-xs transition-colors hover:bg-surface-hover sm:border-t-0 sm:border-l"
          >
            <CornerDownRightIcon className="mt-0.5 size-3.5 shrink-0 text-foreground-subtle" />
            <span className="flex flex-col gap-0.5">
              <span className="font-medium tabular-nums">
                {amounts
                  ? t(
                      "{0} · {1}",
                      invoices(overview.chargedBack.count),
                      money(overview.chargedBack.amountMinor),
                    )
                  : invoices(overview.chargedBack.count)}
              </span>
              <span className="text-muted-foreground">
                {t("Charged back in the last {0} days", overview.movedWindowDays)}
              </span>
            </span>
          </button>
        </div>
      </div>
    </section>
  );
}
