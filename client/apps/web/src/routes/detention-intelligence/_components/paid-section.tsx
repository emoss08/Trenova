import {
  HoverCardHeading,
  HoverCardRow,
  PointerHoverCard,
  usePointerHoverTarget,
} from "@/components/pointer-hover-card";
import { SectionPanel } from "@/components/section-panel";
import { VirtualRows, type VirtualRow } from "@/components/virtual-rows";
import type {
  DetentionCustomer,
  DetentionIntelligence,
} from "@/lib/graphql/detention-intelligence";
import { detentionWaiverReasonChoices, findChoice } from "@/lib/choices";
import { WithheldValue } from "@trenova/shared/components/withheld-value";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { EASE_SETTLE } from "@/lib/motion";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { m, useReducedMotionConfig } from "motion/react";
import { useEffect, useRef, useState } from "react";
import { Link } from "react-router";
import {
  DETENTION_DESK_PATH,
  DETENTION_POLICIES_PATH,
  NOTICE_ON_TIME_TARGET,
  RECOVERY_LABELS,
  amount,
  hours,
  keptAfterDriverPay,
  recoverySteps,
  shareOf,
} from "./intelligence-model";
import {
  BESIDE_PANEL,
  Finding,
  GrowingRule,
  HATCH_BACKGROUND,
  IntelligenceSection,
  money,
  percent,
} from "./intelligence-parts";

const strong = (chunk: React.ReactNode) => (
  <span className="text-foreground font-medium">{chunk}</span>
);

const COLLAPSED_CUSTOMERS = 6;
/** The expanded list's height: its header and six rows (24px + 6 × 36px), as tall as the collapsed list. */
const EXPANDED_LIST_HEIGHT = "h-60";
const ROW_STAGGER = 0.035;

/** The leak colours the mockup draws, each bar where its money left the chain. */
const LEAK_CLASSES: Record<string, string> = {
  Late: "bg-accent-rose",
  Notice: "bg-danger",
  Waived: "bg-warning",
  Pending: "bg-border-strong",
  Other: "bg-muted-foreground",
  Capped: "bg-muted-foreground",
};
/** A dispute is half lost and half still in play: the danger and warning hues, mixed. */
const DISPUTED_BACKGROUND = "color-mix(in oklch, var(--danger) 50%, var(--warning))";

type Step = DetentionIntelligence["recovery"]["steps"][number];

/** What each leak counted, in its own words: stops, held charges, waivers, open disputes. */
function useStepCaption() {
  const t = useT();
  return (step: Step): string => {
    switch (step.bucket) {
      case "Late":
        return t("{0, plural, one {# stop} other {# stops}}", step.occurrenceCount);
      case "Notice":
        return t("{0, plural, one {# charge held} other {# charges held}}", step.occurrenceCount);
      case "Waived":
        return t("{0, plural, one {# waiver} other {# waivers}}", step.occurrenceCount);
      case "Disputed":
        return t("{0} open", step.occurrenceCount);
      case "Pending":
        return t("{0} waiting", step.occurrenceCount);
      default:
        return t("{0, plural, one {# charge} other {# charges}}", step.occurrenceCount);
    }
  };
}

function fixLink(to: string) {
  return (chunk: React.ReactNode) => (
    <Link
      to={to}
      className="ui-focus-ring text-foreground decoration-border-strong hover:decoration-foreground rounded-sm underline underline-offset-3"
    >
      {chunk}
    </Link>
  );
}

/** What closes a leak, with the action linked to the screen it is taken on. */
function LeakFix({ bucket }: { bucket: Step["bucket"] }) {
  const t = useT();
  const rt = useRichT();
  switch (bucket) {
    case "Late":
      return t("Free time forfeited; not billable under the contract.");
    case "Notice":
      return rt("<a>Send notice</a> when free time runs out.", {
        a: fixLink(DETENTION_POLICIES_PATH),
      });
    case "Waived":
      return rt("<a>Require a coded reason and a note</a> on every waiver.", {
        a: fixLink(DETENTION_POLICIES_PATH),
      });
    case "Disputed":
      return rt("<a>Work the disputes</a> in the detention desk.", {
        a: fixLink(DETENTION_DESK_PATH),
      });
    case "Pending":
      return rt("<a>Approve and bill them</a> from the detention desk.", {
        a: fixLink(DETENTION_DESK_PATH),
      });
    default:
      return null;
  }
}

