import { recordPath } from "@/config/record-links";
import { customerPaymentTermChoices, findChoice } from "@/lib/choices";
import type { RecordShipment } from "@/lib/graphql/record-shipments";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  cn,
  formatCompactCurrency,
  formatCurrency,
  formatPercent,
} from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { PALETTE_ENTITY_ICONS } from "../../palette-entities";
import type { StatusState } from "../../palette-parts";
import { shipmentStatus } from "../../palette-status";
import {
  PreviewError,
  PreviewFields,
  PreviewHead,
  PreviewSection,
  PreviewSkeleton,
  PreviewSource,
  PreviewStats,
  RelatedRow,
  StatePill,
  type PreviewStat,
} from "./preview-frame";
import {
  customerAgingQuery,
  customerPreviewQuery,
  OPEN_LOADS_SHOWN,
  openShipmentsQuery,
} from "./preview-queries";
import type { RecordPreviewProps } from "./record-preview-props";

const CREDIT_WARN_SHARE = 0.8;

function money(minor: number): string {
  return formatCurrency(minor / 100);
}

function compactMoney(minor: number): string {
  return formatCompactCurrency(minor / 100);
}

/** "Dallas, TX → Atlanta, GA" from a listed shipment's first and last stops. */
export function recordShipmentLane(shipment: RecordShipment): string {
  const stops = [...(shipment.moves ?? [])]
    .sort((a, b) => a.sequence - b.sequence)
    .flatMap((move) => [...(move.stops ?? [])].sort((a, b) => a.sequence - b.sequence));
  const place = (stop: (typeof stops)[number] | undefined) =>
    [stop?.location?.city, stop?.location?.state?.abbreviation].filter(Boolean).join(", ");
  const from = place(stops.find((stop) => stop.type.includes("Pickup")) ?? stops[0]);
  const to = place(
    [...stops].reverse().find((stop) => stop.type.includes("Delivery")) ?? stops.at(-1),
  );
  return from && to ? `${from} → ${to}` : from || to;
}

interface AgingBucket {
  label: string;
  minor: number;
  tone: string;
}

function AgingBar({ buckets, total }: { buckets: readonly AgingBucket[]; total: number }) {
  return (
    <>
      <div className="bg-sunken flex h-2 overflow-hidden rounded-full" aria-hidden>
        {buckets.map((bucket) =>
          bucket.minor > 0 ? (
            <span
              key={bucket.label}
              className={cn(
                "cp-grow-x h-full first:rounded-l-full last:rounded-r-full",
                bucket.tone,
              )}
              style={{ flexGrow: bucket.minor / Math.max(total, 1) }}
            />
          ) : null,
        )}
      </div>
      <div className="grid grid-cols-5 gap-2">
        {buckets.map((bucket) => (
          <span key={bucket.label} className="flex min-w-0 flex-col gap-0.5">
            <span className="text-foreground-subtle truncate text-sm">{bucket.label}</span>
            <span className="text-foreground truncate font-mono text-sm">
              {bucket.minor > 0 ? formatCompactCurrency(bucket.minor / 100) : "—"}
            </span>
          </span>
        ))}
      </div>
    </>
  );
}

