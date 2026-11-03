import { resolveCoverage } from "@/lib/shipment-board/coverage";
import { STAGE_META } from "@/lib/shipment-board/stage";
import { STOP_TYPE_LABELS } from "@/lib/shipment-board/stop-type-labels";
import { getTotalMiles, shipmentPanelPath } from "@/lib/shipment-utils";
import {
  ALERT_CATEGORY_CONFIG,
  SEVERITY_BADGE_MAP,
  type WeatherAlertFeature,
} from "@/types/weather-alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatMinutesSpan } from "@trenova/shared/lib/date";
import { useNavigate } from "react-router";
import { AlertSwatch, formatMapTime, MapCardHeader, MapCardSection } from "./map-card-parts";
import { useMapFocusStore } from "./map-focus-store";
import { placeLabel, type MapShipment } from "./map-shipment-model";
import type { StopWithCoord } from "./shipment-route-coordinates";
import type { AlertHit } from "./weather-alert-matching";

/** The selected shipment: its stops as the map numbers them, who has it, and weather on its way. */
export function ShipmentFocusCard({
  model,
  hits,
  featuresById,
  onClose,
}: {
  model: MapShipment;
  hits: readonly AlertHit[];
  featuresById: ReadonlyMap<string, WeatherAlertFeature>;
  onClose: () => void;
}) {
  const t = useT();
  const navigate = useNavigate();
  const toggleAlert = useMapFocusStore((s) => s.toggleAlert);
  const setRailTab = useMapFocusStore((s) => s.setRailTab);
  const { shipment } = model;
  const meta = STAGE_META[model.stage];
  const coverage = resolveCoverage(shipment);
  const miles = Math.round(getTotalMiles(shipment));
  const hauler =
    coverage.kind === "driver"
      ? [coverage.name, coverage.detail].filter(Boolean).join(" · ")
      : coverage.kind === "carrier"
        ? coverage.name
        : coverage.kind === "tendered"
          ? t("Tendered, waiting on the carrier")
          : t("Not covered");

  return (
    <>
      <MapCardHeader
        title={
          <span className="font-mono text-sm font-semibold tabular-nums">
            {shipment.proNumber || "—"}
          </span>
        }
        badge={<Badge variant={meta.tone}>{t(meta.label)}</Badge>}
        subtitle={shipment.customer?.name}
        onClose={onClose}
      />
      <ol className="flex flex-col px-3 pt-3">
        {model.stops.map((entry, index) => (
          <StopRow
            key={entry.stop.id ?? index}
            entry={entry}
            number={index + 1}
            last={index === model.stops.length - 1}
            colorVar={STAGE_META[model.stage].colorVar}
            model={model}
          />
        ))}
      </ol>
      <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 px-3 pb-3 text-xs">
        <dt className="text-muted-foreground">
          {coverage.kind === "driver" ? t("Driver") : t("Carrier")}
        </dt>
        <dd className="min-w-0 truncate">{hauler}</dd>
        {miles > 0 ? (
          <>
            <dt className="text-muted-foreground">{t("Distance")}</dt>
            <dd className="font-mono tabular-nums">{t("{0} mi", miles.toLocaleString())}</dd>
          </>
        ) : null}
      </dl>
      {hits.length > 0 ? (
        <MapCardSection title={t("Weather on route")}>
          <ul className="flex flex-col gap-1.5">
            {hits.map((hit) => {
              const feature = featuresById.get(hit.alertId);
              if (!feature) return null;
              const props = feature.properties;
              const config =
                ALERT_CATEGORY_CONFIG[props.alertCategory] ?? ALERT_CATEGORY_CONFIG.other;
              return (
                <li key={hit.alertId}>
                  <button
                    type="button"
                    className="border-subtle hover:bg-surface-hover ui-focus-ring flex w-full flex-col gap-0.5 rounded-md border px-2 py-1.5 text-left"
                    onClick={() => {
                      toggleAlert(hit.alertId);
                      setRailTab("alerts");
                    }}
                  >
                    <span className="flex min-w-0 items-center gap-1.5">
                      <AlertSwatch color={config.stroke} />
                      <span className="min-w-0 truncate text-sm font-medium">{props.event}</span>
                      {props.severity ? (
                        <Badge variant={SEVERITY_BADGE_MAP[props.severity] ?? "neutral"}>
                          {props.severity}
                        </Badge>
                      ) : null}
                      <span className="text-muted-foreground ml-auto shrink-0 text-xs">
                        {hit.at === "stop" ? t("Next stop is inside") : t("On the route ahead")}
                      </span>
                    </span>
                    {props.headline ? (
                      <span className="text-muted-foreground truncate text-xs">
                        {props.headline}
                      </span>
                    ) : null}
                  </button>
                </li>
              );
            })}
          </ul>
        </MapCardSection>
      ) : null}
      <footer className="border-subtle flex items-center gap-2 border-t px-3 py-2.5">
        <Button size="sm" onClick={() => void navigate(shipmentPanelPath(model.id))}>
          {t("Open shipment")}
        </Button>
      </footer>
    </>
  );
}

function StopRow({
  entry,
  number,
  last,
  colorVar,
  model,
}: {
  entry: StopWithCoord;
  number: number;
  last: boolean;
  colorVar: string;
  model: MapShipment;
}) {
  const t = useT();
  const { stop } = entry;
  const done = stop.status === "Completed";
  const color = `var(${colorVar})`;
  const stopEta = model.shipment.eta?.stops?.find((eta) => eta.stopId === stop.id) ?? null;
  const place = placeLabel(stop.location) ?? stop.location?.name ?? "—";

  return (
    <li className="relative flex gap-2.5 pb-3">
      {!last ? (
        <span aria-hidden className="bg-border absolute top-5 bottom-0 left-2 w-px" />
      ) : null}
      <span
        aria-hidden
        className="relative grid size-4 shrink-0 place-items-center rounded-full font-mono text-2xs font-semibold tabular-nums"
        style={{
          background: done ? color : "var(--card)",
          color: done
            ? colorVar === "--warning"
              ? "var(--warning-on-solid)"
              : "var(--foreground-on-solid)"
            : color,
          border: `2px solid ${color}`,
        }}
      >
        {number}
      </span>
      <div className="flex min-w-0 flex-col">
        <span className="truncate text-sm font-medium">
          {place}{" "}
          <span className="text-muted-foreground text-xs font-normal">
            {STOP_TYPE_LABELS[stop.type]}
          </span>
        </span>
        <span className="text-muted-foreground flex flex-wrap gap-x-1.5 font-mono text-xs tabular-nums">
          {stop.status === "Canceled" ? (
            <span>{t("Canceled")}</span>
          ) : done ? (
            <span>
              {stop.actualDeparture
                ? t("Departed {0}", formatMapTime(stop.actualDeparture))
                : stop.actualArrival
                  ? t("Arrived {0}", formatMapTime(stop.actualArrival))
                  : t("Done")}
            </span>
          ) : (
            <>
              {stop.scheduledWindowStart ? (
                <span>{t("Appt {0}", formatMapTime(stop.scheduledWindowStart))}</span>
              ) : null}
              {stopEta?.estimatedArrival && !stopEta.arrived ? (
                <span>{t("ETA {0}", formatMapTime(stopEta.estimatedArrival))}</span>
              ) : null}
              {stopEta?.slackMinutes != null && stopEta.slackMinutes < 0 ? (
                <span className="text-danger font-medium">
                  {t("{0} late", formatMinutesSpan(-stopEta.slackMinutes))}
                </span>
              ) : null}
            </>
          )}
        </span>
      </div>
    </li>
  );
}