/** A leak as its tooltip reads it. */
type LeakTip = { id: string; label: string; caption: string; value: string };

function LeakDetail({ leak }: { leak: LeakTip }) {
  return (
    <>
      <HoverCardHeading>{leak.label}</HoverCardHeading>
      <HoverCardRow label={leak.caption} value={leak.value} />
    </>
  );
}

type Bar =
  | { kind: "fill"; left: number; width: number; className?: string; background?: string }
  | { kind: "withheld"; width: number };

function MoneyRow({
  label,
  caption,
  bar,
  connector,
  value,
  valueClass,
  fix,
  tip,
  total = false,
}: {
  label: string;
  caption: string;
  bar: Bar;
  /** Where the money stood before this leak: a dashed line ties the bar to the row above. */
  connector?: number;
  value: React.ReactNode;
  valueClass?: string;
  fix?: React.ReactNode;
  tip?: LeakTip;
  total?: boolean;
}) {
  const hover = usePointerHoverTarget(tip ?? null, tip?.id ?? label);

  return (
    <div
      className="grid min-h-10.5 grid-cols-[minmax(8rem,10.5rem)_minmax(0,1fr)_5.25rem] items-center gap-x-4 border-t first:border-t-0 lg:grid-cols-[minmax(9.5rem,12.5rem)_minmax(0,1fr)_5.5rem_minmax(9.5rem,14.5rem)]"
      {...(tip ? hover : {})}
    >
      <div className="min-w-0 py-1.5">
        <p className={cn("truncate text-xs", total ? "font-semibold" : "font-medium")}>{label}</p>
        <p className="text-muted-foreground text-2xs truncate">{caption}</p>
      </div>
      <span className="relative h-3.5">
        {connector !== undefined ? (
          <span
            aria-hidden
            className="border-border-strong absolute -inset-y-3.5 border-l border-dashed"
            style={{ left: `${connector * 100}%` }}
          />
        ) : null}
        {bar.kind === "withheld" ? (
          <span
            className="ring-border absolute inset-y-0 left-0 rounded-xs ring-1 ring-inset"
            style={{ width: `${bar.width * 100}%`, background: HATCH_BACKGROUND }}
          />
        ) : (
          <span
            className={cn("absolute inset-y-0 rounded-xs", bar.className)}
            style={{
              left: `${bar.left * 100}%`,
              width: `${Math.max(bar.width * 100, 0.4)}%`,
              background: bar.background,
            }}
          />
        )}
      </span>
      <span className={cn("text-right font-mono text-xs font-medium tabular-nums", valueClass)}>
        {value}
      </span>
      {fix ? (
        <span className="text-muted-foreground col-span-full -mt-1 pb-2 text-xs lg:col-span-1 lg:mt-0 lg:pb-0">
          {fix}
        </span>
      ) : (
        <span className="hidden lg:block" />
      )}
    </div>
  );
}

/**
 * The money chain as the mockup draws it: what the clock accrued, each leak floating where its
 * money left, what was billed, invoiced and collected, then driver pay taken from the billed and
 * what was kept.
 */
