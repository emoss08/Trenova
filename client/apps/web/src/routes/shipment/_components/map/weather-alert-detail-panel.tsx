import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { weatherAlert } from "@/lib/queries/weather-alert";
import { STAGE_META } from "@/lib/shipment-board/stage";
import {
  ACTIVITY_TYPE_COLORS,
  ALERT_CATEGORY_CONFIG,
  SEVERITY_BADGE_MAP,
  type WeatherAlertActivity,
  type WeatherAlertActivityType,
  type WeatherAlertFeature,
} from "@/types/weather-alert";
import { useQuery } from "@tanstack/react-query";
import { formatUnixDateTimeShort, formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { useEtaDelta } from "../board/cells/eta-cell";
import { AlertSwatch, MapCardHeader, MapCardSection } from "./map-card-parts";
import { useMapFocusStore } from "./map-focus-store";
import type { MapShipment } from "./map-shipment-model";
import type { AlertHit } from "./weather-alert-matching";

const ACTIVITY_LABELS = defineLabels({
  issued: "Issued",
  updated: "Updated",
  expired: "Expired",
  cancelled: "Cancelled",
});

function formatUnixTimestamp(unix: number | null | undefined): string {
  if (!unix) return "—";
  return formatUnixInUserTimezone(
    unix,
    {
      month: "short",
      day: "numeric",
      hour: "numeric",
      minute: "2-digit",
      timeZoneName: "short",
    },
    "—",
  );
}

/**
 * The open alert: what it is, which of our shipments it touches, what to do,
 * the full text, and its history.
 */
export function WeatherAlertDetailCard({
  feature,
  hits,
  modelsById,
  onClose,
}: {
  feature: WeatherAlertFeature;
  hits: readonly AlertHit[];
  modelsById: ReadonlyMap<string, MapShipment>;
  onClose: () => void;
}) {
  const t = useT();
  const props = feature.properties;
  const config = ALERT_CATEGORY_CONFIG[props.alertCategory] ?? ALERT_CATEGORY_CONFIG.other;
  const [fullText, setFullText] = useState(false);
  const { data, isLoading } = useQuery({
    ...weatherAlert.detail(props.id),
    enabled: !!props.id,
  });

  const affected = hits
    .map((hit) => ({ hit, model: modelsById.get(hit.shipmentId) }))
    .filter((entry): entry is { hit: AlertHit; model: MapShipment } => !!entry.model)
    .sort(
      (a, b) =>
        Number(b.model.stage === "Late") - Number(a.model.stage === "Late") ||
        Number(b.hit.at === "stop") - Number(a.hit.at === "stop"),
    );

  return (
    <>
      <MapCardHeader
        leading={<AlertSwatch color={config.stroke} />}
        title={<span className="text-sm font-semibold">{props.event}</span>}
        badge={
          props.severity ? (
            <Badge variant={SEVERITY_BADGE_MAP[props.severity] ?? "neutral"}>
              {props.severity}
            </Badge>
          ) : null
        }
        subtitle={props.headline}
        onClose={onClose}
      />
      <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 px-3 py-2.5 text-xs">
        {props.areaDesc ? (
          <>
            <dt className="text-muted-foreground">{t("Area")}</dt>
            <dd className="min-w-0">{props.areaDesc}</dd>
          </>
        ) : null}
        <dt className="text-muted-foreground">{t("Effective")}</dt>
        <dd className="tabular-nums">{formatUnixTimestamp(props.effective)}</dd>
        <dt className="text-muted-foreground">{t("Expires")}</dt>
        <dd className="tabular-nums">{formatUnixTimestamp(props.expires)}</dd>
        {props.urgency || props.certainty ? (
          <>
            <dt className="text-muted-foreground">{t("Urgency")}</dt>
            <dd>{[props.urgency, props.certainty].filter(Boolean).join(" · ")}</dd>
          </>
        ) : null}
      </dl>

      <MapCardSection
        title={t("Affected shipments")}
        aside={<span className="font-mono tabular-nums">{affected.length}</span>}
      >
        {affected.length > 0 ? (
          <ul className="flex flex-col gap-1.5">
            {affected.map(({ hit, model }) => (
              <li key={model.id}>
                <AffectedShipmentRow model={model} at={hit.at} />
              </li>
            ))}
          </ul>
        ) : (
          <p className="text-muted-foreground text-xs">
            {t("No active shipment is inside this area or routed through it.")}
          </p>
        )}
      </MapCardSection>

      {props.instruction ? (
        <MapCardSection title={t("Recommended action")}>
          <p className="text-xs leading-relaxed">{props.instruction}</p>
        </MapCardSection>
      ) : null}

      {props.description ? (
        <MapCardSection title={t("Details")}>
          <p
            className={cn(
              "text-xs leading-relaxed whitespace-pre-line",
              !fullText && "line-clamp-3",
            )}
          >
            {props.description}
          </p>
          <button
            type="button"
            className="text-brand ui-focus-ring self-start rounded-sm text-xs font-medium"
            onClick={() => setFullText((open) => !open)}
          >
            {fullText ? t("Show less") : t("Show full text")}
          </button>
        </MapCardSection>
      ) : null}

      <MapCardSection title={t("Activity")}>
        {isLoading ? (
          <p className="text-muted-foreground text-xs">{t("Loading activity...")}</p>
        ) : (
          <ActivityList activities={data?.activities ?? []} />
        )}
      </MapCardSection>
    </>
  );
}

function AffectedShipmentRow({ model, at }: { model: MapShipment; at: AlertHit["at"] }) {
  const t = useT();
  const toggleShipment = useMapFocusStore((s) => s.toggleShipment);
  const setRailTab = useMapFocusStore((s) => s.setRailTab);
  const delta = useEtaDelta(model.shipment);
  const lane = model.lane;

  return (
    <button
      type="button"
      className="border-subtle hover:bg-surface-hover ui-focus-ring flex w-full flex-col gap-0.5 rounded-md border px-2 py-1.5 text-left"
      onClick={() => {
        toggleShipment(model.id);
        setRailTab("shipments");
      }}
    >
      <span className="flex min-w-0 items-center gap-1.5">
        <span
          aria-hidden
          className={cn("size-2 shrink-0 rounded-full", STAGE_META[model.stage].swatchClassName)}
        />
        <span className="font-mono text-xs font-medium tabular-nums">
          {model.shipment.proNumber || "—"}
        </span>
        {model.stage === "Late" && delta ? (
          <span className={cn("font-mono text-xs tabular-nums", delta.tone)}>{delta.text}</span>
        ) : null}
        <span className="text-muted-foreground ml-auto shrink-0 text-xs">
          {at === "stop" ? t("Next stop is inside") : t("On the route ahead")}
        </span>
      </span>
      {lane ? (
        <span className="truncate text-xs">
          {t("{0} to {1}", lane.originLabel, lane.destinationLabel)}
        </span>
      ) : null}
    </button>
  );
}

function ActivityList({ activities }: { activities: WeatherAlertActivity[] }) {
  const t = useT();

  if (activities.length === 0) {
    return <p className="text-muted-foreground text-xs">{t("No activity recorded")}</p>;
  }

  return (
    <ul className="flex flex-col gap-1 text-xs">
      {activities.map((activity) => {
        const type = activity.activityType as WeatherAlertActivityType;
        return (
          <li key={activity.id} className="flex items-center gap-2">
            <span
              aria-hidden
              className="size-2 shrink-0 rounded-full"
              style={{ backgroundColor: ACTIVITY_TYPE_COLORS[type] }}
            />
            {ACTIVITY_LABELS[type] ?? activity.activityType}
            <span className="text-muted-foreground ml-auto font-mono tabular-nums">
              {formatUnixDateTimeShort(activity.timestamp)}
            </span>
          </li>
        );
      })}
    </ul>
  );
}
