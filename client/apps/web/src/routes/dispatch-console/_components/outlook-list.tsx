import { HoverGlideGroup } from "@/components/hover-glide";
import { usePermission } from "@/hooks/use-permission";
import {
  forecastStanding,
  forecastStandingCopy,
  formatLoads,
  type ForecastStanding,
} from "@/lib/dispatch-forecast";
import type {
  DispatchForecastQuality,
  DispatchFreightWindow,
} from "@/lib/graphql/dispatch-console";
import { dispatchConsoleQueries } from "@/lib/queries/dispatch-console";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { ChevronDownIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import {
  EmptyTable,
  EmptyTableSketch,
  type EmptyTableColumn,
} from "@trenova/shared/components/ui/empty-table";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useNow } from "@trenova/shared/hooks/use-now";
import { formatList, formatNumber, formatRelativeTime } from "@trenova/shared/i18n/format";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDayTimeRange } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useId, useState } from "react";
import { Link } from "react-router";
import { OUTLOOK_GRID_CLASS, OutlookRowsSkeleton } from "./console-skeletons";
import { forecastCustomersSentence, formatCents } from "./cost-vocabulary";

export const OUTLOOK_HOURS = 48;
export const OUTLOOK_LIMIT = 20;
const DISPATCH_CONTROLS_PATH = "/admin/dispatch-controls/";
// Short by more than half a load is a driver the area is likely to miss.
const SHORTFALL_WARNING = 0.5;
const OUTLOOK_CUSTOMERS_SHOWN = 3;

export const OUTLOOK_COLUMNS: readonly EmptyTableColumn[] = [
  { label: "Area" },
  { label: "Window" },
  { label: "Likely loads", numeric: true },
  { label: "Drivers expected", numeric: true },
  { label: "Short by", numeric: true },
  { label: "One more driver", numeric: true },
];

const STANDING_ALERT: Record<Exclude<ForecastStanding, "Ready">, "info" | "warning"> = {
  Unfitted: "info",
  InsufficientHistory: "info",
  NoSkill: "warning",
};

function likelyLoads(freight: DispatchFreightWindow, t: TranslateFn): string {
  return freight.likelyLow === freight.likelyHigh
    ? formatNumber(freight.likelyHigh)
    : t("{0}–{1}", formatNumber(freight.likelyLow), formatNumber(freight.likelyHigh));
}

function usualLoads(freight: DispatchFreightWindow, t: TranslateFn): string {
  const { typicalLow: low, typicalHigh: high } = freight;
  if (low === high) {
    return t(
      "Usually {0, plural, =0 {no loads} one {# load} other {# loads}} in this window.",
      high,
    );
  }
  if (low <= 0) {
    return t("Usually up to {0, plural, one {# load} other {# loads}} in this window.", high);
  }
  return t("Usually {0}–{1} loads in this window.", formatNumber(low), formatNumber(high));
}

function OutlookHeading() {
  const t = useT();

  return (
    <div
      aria-hidden
      className={cn(
        OUTLOOK_GRID_CLASS,
        "bg-muted/40 border-border text-muted-foreground h-7.5 shrink-0 items-center border-b pr-5.5 pl-3.5 text-2xs font-medium",
      )}
    >
      <span>{t("Area")}</span>
      <span>{t("Window")}</span>
      <span className="text-right">{t("Likely loads")}</span>
      <span className="text-right">{t("Drivers expected")}</span>
      <span className="text-right">{t("Short by")}</span>
      <span className="text-right">{t("One more driver")}</span>
    </div>
  );
}