export function CustomerPreview({ record, onNavigate }: RecordPreviewProps) {
  const t = useT();
  const customer = useQuery(customerPreviewQuery(record.id));
  const aging = useQuery(customerAgingQuery(record.id));
  const loads = useQuery(openShipmentsQuery({ kind: "customer", id: record.id }, OPEN_LOADS_SHOWN));

  if (customer.isLoading) {
    return <PreviewSkeleton />;
  }
  if (customer.isError || !customer.data) {
    return <PreviewError />;
  }

  const data = customer.data;
  const billing = data.billingProfile;
  const place = [data.city, data.state?.abbreviation].filter(Boolean).join(", ");
  const snapshot = aging.data?.snapshot;
  const buckets = snapshot?.buckets;
  const terms = billing
    ? t(findChoice(customerPaymentTermChoices, billing.paymentTerm)?.label ?? billing.paymentTerm)
    : null;
  const openLoads = loads.data?.total ?? null;

  const stats: PreviewStat[] = [];
  if (snapshot && buckets) {
    const overdue = snapshot.overdueMinor;
    const pastState: StatusState =
      buckets.daysOver90Minor > 0 ? "bad" : overdue > 0 ? "warn" : "neutral";
    stats.push(
      { label: t("Open balance"), value: compactMoney(snapshot.totalOpenMinor) },
      {
        label: t("Past terms"),
        value: overdue > 0 ? compactMoney(overdue) : "—",
        state: pastState,
      },
    );
    if (snapshot.hasCreditLimit && snapshot.creditLimitMinor > 0) {
      const used = aging.data!.creditUtilization;
      stats.push({
        label: t("Credit used"),
        value: formatPercent(used * 100, 0),
        state: used > 1 ? "bad" : used > CREDIT_WARN_SHARE ? "warn" : "neutral",
      });
    }
  }

  const agingBuckets: AgingBucket[] = buckets
    ? [
        { label: t("Current"), minor: buckets.currentMinor, tone: "bg-border-strong" },
        { label: t("1–30"), minor: buckets.days1To30Minor, tone: "bg-foreground-faint" },
        { label: t("31–60"), minor: buckets.days31To60Minor, tone: "bg-foreground-subtle" },
        { label: t("61–90"), minor: buckets.days61To90Minor, tone: "bg-foreground-muted" },
        { label: t("90+"), minor: buckets.daysOver90Minor, tone: "bg-danger" },
      ]
    : [];

  const contact = data.emailProfile?.toRecipients
    ?.split(",")
    .map((address) => address.trim())
    .find(Boolean);
  const creditLimit =
    snapshot?.hasCreditLimit && snapshot.creditLimitMinor > 0
      ? money(snapshot.creditLimitMinor)
      : billing?.creditLimit && Number(billing.creditLimit) > 0
        ? formatCurrency(Number(billing.creditLimit))
        : null;
  const active = data.status === "Active";

  return (
    <>
      <PreviewSource
        kind="customer"
        facts={[
          terms,
          openLoads !== null
            ? t("{0, plural, one {# open load} other {# open loads}}", openLoads)
            : null,
        ]}
      />
      <PreviewHead
        title={data.name}
        subtitle={[data.code, place].filter(Boolean).join(" · ")}
        status={
          <StatePill label={active ? t("Active") : t("Inactive")} state={active ? "ok" : "warn"} />
        }
      />
      <PreviewStats items={stats} />
      {buckets && buckets.totalOpenMinor > 0 && (
        <PreviewSection
          label={t("Aging")}
          aside={t("of {0}", money(buckets.totalOpenMinor))}
          className="gap-2.5"
        >
          <AgingBar buckets={agingBuckets} total={buckets.totalOpenMinor} />
        </PreviewSection>
      )}
      {loads.data && loads.data.rows.length > 0 && (
        <PreviewSection label={t("Open loads")} aside={loads.data.total ?? undefined}>
          <div className="flex flex-col">
            {loads.data.rows.map((load) => (
              <RelatedRow
                key={load.id}
                icon={PALETTE_ENTITY_ICONS.shipment}
                title={load.proNumber ?? load.id}
                mono
                sub={recordShipmentLane(load)}
                note={shipmentStatus(load.status, t)?.label}
                onOpen={() => onNavigate(recordPath("shipment", load.id))}
              />
            ))}
          </div>
        </PreviewSection>
      )}
      <PreviewFields
        rows={[
          contact ? { label: t("Contact"), value: contact } : null,
          creditLimit
            ? { label: t("Credit limit"), value: <span className="font-mono">{creditLimit}</span> }
            : null,
        ]}
      />
    </>
  );
}