function RecoveryCard({ data }: { data: DetentionIntelligence }) {
  const t = useT();
  const stepCaption = useStepCaption();
  const { recovery, totals } = data;
  const accrued = amount(recovery.accrued);
  const scale = (value: number) => shareOf(value, accrued);
  const steps = recoverySteps(recovery);
  const billed = amount(recovery.billed);
  const driverPay = totals.driverPayAmount === null ? null : amount(totals.driverPayAmount);
  // Kept is what was billed less what drivers were paid, so it never outruns the billed bar.
  const kept = keptAfterDriverPay(data);
  let remaining = accrued;

  return (
    <SectionPanel
      title={t("From accrued to kept")}
      description={t("Each row takes away what never reached an invoice, and names the fix.")}
    >
      <PointerHoverCard<LeakTip | null>
        follow="pointer"
        itemId={(leak) => leak?.id ?? ""}
        render={(leak) => (leak ? <LeakDetail leak={leak} /> : null)}
      >
        <div className="px-3 pt-1 pb-1.5">
          <MoneyRow
            total
            label={t("Accrued")}
            caption={t("Every billable minute past free time")}
            bar={{ kind: "fill", left: 0, width: 1, className: "bg-foreground/20" }}
            value={money(accrued)}
          />
          {steps.map((step) => {
            const value = amount(step.amount);
            remaining -= value;
            const left = scale(Math.max(remaining, 0));
            const caption = stepCaption(step);
            return (
              <MoneyRow
                key={step.bucket}
                label={RECOVERY_LABELS[step.bucket]}
                caption={caption}
                connector={left}
                bar={{
                  kind: "fill",
                  left,
                  width: scale(value),
                  className: step.bucket === "Disputed" ? undefined : LEAK_CLASSES[step.bucket],
                  background: step.bucket === "Disputed" ? DISPUTED_BACKGROUND : undefined,
                }}
                value={t("−{0}", money(value))}
                valueClass="text-danger-foreground"
                fix={<LeakFix bucket={step.bucket} />}
                tip={{
                  id: step.bucket,
                  label: RECOVERY_LABELS[step.bucket],
                  caption,
                  value: t("−{0}", money(value)),
                }}
              />
            );
          })}
          <MoneyRow
            total
            label={t("Billed")}
            caption={t("{0} of accrued", percent(scale(billed)))}
            bar={{ kind: "fill", left: 0, width: scale(billed), className: "bg-foreground/60" }}
            value={money(billed)}
          />
          <MoneyRow
            label={t("On a posted invoice")}
            caption={t("{0} of billed", percent(shareOf(amount(recovery.invoiced), billed)))}
            bar={{
              kind: "fill",
              left: 0,
              width: scale(amount(recovery.invoiced)),
              className: "bg-foreground/40",
            }}
            value={money(amount(recovery.invoiced))}
          />
          <MoneyRow
            total
            label={t("Collected")}
            caption={t("Invoices paid in full")}
            bar={{
              kind: "fill",
              left: 0,
              width: scale(amount(recovery.collected)),
              className: "bg-accent-emerald",
            }}
            value={money(amount(recovery.collected))}
            fix={t("Only fully paid invoices count; a partly paid one is still open.")}
          />
          <MoneyRow
            label={t("Driver pay")}
            caption={t("Detention paid out to drivers")}
            bar={
              driverPay === null
                ? { kind: "withheld", width: scale(billed) }
                : {
                    kind: "fill",
                    left: scale(Math.max(billed - driverPay, 0)),
                    width: scale(Math.min(driverPay, billed)),
                    className: "bg-info",
                  }
            }
            value={
              driverPay === null ? <WithheldValue group="Margins" /> : t("−{0}", money(driverPay))
            }
          />
          <MoneyRow
            total
            label={t("Kept")}
            caption={t("Billed less driver pay")}
            bar={
              kept === null
                ? { kind: "withheld", width: scale(billed) }
                : {
                    kind: "fill",
                    left: 0,
                    width: scale(Math.max(kept, 0)),
                    className: kept < 0 ? "bg-danger" : "bg-success",
                  }
            }
            value={kept === null ? <WithheldValue group="Margins" /> : money(kept)}
            valueClass={
              kept === null
                ? undefined
                : kept < 0
                  ? "text-danger-foreground"
                  : "text-success-foreground"
            }
          />
        </div>
      </PointerHoverCard>
    </SectionPanel>
  );
}

/** One grid for the header and every row, so the columns line up without a table. */
const CUSTOMER_COLUMNS =
  "grid grid-cols-[minmax(0,1fr)_4.5rem_6rem_7rem] items-center gap-x-3 sm:grid-cols-[minmax(0,1fr)_4.5rem_5.5rem_6rem_7rem] md:grid-cols-[minmax(0,1fr)_4.5rem_5.5rem_5.5rem_6rem_7rem]";
const CUSTOMER_ROW_HEIGHT = 36;
const CUSTOMER_HEADER_HEIGHT = 24;
/** How long after "Show more" a row that mounts still arrives with motion; later ones scrolled to do not. */
const REVEAL_MS = 700;
const REVEAL_STAGGERED = 10;
/** How long "Show fewer" lets the added rows fade before the list collapses. */
const LEAVE_MS = 380;
const LEAVE_STAGGER = 0.015;