function OutlookRow({
  freight,
  nowSeconds,
}: {
  freight: DispatchFreightWindow;
  nowSeconds: number;
}) {
  const t = useT();
  const detailId = useId();
  const [open, setOpen] = useState(false);
  const short = freight.shortfall > SHORTFALL_WARNING;
  const customers = freight.customers.slice(0, OUTLOOK_CUSTOMERS_SHOWN).map((c) => c.name);
  const customersSentence = forecastCustomersSentence(freight.customers, t);

  return (
    <li className="px-1.5 pt-px">
      <button
        type="button"
        aria-expanded={open}
        aria-controls={detailId}
        data-glide-target
        onClick={() => setOpen((current) => !current)}
        className={cn(
          OUTLOOK_GRID_CLASS,
          "ui-focus-ring relative w-full cursor-pointer items-center rounded-md px-2 py-1.75 text-left text-xs",
          open && "bg-surface-hover",
        )}
      >
        <span className="min-w-0">
          <span className="flex min-w-0 items-center gap-1.5">
            <ChevronDownIcon
              aria-hidden
              className={cn(
                "text-muted-foreground size-3 shrink-0 transition-transform",
                !open && "-rotate-90",
              )}
            />
            <span className="truncate font-medium">{freight.areaLabel}</span>
          </span>
          <span className="text-muted-foreground block truncate pl-4.5 text-2xs">
            {customers.length > 0 ? formatList(customers) : t("No customer stands out")}
          </span>
        </span>
        <span className="min-w-0">
          <span className="block truncate">
            {formatUnixDayTimeRange(freight.windowStart, freight.windowEnd)}
          </span>
          <span className="text-muted-foreground block truncate text-2xs">
            {freight.windowStart > nowSeconds
              ? formatRelativeTime(freight.windowStart - nowSeconds)
              : t("Under way")}
          </span>
        </span>
        <span className="min-w-0 text-right">
          <span className="block truncate font-mono tabular-nums">{likelyLoads(freight, t)}</span>
          <span className="text-muted-foreground block truncate text-2xs">
            {t("{0, plural, other {# booked}}", freight.booked)}
          </span>
        </span>
        <span className="truncate text-right font-mono tabular-nums">
          {formatNumber(freight.driversExpected)}
        </span>
        <span
          className={cn(
            "truncate text-right font-mono tabular-nums",
            short ? "text-warning-foreground" : "text-muted-foreground",
          )}
        >
          {formatLoads(freight.shortfall)}
        </span>
        <span className="truncate text-right font-mono tabular-nums">
          {formatCents(freight.valueCents)}
        </span>
      </button>
      <div
        id={detailId}
        hidden={!open}
        className="animate-expand-in text-muted-foreground flex flex-col gap-0.5 pt-1 pr-2 pb-2 pl-6.5 text-2xs"
      >
        <p className="text-foreground">{usualLoads(freight, t)}</p>
        {customersSentence ? <p>{customersSentence}</p> : null}
        <p>
          {freight.expectedUnbooked < 0.05
            ? t(
                "Nothing more is expected to be booked; one more driver ending here is worth {0} to the plan.",
                formatCents(freight.valueCents),
              )
            : t(
                "{0} still expected to be booked; one more driver ending here is worth {1} to the plan.",
                formatLoads(freight.expectedUnbooked),
                formatCents(freight.valueCents),
              )}
        </p>
      </div>
    </li>
  );
}

function ForecastNotReady({ quality }: { quality: DispatchForecastQuality }) {
  const t = useT();
  const standing = forecastStanding(quality);
  const copy = forecastStandingCopy(quality, t);
  const { allowed: canOpenControls } = usePermission(Resource.DispatchControl, Operation.Read);

  if (standing === "Ready") return null;

  return (
    <div className="px-3.5 py-3">
      <Alert size="sm" variant={STANDING_ALERT[standing]}>
        <AlertTitle>{t("Freight forecast: {0}", copy.title)}</AlertTitle>
        <AlertDescription className="flex flex-col gap-1">
          <span>{copy.description}</span>
          <span>
            {t(
              "The outlook and Anticipate freight use the forecast only once it is Ready, so drivers are not moved for freight it cannot yet predict.",
            )}
          </span>
          {canOpenControls ? (
            <Link
              to={DISPATCH_CONTROLS_PATH}
              className="text-brand ui-focus-ring self-start rounded-sm font-medium"
            >
              {t("See the forecast in Dispatch controls")}
            </Link>
          ) : null}
        </AlertDescription>
      </Alert>
    </div>
  );
}

/**
 * Where freight is likely to appear in the next two days beyond the drivers expected there,
 * in the order the server ranks it (most short of drivers first), with what one more driver
 * ending in each area is worth. Shown only once the forecast has proven itself on the
 * organization's own history; until then it says why, and where to see the forecast.
 */
export function OutlookList() {
  const t = useT();
  const nowSeconds = useNow("minute");
  const query = useQuery(
    dispatchConsoleQueries.freightOutlook({ hours: OUTLOOK_HOURS, limit: OUTLOOK_LIMIT }),
  );
  const outlook = query.data;

  if (query.isPending) {
    return <OutlookRowsSkeleton />;
  }
  if (!outlook) {
    return (
      <ErrorState
        error={query.error}
        layout="sheet"
        sketch={<EmptyTableSketch columns={OUTLOOK_COLUMNS} />}
        title={t("The outlook didn't load")}
        description={t(
          "This isn't an empty outlook: freight may still be coming where drivers are short.",
        )}
        onRetry={() => void query.refetch()}
      />
    );
  }
  if (forecastStanding(outlook.quality) !== "Ready") {
    return <ForecastNotReady quality={outlook.quality} />;
  }
  if (outlook.windows.length === 0) {
    return (
      <EmptyTable
        columns={OUTLOOK_COLUMNS}
        title={t("Drivers cover the freight ahead")}
        description={t("No freight expected beyond your drivers in the next 48 hours.")}
      />
    );
  }

  return (
    <>
      <OutlookHeading />
      <ScrollArea
        maskVariant="card"
        maskHeight={18}
        className="min-h-0 flex-1"
        viewportClassName="overscroll-contain"
      >
        <HoverGlideGroup className="w-full">
          <ul aria-label={t("Freight outlook")} className="flex flex-col pb-2">
            {outlook.windows.map((freight) => (
              <OutlookRow
                key={`${freight.areaLabel}:${freight.windowStart}`}
                freight={freight}
                nowSeconds={nowSeconds}
              />
            ))}
          </ul>
        </HoverGlideGroup>
      </ScrollArea>
    </>
  );
}
