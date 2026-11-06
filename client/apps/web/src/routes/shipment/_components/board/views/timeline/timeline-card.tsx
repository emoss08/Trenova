import { STAGE_META } from "@/lib/shipment-board/stage";
import { STOP_TYPE_LABELS } from "@/lib/shipment-board/stop-type-labels";
import type { TimelineLoad } from "@/lib/shipment-board/timeline";
import { XCloseIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatMinutesSpan } from "@trenova/shared/lib/date";
import { useMapFocusStore } from "../../../map/map-focus-store";
import { NextStep } from "../../expanded-row/next-step";
import { useShipmentBoardUrl } from "../../url-state";
import { formatTimelineTime } from "@/components/timeline/timeline-format";
import type { TimelineScale } from "@/components/timeline/timeline-scale";

/**
 * The selected load beside the timeline: its stops with appointments and
 * ETAs, then the board's own next step for it (coverage to choose, a delay to
 * tell the customer about, or who has it), then the ways out.
 */
export function TimelineCard({
  load,
  scale,
  onClose,
}: {
  load: TimelineLoad;
  scale: TimelineScale;
  onClose: () => void;
}) {
  const t = useT();
  const [, setUrl] = useShipmentBoardUrl();
  const meta = STAGE_META[load.stage];
  const tone = `var(${load.coverage.kind === "uncovered" ? "--warning" : meta.colorVar})`;

  const showOnMap = () => {
    useMapFocusStore.setState({ focus: { kind: "shipment", id: load.id }, railTab: "shipments" });
    void setUrl({ view: "map" });
  };

  return (
    <aside
      aria-label={t("Shipment details")}
      className="ui-lift bg-card absolute top-2.5 right-2.5 bottom-2.5 z-20 flex w-85 flex-col overflow-hidden rounded-lg @max-3xl/timeline:top-auto @max-3xl/timeline:left-2.5 @max-3xl/timeline:h-3/5 @max-3xl/timeline:w-auto"
    >
      <header className="border-subtle flex flex-col gap-0.5 border-b px-3 pt-2.5 pb-2">
        <div className="flex min-w-0 items-center gap-2">
          <span className="truncate font-mono text-sm font-semibold tabular-nums">
            {load.shipment.proNumber ?? "—"}
          </span>
          <Badge variant={meta.tone}>{t(meta.label)}</Badge>
          <Button
            variant="ghost"
            size="icon-sm"
            className="ml-auto shrink-0"
            aria-label={t("Close")}
            onClick={onClose}
          >
            <XCloseIcon className="size-3.5" />
          </Button>
        </div>
        {load.shipment.customer?.name ? (
          <p className="text-muted-foreground truncate text-xs">{load.shipment.customer.name}</p>
        ) : null}
      </header>

      <div className="flex min-h-0 flex-1 flex-col overflow-y-auto">
        <ol className="flex flex-col px-3 pt-3">
          {load.stops.map((stop, index) => {
            const projected = load.eta && load.eta.stopIndex === index ? load.eta : null;
            return (
              <li key={stop.id} className="relative flex gap-2.5 pb-3">
                {index < load.stops.length - 1 ? (
                  <span aria-hidden className="bg-border absolute top-5 bottom-0 left-2 w-px" />
                ) : null}
                <span
                  aria-hidden
                  className="relative grid size-4 shrink-0 place-items-center rounded-full font-mono text-2xs font-semibold tabular-nums"
                  style={{
                    background: stop.done ? tone : "var(--card)",
                    color: stop.done ? "var(--foreground-on-solid)" : tone,
                    border: `2px solid ${tone}`,
                  }}
                >
                  {index + 1}
                </span>
                <div className="flex min-w-0 flex-col">
                  <span className="truncate text-sm font-medium">
                    {stop.place || "—"}{" "}
                    <span className="text-muted-foreground text-xs font-normal">
                      {STOP_TYPE_LABELS[stop.type]}
                    </span>
                  </span>
                  <span className="text-muted-foreground flex flex-wrap gap-x-1.5 font-mono text-xs tabular-nums">
                    {stop.done ? (
                      <span>
                        {stop.departedAt
                          ? t("Departed {0}", formatTimelineTime(stop.departedAt, scale))
                          : t(
                              "Arrived {0}",
                              formatTimelineTime(stop.arrivedAt ?? stop.windowStart, scale),
                            )}
                      </span>
                    ) : (
                      <>
                        <span>
                          {t(
                            "Appt {0} to {1}",
                            formatTimelineTime(stop.windowStart, scale),
                            formatTimelineTime(stop.windowEnd, scale),
                          )}
                        </span>
                        {projected ? (
                          <span>{t("ETA {0}", formatTimelineTime(projected.at, scale))}</span>
                        ) : null}
                        {projected && projected.lateMinutes > 0 ? (
                          <span className="text-danger font-medium">
                            {t("{0} late", formatMinutesSpan(projected.lateMinutes))}
                          </span>
                        ) : null}
                      </>
                    )}
                  </span>
                </div>
              </li>
            );
          })}
        </ol>
        <div className="border-subtle border-t px-3 py-3">
          <NextStep shipment={load.shipment} />
        </div>
      </div>

      <footer className="border-subtle flex items-center gap-2 border-t px-3 py-2.5">
        <Button variant="outline" size="sm" onClick={showOnMap}>
          {t("Show on map")}
        </Button>
        <Button
          size="sm"
          className="ml-auto"
          onClick={() => void setUrl({ view: "table", expanded: load.id })}
        >
          {t("Open shipment")}
        </Button>
      </footer>
    </aside>
  );
}