function CustomerHeader() {
  const t = useT();
  return (
    <div
      role="row"
      className={cn(
        CUSTOMER_COLUMNS,
        "bg-card text-muted-foreground text-2xs h-6 border-b text-right font-medium",
      )}
    >
      <span role="columnheader" className="text-left">
        {t("Customer")}
      </span>
      <span role="columnheader">{t("Past free")}</span>
      <span role="columnheader" className="hidden sm:block">
        {t("Billed")}
      </span>
      <span role="columnheader" className="hidden md:block">
        {t("Driver pay")}
      </span>
      <span role="columnheader">{t("Net")}</span>
      <span role="columnheader">{t("Notice on time")}</span>
    </div>
  );
}

function CustomerRow({
  customer,
  entering,
  revealing,
  leaving,
  last,
}: {
  customer: DetentionCustomer;
  /** Its place among the rows "Show more" added, or null for the rows shown from the start. */
  entering: number | null;
  /** True just after "Show more", so a row mounted by scrolling later does not replay it. */
  revealing: boolean;
  /** True while "Show fewer" takes the added rows away, before the list collapses. */
  leaving: boolean;
  /** The last row shown draws no rule under it: the panel's edge closes the list. */
  last: boolean;
}) {
  const t = useT();
  const [arrives] = useState(entering !== null && revealing);
  const net = customer.netMargin === null ? null : amount(customer.netMargin);
  const notice =
    customer.noticeDueCount > 0
      ? shareOf(customer.noticeOnTimeCount, customer.noticeDueCount)
      : null;

  return (
    <m.div
      role="row"
      className={cn(CUSTOMER_COLUMNS, "h-9 text-sm", !last && "border-b")}
      initial={arrives ? { opacity: 0, y: -6 } : false}
      animate={leaving && entering !== null ? { opacity: 0, y: -6 } : { opacity: 1, y: 0 }}
      transition={
        leaving
          ? {
              duration: 0.18,
              ease: EASE_SETTLE,
              delay: (REVEAL_STAGGERED - Math.min(entering ?? 0, REVEAL_STAGGERED)) * LEAVE_STAGGER,
            }
          : {
              duration: 0.24,
              ease: EASE_SETTLE,
              delay: arrives ? Math.min(entering ?? 0, REVEAL_STAGGERED) * ROW_STAGGER : 0,
            }
      }
    >
      <span role="cell" className="truncate font-medium" title={customer.customerName}>
        {customer.customerName}
      </span>
      <span role="cell" className="text-right font-mono tabular-nums">
        {t("{0} h", hours(customer.pastFreeMinutes).toLocaleString())}
      </span>
      <span role="cell" className="hidden text-right font-mono tabular-nums sm:block">
        {money(amount(customer.billedAmount))}
      </span>
      <span role="cell" className="hidden text-right font-mono tabular-nums md:block">
        {customer.driverPayAmount === null ? (
          <WithheldValue group="Margins" />
        ) : (
          money(amount(customer.driverPayAmount))
        )}
      </span>
      <span
        role="cell"
        className={cn(
          "text-right font-mono tabular-nums",
          net !== null && (net < 0 ? "text-danger-foreground" : "text-success-foreground"),
        )}
      >
        {net === null ? (
          <WithheldValue group="Margins" />
        ) : net > 0 ? (
          t("+{0}", money(net))
        ) : (
          money(net)
        )}
      </span>
      <span role="cell" className="text-right font-mono tabular-nums">
        {notice === null ? (
          "—"
        ) : (
          <span
            className={cn(
              "inline-flex items-center justify-end gap-2",
              notice < NOTICE_ON_TIME_TARGET && "text-danger-foreground",
            )}
          >
            {percent(notice)}
            <span aria-hidden className="bg-muted block h-1 w-6 overflow-hidden rounded-full">
              <span
                className={cn(
                  "block h-full rounded-full",
                  notice < NOTICE_ON_TIME_TARGET ? "bg-danger" : "bg-foreground/70",
                )}
                style={{ width: `${notice * 100}%` }}
              />
            </span>
          </span>
        )}
      </span>
    </m.div>
  );
}

