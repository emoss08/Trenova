import { LinkedRecordLink } from "@/components/linked-record-link";
import { recordPath } from "@/config/record-links";
import type { BillingQueueShipment } from "@/lib/graphql/billing-queue";
import { billingQueue } from "@/lib/queries/billing-queue";
import { dwellText, shipmentMiles, stopRows, type StopRow } from "@/lib/billing-queue-stops";
import { Lock01Icon } from "@trenova/shared/components/icons";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { useCanReadKind } from "@trenova/shared/hooks/use-can-read-kind";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { humanizeTariffName } from "@trenova/shared/lib/tariff-inputs";
import { formatTariffValue } from "@trenova/shared/lib/tariff-value";
import { cn } from "@trenova/shared/lib/utils";
import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";
import type { TariffWireValue } from "@trenova/shared/types/tariff";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router";
import { BillingQueueRatedBy, ratedByName } from "./billing-queue-rated-by";
import { BillingQueueShipmentTabSkeleton, STOP_GRID } from "./billing-queue-shipment-tab-skeleton";

const DAY = { month: "short", day: "numeric" } as const;
const TIME = { hour: "numeric", minute: "2-digit" } as const;
const DATE_TIME = { month: "short", day: "numeric", hour: "numeric", minute: "2-digit" } as const;
const EMPTY = "—";

/**
 * The shipment as billing checks the charges against it, read-only: each stop with its
 * scheduled window, when the driver arrived and left and the time on site; the load's
 * references, equipment, driver and freight; and how it was rated. The shipment is read
 * when the tab opens. Changing any of it happens on the shipment.
 */
export default function BillingQueueShipmentTab({ item }: { item: BillingQueueItem }) {
  const t = useT();
  const canRead = useCanReadKind("shipment");
  const shipmentId = item.shipmentId ?? "";
  const { data, isPending, isError } = useQuery({
    ...billingQueue.shipment(shipmentId),
    enabled: Boolean(shipmentId) && canRead,
  });

  if (!canRead) {
    return (
      <div
        className="text-foreground-muted flex flex-col items-center gap-2 px-4 py-12 text-center text-sm"
        data-testid="billing-queue-shipment-locked"
      >
        <span className="border-border flex size-8 items-center justify-center rounded-full border">
          <Lock01Icon className="size-3.5" aria-hidden />
        </span>
        <p className="text-foreground font-medium">{t("You don't have access to shipments")}</p>
        <p className="max-w-90">
          {t(
            "The route, PRO and charges above are still yours to bill. An administrator can grant shipment access to see the stops and load.",
          )}
        </p>
      </div>
    );
  }
  if (isPending) {
    return <BillingQueueShipmentTabSkeleton />;
  }
  if (isError || !data) {
    return (
      <p className="text-foreground-muted py-12 text-center text-sm" role="alert">
        {t("The shipment couldn't be loaded. Open the tab again to retry.")}
      </p>
    );
  }

  const edit = recordPath("shipment", data.id);

  return (
    <div className="flex flex-col gap-5.5 text-sm" data-testid="billing-queue-shipment-tab">
      <Stops shipment={data} editPath={edit} />
      <Load item={item} shipment={data} editPath={edit} />
      <Rated item={item} shipment={data} />
    </div>
  );
}

function SectionHeader({
  title,
  detail,
  editPath,
}: {
  title: string;
  detail?: string;
  editPath?: string;
}) {
  const t = useT();

  return (
    <div className="flex items-baseline gap-2">
      <h3 className="text-foreground text-sm font-semibold">{title}</h3>
      {detail ? <span className="text-foreground-muted text-xs">{detail}</span> : null}
      <span className="flex-1" />
      {editPath ? (
        <Link
          to={editPath}
          className="text-brand ui-focus-ring rounded-xs text-xs font-medium hover:underline"
        >
          {t("Edit on the shipment")}
        </Link>
      ) : null}
    </div>
  );
}

