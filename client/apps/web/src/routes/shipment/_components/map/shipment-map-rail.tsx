import { resolveCoverage } from "@/lib/shipment-board/coverage";
import { STAGE_META } from "@/lib/shipment-board/stage";
import {
  ALERT_CATEGORY_CONFIG,
  SEVERITY_BADGE_MAP,
  type WeatherAlertCategory,
  type WeatherAlertFeature,
} from "@/types/weather-alert";
import { SearchMdIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Input } from "@trenova/shared/components/ui/input";
import { Tabs, TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { useEtaDelta } from "../board/cells/eta-cell";
import { useShipmentBoardStore } from "../board/store";
import { AlertSwatch, formatMapTime } from "./map-card-parts";
import { useMapFocusStore, type RailTab } from "./map-focus-store";
import {
  groupByStage,
  matchesMapSearch,
  nextOpenStop,
  placeLabel,
  type MapShipment,
} from "./map-shipment-model";
import { severityRank, type AlertHit, type AlertMatches } from "./weather-alert-matching";

/**
 * The list beside the map: shipments grouped by stage, most urgent first, and,
 * while public alerts are on, the alerts with how many of our loads each one
 * touches. Hovering a row lights its mark; clicking opens it in the card.
 */
export function ShipmentMapRail({
  models,
  matches,
  alertsOn,
  features,
  allFeatures,
  hiddenAlertCategories,
  onToggleAlertCategory,
  className,
}: {
  models: readonly MapShipment[];
  matches: AlertMatches;
  alertsOn: boolean;
  /** Alerts in the categories that are switched on. */
  features: readonly WeatherAlertFeature[];
  /** Every alert, for the category counts. */
  allFeatures: readonly WeatherAlertFeature[];
  hiddenAlertCategories: readonly WeatherAlertCategory[];
  onToggleAlertCategory: (category: WeatherAlertCategory) => void;
  className?: string;
}) {
  const t = useT();
  const storedTab = useMapFocusStore((s) => s.railTab);
  const setRailTab = useMapFocusStore((s) => s.setRailTab);
  const scope = useMapFocusStore((s) => s.scope);
  const setScope = useMapFocusStore((s) => s.setScope);
  const [query, setQuery] = useState("");
  const tab: RailTab = alertsOn ? storedTab : "shipments";

  const scoped = scope ? models.filter((m) => scope.ids.includes(m.id)) : models;
  const listed = scoped.filter((m) => matchesMapSearch(m, query));

  return (
    <aside
      aria-label={t("Shipments and alerts")}
      className={cn("border-subtle min-h-0 w-80 shrink-0 flex-col border-r", className)}
    >
      {alertsOn ? (
        <Tabs
          value={tab}
          onValueChange={(value) => {
            setRailTab(value as RailTab);
            setQuery("");
          }}
          className="border-subtle shrink-0 gap-0 border-b px-2"
        >
          <TabsList variant="underline">
            <TabsTab value="shipments">
              {t("Shipments")}
              <span className="text-muted-foreground font-mono text-xs tabular-nums">
                {scoped.length}
              </span>
            </TabsTab>
            <TabsTab value="alerts">
              {t("Weather alerts")}
              <span className="text-muted-foreground font-mono text-xs tabular-nums">
                {features.length}
              </span>
            </TabsTab>
          </TabsList>
        </Tabs>
      ) : null}

      <div className="border-subtle flex shrink-0 flex-col gap-2 border-b p-2">
        <Input
          type="search"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder={
            tab === "alerts" ? t("Search alerts or areas") : t("Search PRO, customer or city")
          }
          aria-label={tab === "alerts" ? t("Search alerts") : t("Search shipments")}
          leftElement={<SearchMdIcon className="text-muted-foreground size-3.5" />}
          autoComplete="off"
        />
        {tab === "alerts" ? (
          <AlertCategoryChips
            allFeatures={allFeatures}
            hidden={hiddenAlertCategories}
            onToggle={onToggleAlertCategory}
          />
        ) : scope ? (
          <div className="bg-brand-subtle text-brand-subtle-foreground flex items-center gap-2 rounded-md px-2 py-1 text-xs">
            <span className="min-w-0 truncate">{scope.label}</span>
            <button
              type="button"
              className="ui-focus-ring ml-auto shrink-0 rounded-sm font-medium"
              onClick={() => setScope(null)}
            >
              {t("Show all")}
            </button>
          </div>
        ) : null}
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain">
        {tab === "alerts" ? (
          <AlertList features={features} matches={matches} query={query} />
        ) : (
          <ShipmentList models={listed} matches={matches} alertsOn={alertsOn} features={features} />
        )}
      </div>
    </aside>
  );
}

function GroupHeader({
  swatchClassName,
  label,
  count,
}: {
  swatchClassName?: string;
  label: string;
  count: number;
}) {
  return (
    <div className="bg-sunken border-subtle text-muted-foreground sticky top-0 z-1 flex items-center gap-1.5 border-b px-3 py-1.5 text-xs font-medium">
      {swatchClassName ? (
        <span aria-hidden className={cn("size-2 rounded-full", swatchClassName)} />
      ) : null}
      {label}
      <span className="ml-auto font-mono tabular-nums">{count}</span>
    </div>
  );
}

function ShipmentList({
  models,
  matches,
  alertsOn,
  features,
}: {
  models: readonly MapShipment[];
  matches: AlertMatches;
  alertsOn: boolean;
  features: readonly WeatherAlertFeature[];
}) {
  const t = useT();
  const groups = groupByStage(models);

  if (groups.length === 0) {
    return (
      <p className="text-muted-foreground px-4 py-8 text-center text-sm">
        {t("No shipments match. Clear the search or turn a stage back on.")}
      </p>
    );
  }

  return (
    <>
      {groups.map((group) => (
        <section key={group.stage} aria-label={t(STAGE_META[group.stage].label)}>
          <GroupHeader
            swatchClassName={STAGE_META[group.stage].swatchClassName}
            label={t(STAGE_META[group.stage].label)}
            count={group.items.length}
          />
          <ul>
            {group.items.map((model) => (
              <li key={model.id}>
                <ShipmentRow
                  model={model}
                  hits={alertsOn ? matches.byShipment.get(model.id) : undefined}
                  features={features}
                />
              </li>
            ))}
          </ul>
        </section>
      ))}
    </>
  );
}

function ShipmentRow({
  model,
  hits,
  features,
}: {
  model: MapShipment;
  hits: readonly AlertHit[] | undefined;
  features: readonly WeatherAlertFeature[];
}) {
  const t = useT();
  const highlighted = useShipmentBoardStore((s) => s.highlightId === model.id);
  const setHighlightId = useShipmentBoardStore((s) => s.setHighlightId);
  const selected = useMapFocusStore((s) => s.focus?.kind === "shipment" && s.focus.id === model.id);
  const toggleShipment = useMapFocusStore((s) => s.toggleShipment);
  const { shipment } = model;
  const coverage = resolveCoverage(shipment);
  const next = nextOpenStop(model);
  const extraStops = Math.max(0, model.stops.length - 2);
  const firstHit = hits?.[0];
  const firstHitFeature = firstHit
    ? features.find((f) => f.properties.id === firstHit.alertId)
    : undefined;

  return (
    <button
      type="button"
      aria-pressed={selected}
      className={cn(
        "border-subtle ui-focus-ring flex w-full flex-col gap-0.5 border-b px-3 py-2 text-left",
        selected
          ? model.stage === "Late"
            ? "bg-danger-subtle"
            : "bg-surface-selected"
          : highlighted
            ? "bg-surface-hover"
            : "hover:bg-surface-hover",
      )}
      onMouseEnter={() => setHighlightId(model.id)}
      onMouseLeave={() => {
        if (useShipmentBoardStore.getState().highlightId === model.id) setHighlightId(null);
      }}
      onFocus={() => setHighlightId(model.id)}
      onClick={() => toggleShipment(model.id)}
    >
      <span className="flex min-w-0 items-center gap-2">
        <span className="font-mono text-xs font-medium tabular-nums">
          {shipment.proNumber || "—"}
        </span>
        <span className="text-muted-foreground min-w-0 flex-1 truncate text-xs">
          {shipment.customer?.name}
        </span>
        <RowTiming model={model} />
      </span>
      <span className="truncate text-sm font-medium">
        {model.lane ? t("{0} to {1}", model.lane.originLabel, model.lane.destinationLabel) : "—"}
        {extraStops > 0 ? (
          <span className="text-muted-foreground ml-1.5 text-xs font-normal">
            {t("{0, plural, one {+# stop} other {+# stops}}", extraStops)}
          </span>
        ) : null}
      </span>
      <span className="text-muted-foreground truncate text-xs">
        {[
          next && model.stage !== "Scheduled" && model.stage !== "NeedsCoverage"
            ? t("Next: {0}", placeLabel(next.stop.location) ?? "—")
            : null,
          coverage.kind === "driver" || coverage.kind === "carrier"
            ? coverage.name
            : coverage.kind === "tendered"
              ? t("Tendered")
              : t("Not covered"),
          coverage.kind === "driver" ? coverage.detail : null,
        ]
          .filter(Boolean)
          .join(" · ")}
      </span>
      {firstHit && firstHitFeature ? (
        <span className="flex min-w-0 items-center gap-1.5 text-xs">
          <AlertSwatch
            color={
              (
                ALERT_CATEGORY_CONFIG[firstHitFeature.properties.alertCategory] ??
                ALERT_CATEGORY_CONFIG.other
              ).stroke
            }
            className="size-2 shrink-0 rounded-sm"
          />
          <span className="min-w-0 truncate">
            {firstHit.at === "stop"
              ? t("{0} at the next stop", firstHitFeature.properties.event)
              : t("{0} on the route ahead", firstHitFeature.properties.event)}
          </span>
          {hits && hits.length > 1 ? (
            <span className="text-muted-foreground font-mono tabular-nums">
              {t("+{0}", hits.length - 1)}
            </span>
          ) : null}
        </span>
      ) : null}
    </button>
  );
}

/** The one timing a row leads with: how late or early a moving load runs, or when it picks up. */
function RowTiming({ model }: { model: MapShipment }) {
  const t = useT();
  const delta = useEtaDelta(model.shipment);
  const { shipment } = model;

  if (model.stage === "Late" || model.stage === "Moving") {
    if (!delta) return null;
    return (
      <span className={cn("shrink-0 font-mono text-xs tabular-nums", delta.tone)}>
        {delta.text}
      </span>
    );
  }
  if (model.stage === "Delivered") {
    return shipment.actualDeliveryDate ? (
      <span className="text-muted-foreground shrink-0 font-mono text-xs tabular-nums">
        {formatMapTime(shipment.actualDeliveryDate)}
      </span>
    ) : null;
  }
  const pickup = model.stops[0]?.stop.scheduledWindowStart;
  return pickup ? (
    <span className="text-muted-foreground shrink-0 font-mono text-xs tabular-nums">
      {t("Pickup {0}", formatMapTime(pickup))}
    </span>
  ) : null;
}

function AlertCategoryChips({
  allFeatures,
  hidden,
  onToggle,
}: {
  allFeatures: readonly WeatherAlertFeature[];
  hidden: readonly WeatherAlertCategory[];
  onToggle: (category: WeatherAlertCategory) => void;
}) {
  const t = useT();
  const counts = new Map<WeatherAlertCategory, number>();
  for (const feature of allFeatures) {
    const category = feature.properties.alertCategory;
    counts.set(category, (counts.get(category) ?? 0) + 1);
  }
  const categories = (Object.keys(ALERT_CATEGORY_CONFIG) as WeatherAlertCategory[]).filter(
    (category) => counts.has(category),
  );
  if (categories.length === 0) return null;

  return (
    <div role="group" aria-label={t("Alert categories")} className="flex flex-wrap gap-1">
      {categories.map((category) => {
        const on = !hidden.includes(category);
        return (
          <button
            key={category}
            type="button"
            aria-pressed={on}
            onClick={() => onToggle(category)}
            className={cn(
              "ui-focus-ring inline-flex h-6 items-center gap-1.5 rounded-md border px-1.5 text-xs",
              on
                ? "bg-card text-foreground hover:bg-surface-hover"
                : "text-muted-foreground border-dashed",
            )}
          >
            <AlertSwatch
              color={ALERT_CATEGORY_CONFIG[category].stroke}
              className={cn("size-2 shrink-0 rounded-sm", !on && "opacity-40")}
            />
            {t(ALERT_CATEGORY_CONFIG[category].label)}
            <span className="text-muted-foreground font-mono tabular-nums">
              {counts.get(category)}
            </span>
          </button>
        );
      })}
    </div>
  );
}

function AlertList({
  features,
  matches,
  query,
}: {
  features: readonly WeatherAlertFeature[];
  matches: AlertMatches;
  query: string;
}) {
  const t = useT();
  const needle = query.trim().toLowerCase();
  const listed = features
    .filter(
      (f) =>
        !needle ||
        [f.properties.event, f.properties.areaDesc, f.properties.headline]
          .filter(Boolean)
          .join(" ")
          .toLowerCase()
          .includes(needle),
    )
    .sort(
      (a, b) =>
        severityRank(a.properties.severity) - severityRank(b.properties.severity) ||
        (matches.byAlert.get(b.properties.id)?.length ?? 0) -
          (matches.byAlert.get(a.properties.id)?.length ?? 0),
    );
  const onRoutes = listed.filter((f) => matches.byAlert.has(f.properties.id));
  const elsewhere = listed.filter((f) => !matches.byAlert.has(f.properties.id));

  if (listed.length === 0) {
    return (
      <p className="text-muted-foreground px-4 py-8 text-center text-sm">
        {features.length === 0
          ? t("No public weather alerts in effect.")
          : t("No alerts match. Clear the search or turn a category back on.")}
      </p>
    );
  }

  return (
    <>
      {onRoutes.length > 0 ? (
        <section aria-label={t("On your routes")}>
          <GroupHeader label={t("On your routes")} count={onRoutes.length} />
          <ul>
            {onRoutes.map((feature) => (
              <li key={feature.properties.id}>
                <AlertRow
                  feature={feature}
                  affected={matches.byAlert.get(feature.properties.id)?.length ?? 0}
                />
              </li>
            ))}
          </ul>
        </section>
      ) : null}
      {elsewhere.length > 0 ? (
        <section aria-label={t("Elsewhere")}>
          <GroupHeader label={t("Elsewhere")} count={elsewhere.length} />
          <ul>
            {elsewhere.map((feature) => (
              <li key={feature.properties.id}>
                <AlertRow feature={feature} affected={0} />
              </li>
            ))}
          </ul>
        </section>
      ) : null}
    </>
  );
}

function AlertRow({ feature, affected }: { feature: WeatherAlertFeature; affected: number }) {
  const t = useT();
  const props = feature.properties;
  const config = ALERT_CATEGORY_CONFIG[props.alertCategory] ?? ALERT_CATEGORY_CONFIG.other;
  const selected = useMapFocusStore((s) => s.focus?.kind === "alert" && s.focus.id === props.id);
  const hovered = useMapFocusStore((s) => s.alertHoverId === props.id);
  const toggleAlert = useMapFocusStore((s) => s.toggleAlert);
  const setAlertHoverId = useMapFocusStore((s) => s.setAlertHoverId);
  const nowSeconds = Math.floor(Date.now() / 1000);
  const upcoming = !!props.effective && props.effective > nowSeconds;

  return (
    <button
      type="button"
      aria-pressed={selected}
      className={cn(
        "border-subtle ui-focus-ring flex w-full flex-col gap-0.5 border-b px-3 py-2 text-left",
        selected ? "bg-surface-selected" : hovered ? "bg-surface-hover" : "hover:bg-surface-hover",
      )}
      onMouseEnter={() => setAlertHoverId(props.id)}
      onMouseLeave={() => {
        if (useMapFocusStore.getState().alertHoverId === props.id) setAlertHoverId(null);
      }}
      onClick={() => toggleAlert(props.id)}
    >
      <span className="flex min-w-0 items-center gap-2">
        <AlertSwatch color={config.stroke} />
        <span className="min-w-0 flex-1 truncate text-sm font-medium">{props.event}</span>
        {props.severity ? (
          <Badge variant={SEVERITY_BADGE_MAP[props.severity] ?? "neutral"}>{props.severity}</Badge>
        ) : null}
      </span>
      {props.areaDesc ? (
        <span className="text-muted-foreground truncate text-xs">{props.areaDesc}</span>
      ) : null}
      <span className="text-muted-foreground flex flex-wrap gap-x-1.5 text-xs">
        <span className={cn(affected > 0 && "text-foreground font-medium")}>
          {affected > 0
            ? t("{0, plural, one {# shipment affected} other {# shipments affected}}", affected)
            : t("No shipments affected")}
        </span>
        {upcoming && props.effective ? (
          <span>{t("From {0}", formatMapTime(props.effective))}</span>
        ) : props.expires ? (
          <span>{t("Until {0}", formatMapTime(props.expires))}</span>
        ) : null}
      </span>
    </button>
  );
}