export function CustomerCard({
  customers,
  marginsVisible,
}: {
  customers: readonly DetentionCustomer[];
  /** Read from the totals, so a window with no customer still knows whether margins show. */
  marginsVisible: boolean;
}) {
  const t = useT();
  const rt = useRichT();
  const [expanded, setExpanded] = useState(false);
  const [revealing, setRevealing] = useState(false);
  const [leaving, setLeaving] = useState(false);
  const motionTimer = useRef<number | undefined>(undefined);
  const scrollRef = useRef<HTMLDivElement>(null);
  const reduceMotion = useReducedMotionConfig() ?? false;
  useEffect(() => () => window.clearTimeout(motionTimer.current), []);
  const sorted = [...customers].sort((a, b) =>
    marginsVisible
      ? amount(a.netMargin) - amount(b.netMargin)
      : b.pastFreeMinutes - a.pastFreeMinutes,
  );
  const losing = customers.filter((customer) => amount(customer.netMargin) < 0);
  const worst = losing.length > 0 ? sorted[0] : undefined;
  const shown = expanded ? sorted : sorted.slice(0, COLLAPSED_CUSTOMERS);

  // Only the rows near the viewport are mounted, however many customers there are. The header is
  // the list's first row, pinned while the rest scroll under it.
  const rows: VirtualRow[] = [
    { key: "header", sticky: true, render: () => <CustomerHeader /> },
    ...shown.map((customer, index) => ({
      key: customer.customerId,
      render: () => (
        <CustomerRow
          customer={customer}
          entering={index >= COLLAPSED_CUSTOMERS ? index - COLLAPSED_CUSTOMERS : null}
          revealing={revealing}
          leaving={leaving}
          last={index === shown.length - 1}
        />
      ),
    })),
  ];

  // "Show more" adds the rows, which arrive one after another, and scrolls the last row already
  // shown up under the header so what it added is in view. "Show fewer" runs it backwards: the
  // list scrolls home while the added rows fade away, then it collapses. Every row is one height,
  // so the offset is exact. One timer covers both, so a quick second click never strands a state.
  const toggle = () => {
    const behavior = reduceMotion ? "auto" : "smooth";
    window.clearTimeout(motionTimer.current);

    if (!expanded || leaving) {
      setLeaving(false);
      setExpanded(true);
      setRevealing(true);
      motionTimer.current = window.setTimeout(() => setRevealing(false), REVEAL_MS);
      window.requestAnimationFrame(() => {
        scrollRef.current?.scrollTo({
          top: (COLLAPSED_CUSTOMERS - 1) * CUSTOMER_ROW_HEIGHT,
          behavior,
        });
      });
      return;
    }

    setRevealing(false);
    scrollRef.current?.scrollTo({ top: 0, behavior });
    if (reduceMotion) {
      setExpanded(false);
      return;
    }
    setLeaving(true);
    motionTimer.current = window.setTimeout(() => {
      setExpanded(false);
      setLeaving(false);
    }, LEAVE_MS);
  };

  return (
    <SectionPanel
      title={t("Customers whose terms cost us")}
      description={t(
        "Detention billed against detention paid to drivers, with how often notice went out on time.",
      )}
    >
      <div id="detention-customers" className="flex min-h-0 flex-1 scroll-mt-4 flex-col">
        <div className="shrink-0 px-3 pt-3">
          <Finding>
            {!marginsVisible
              ? t(
                  "Margin by customer needs access to margins. Hours and notice are shown to everyone.",
                )
              : worst
                ? rt(
                    "<b>{0} of {1}</b> customers cost more in driver pay than they are billed. <b>{2}</b> is the largest: {3}.",
                    { b: strong },
                    losing.length,
                    customers.length,
                    worst.customerName,
                    money(amount(worst.netMargin)),
                  )
                : t("Every customer's detention covers what drivers are paid for it.")}
          </Finding>
        </div>
        {expanded ? (
          <VirtualRows
            rows={rows}
            estimateSize={CUSTOMER_ROW_HEIGHT}
            initialHeight={CUSTOMER_HEADER_HEIGHT + COLLAPSED_CUSTOMERS * CUSTOMER_ROW_HEIGHT}
            mask="card"
            maskClassName="top-6"
            role="table"
            aria-label={t("Customers whose terms cost us")}
            className={cn("shrink-0 px-3", EXPANDED_LIST_HEIGHT)}
            scrollRef={scrollRef}
          />
        ) : (
          // Six rows or fewer fit as they are: no scroll area, no fade, nothing to virtualize.
          <div
            role="table"
            aria-label={t("Customers whose terms cost us")}
            className="shrink-0 px-3"
          >
            {rows.map((row) => (
              <div key={row.key}>{row.render()}</div>
            ))}
          </div>
        )}
        {sorted.length > COLLAPSED_CUSTOMERS ? (
          <div className="relative flex shrink-0 justify-end px-3 py-2">
            {/* The rule marks where the scrolling list ends, so it is drawn only while the list
                scrolls. */}
            <GrowingRule shown={expanded && !leaving} className="top-0" />
            <button
              type="button"
              aria-expanded={expanded && !leaving}
              onClick={toggle}
              className="ui-focus-ring text-muted-foreground hover:text-foreground rounded-sm text-xs font-medium"
            >
              {expanded && !leaving
                ? t("Show fewer")
                : t(
                    "{0, plural, one {Show # more customer} other {Show # more customers}}",
                    sorted.length - COLLAPSED_CUSTOMERS,
                  )}
            </button>
          </div>
        ) : null}
      </div>
    </SectionPanel>
  );
}