function Stops({ shipment, editPath }: { shipment: BillingQueueShipment; editPath: string }) {
  const t = useT();
  const rows = stopRows(shipment.moves);

  return (
    <section aria-label={t("Stops")} className="flex flex-col gap-2">
      <SectionHeader
        title={t("Stops")}
        detail={t(
          "{0, plural, one {# stop} other {# stops}} · scheduled against what happened",
          rows.length,
        )}
        editPath={editPath}
      />
      {rows.length === 0 ? (
        <p className="text-foreground-muted border-border rounded-md border px-3 py-4 text-xs">
          {t("The shipment has no stops.")}
        </p>
      ) : (
        <div role="table" aria-label={t("Stops")} className="border-border rounded-md border">
          <div
            role="row"
            className={cn(
              STOP_GRID,
              "border-border text-foreground-subtle h-8 border-b px-3 text-xs font-medium",
            )}
          >
            <span role="columnheader" />
            <span role="columnheader">{t("Stop")}</span>
            <span role="columnheader">{t("Scheduled")}</span>
            <span role="columnheader">{t("Arrived → departed")}</span>
            <span role="columnheader" className="text-right">
              {t("On site")}
            </span>
          </div>
          {rows.map((row, index) => (
            <StopLine key={row.key} row={row} last={index === rows.length - 1} />
          ))}
        </div>
      )}
      <p className="text-foreground-muted text-xs">{t("Times are in your timezone.")}</p>
    </section>
  );
}

function StopLine({ row, last }: { row: StopRow; last: boolean }) {
  const t = useT();
  const pickup = row.type === "Pickup" || row.type === "SplitPickup";

  return (
    <div
      role="row"
      className={cn(STOP_GRID, "px-3 py-2.5", !last && "border-border border-b")}
      data-testid="billing-queue-stop"
    >
      <span
        role="img"
        aria-label={stopTypeLabel(row.type, t)}
        className={cn(
          "flex size-5 items-center justify-center rounded-full text-2xs font-semibold",
          pickup
            ? "bg-info-subtle text-info-foreground"
            : "bg-success-subtle text-success-foreground",
        )}
      >
        {pickup ? t("P") : t("D")}
      </span>
      <span className="flex min-w-0 flex-col gap-px">
        <span className="text-foreground truncate font-medium">{row.name}</span>
        <span className="text-foreground-muted truncate text-xs">
          {t("{0} · {1}", stopTypeLabel(row.type, t), row.place)}
        </span>
      </span>
      <span className="text-foreground-muted font-mono text-xs">{scheduledText(row, t)}</span>
      <span className="text-foreground font-mono text-xs">{actualText(row, t)}</span>
      <span className="text-foreground text-right font-mono text-xs font-medium">
        {row.onSite != null ? dwellText(row.onSite, t) : EMPTY}
      </span>
    </div>
  );
}

function Load({
  item,
  shipment,
  editPath,
}: {
  item: BillingQueueItem;
  shipment: BillingQueueShipment;
  editPath: string;
}) {
  const t = useT();
  const assignment = [...shipment.moves]
    .sort((a, b) => a.sequence - b.sequence)
    .map((move) => move.assignment)
    .find(Boolean);
  const miles = shipmentMiles(shipment.moves);
  const commodities = shipment.commodities
    .map((entry) => entry.commodity?.name)
    .filter(Boolean)
    .join(", ");
  const equipment = [assignment?.tractor?.code, assignment?.trailer?.code]
    .filter(Boolean)
    .join(" / ");
  const temperature =
    shipment.temperatureMin != null && shipment.temperatureMax != null
      ? t("{0}°F to {1}°F", shipment.temperatureMin, shipment.temperatureMax)
      : shipment.temperatureMin != null
        ? t("Above {0}°F", shipment.temperatureMin)
        : shipment.temperatureMax != null
          ? t("Below {0}°F", shipment.temperatureMax)
          : null;

  return (
    <section aria-label={t("Load")} className="flex flex-col gap-2">
      <SectionHeader title={t("Load")} editPath={editPath} />
      <DescriptionList
        columns={3}
        className="border-border rounded-md border px-3.5 py-3"
        data-testid="billing-queue-load"
      >
        <DescriptionItem label={t("BOL")} valueClassName="font-mono">
          {shipment.bol || EMPTY}
        </DescriptionItem>
        <DescriptionItem label={t("Customer PO")} valueClassName="font-mono">
          {item.poNumber || EMPTY}
        </DescriptionItem>
        <DescriptionItem label={t("Customer reference")} valueClassName="font-mono">
          {shipment.externalReference || EMPTY}
        </DescriptionItem>
        <DescriptionItem label={t("Customer")}>
          {shipment.customer ? (
            <LinkedRecordLink
              kind="customer"
              id={shipment.customer.id}
              className="text-brand ui-focus-ring rounded-xs hover:underline"
            >
              {shipment.customer.name}
            </LinkedRecordLink>
          ) : (
            EMPTY
          )}
        </DescriptionItem>
        <DescriptionItem label={t("Driver")}>
          {assignment?.primaryWorker?.wholeName || EMPTY}
        </DescriptionItem>
        <DescriptionItem label={t("Tractor / trailer")} valueClassName="font-mono">
          {equipment || EMPTY}
        </DescriptionItem>
        <DescriptionItem label={t("Weight")} numeric valueClassName="font-mono">
          {shipment.weight != null ? t("{0} lb", shipment.weight.toLocaleString()) : EMPTY}
        </DescriptionItem>
        <DescriptionItem label={t("Pieces")} numeric>
          {shipment.pieces != null ? shipment.pieces.toLocaleString() : EMPTY}
        </DescriptionItem>
        <DescriptionItem label={t("Commodity")}>{commodities || EMPTY}</DescriptionItem>
        <DescriptionItem label={t("Miles")} numeric valueClassName="font-mono">
          {miles.loaded || miles.empty
            ? t(
                "{0} loaded · {1} empty",
                miles.loaded.toLocaleString(),
                miles.empty.toLocaleString(),
              )
            : EMPTY}
        </DescriptionItem>
        {temperature ? (
          <DescriptionItem label={t("Temperature")}>{temperature}</DescriptionItem>
        ) : null}
      </DescriptionList>
    </section>
  );
}

function Rated({ item, shipment }: { item: BillingQueueItem; shipment: BillingQueueShipment }) {
  const t = useT();
  const rating = shipment.ratingDetail;
  if (!item.ratedBy && !rating) return null;

  const inputs = Object.entries((rating?.inputs as Record<string, unknown> | null) ?? {}).slice(
    0,
    3,
  );
  const terms = [
    ...(rating?.ruleLabel ? [t("Rule {0}", rating.ruleLabel)] : []),
    ...inputs.map(([name, value]) =>
      t("{0} {1}", humanizeTariffName(name), formatTariffValue(value as TariffWireValue, t)),
    ),
    ...(rating?.ratedAt
      ? [t("rated {0}", formatUnixInUserTimezone(rating.ratedAt, DATE_TIME))]
      : []),
  ];

  return (
    <section aria-label={t("How it was rated")} className="flex flex-col gap-2">
      <SectionHeader title={t("How it was rated")} />
      <div className="border-border flex items-center gap-2.5 rounded-md border px-3.5 py-2.5">
        <span className="flex min-w-0 flex-1 flex-col gap-0.5">
          <span className="text-foreground-muted">
            {item.ratedBy
              ? t("Rated by {0}", ratedByName(item.ratedBy, t))
              : t("Rated by {0}", rating?.formulaTemplateName || EMPTY)}
          </span>
          {terms.length > 0 ? (
            <span className="text-foreground-muted truncate text-xs">{terms.join(" · ")}</span>
          ) : null}
        </span>
        {item.ratedBy ? (
          <BillingQueueRatedBy item={item} ratedBy={item.ratedBy} trigger="terms" />
        ) : null}
      </div>
    </section>
  );
}

function stopTypeLabel(type: string, t: TranslateFn): string {
  switch (type) {
    case "Pickup":
      return t("Pickup");
    case "SplitPickup":
      return t("Split pickup");
    case "Delivery":
      return t("Delivery");
    case "SplitDelivery":
      return t("Split delivery");
    default:
      return type;
  }
}

function scheduledText(row: StopRow, t: TranslateFn): string {
  const day = formatUnixInUserTimezone(row.scheduledStart, DAY);
  const start = formatUnixInUserTimezone(row.scheduledStart, TIME);
  if (row.scheduledEnd == null || row.scheduledEnd === row.scheduledStart) {
    return t("{0} · {1}", day, start);
  }
  const endDay = formatUnixInUserTimezone(row.scheduledEnd, DAY);
  const end = formatUnixInUserTimezone(row.scheduledEnd, TIME);

  return endDay === day
    ? t("{0} · {1}–{2}", day, start, end)
    : t("{0} {1} – {2} {3}", day, start, endDay, end);
}

function actualText(row: StopRow, t: TranslateFn): string {
  if (row.arrival == null) return t("Not arrived");
  const arrived = formatUnixInUserTimezone(row.arrival, TIME);
  if (row.departure == null) return t("Arrived {0}", arrived);

  return t("{0} → {1}", arrived, formatUnixInUserTimezone(row.departure, TIME));
}