function reasonLabel(reason: string): string {
  return findChoice(detentionWaiverReasonChoices, reason)?.label ?? reason;
}

function WaiverCard({ waivers }: { waivers: DetentionIntelligence["waivers"] }) {
  const t = useT();
  const rt = useRichT();
  const sorted = [...waivers].sort((a, b) => amount(b.waivedAmount) - amount(a.waivedAmount));
  const total = sorted.reduce((sum, row) => sum + amount(row.waivedAmount), 0);
  const largest = sorted[0];
  const max = largest ? amount(largest.waivedAmount) : 0;

  return (
    <SectionPanel
      title={t("What was forgiven, and why")}
      description={t("Waived at someone's discretion, by the coded reason.")}
      className={BESIDE_PANEL}
    >
      <div className="shrink-0 px-3 pt-3">
        <Finding>
          {largest
            ? rt(
                "<b>{0}</b> is {1} of everything forgiven, granted by {2, plural, one {# person} other {# people}} across {3, plural, one {# waiver} other {# waivers}}.",
                { b: strong },
                t(reasonLabel(largest.reason)),
                percent(shareOf(amount(largest.waivedAmount), total)),
                largest.approverCount,
                largest.waiverCount,
              )
            : t("Nothing was waived in this window. Every accrued charge stood.")}
        </Finding>
      </div>
      <ScrollArea maskVariant="card" className="min-h-0 flex-1">
        <ul className="divide-y px-3 pb-3">
          {sorted.map((row) => (
            <li key={row.reason} className="py-2">
              <div className="flex items-baseline justify-between gap-3">
                <span className="text-sm font-medium">{t(reasonLabel(row.reason))}</span>
                <span className="font-mono text-sm tabular-nums">
                  {money(amount(row.waivedAmount))}
                </span>
              </div>
              <div className="mt-1 flex items-center gap-3">
                <span className="bg-muted h-1.5 flex-1 overflow-hidden rounded-full">
                  <span
                    className="bg-warning block h-full rounded-full"
                    style={{ width: `${shareOf(amount(row.waivedAmount), max) * 100}%` }}
                  />
                </span>
                <span className="text-muted-foreground text-2xs shrink-0 tabular-nums">
                  {t(
                    "{0, plural, one {# waiver} other {# waivers}} · {1, plural, one {# approver} other {# approvers}}",
                    row.waiverCount,
                    row.approverCount,
                  )}
                </span>
              </div>
            </li>
          ))}
        </ul>
      </ScrollArea>
    </SectionPanel>
  );
}

export function PaidSection({ data }: { data: DetentionIntelligence }) {
  const t = useT();
  const rt = useRichT();
  const accrued = amount(data.recovery.accrued);
  const collected = amount(data.recovery.collected);

  return (
    <IntelligenceSection
      id="paid"
      title={t("Getting paid for it")}
      answer={rt(
        "Of <b>{0}</b> the clock accrued, <b>{1}</b> was billed and <b>{2}</b> has been collected. Each leak below names its fix.",
        { b: strong },
        money(accrued),
        money(amount(data.recovery.billed)),
        money(collected),
      )}
    >
      <RecoveryCard data={data} />
      <div className="grid gap-3 xl:grid-cols-[minmax(0,1.6fr)_minmax(18rem,1fr)]">
        <CustomerCard customers={data.customers} marginsVisible={data.totals.netMargin !== null} />
        <div className="relative">
          <WaiverCard waivers={data.waivers} />
        </div>
      </div>
    </IntelligenceSection>
  );
}
