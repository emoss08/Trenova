import type { TimelineLoad } from "@/lib/shipment-board/timeline";
import { groupLoadsByStage } from "@/lib/shipment-board/timeline";
import { STAGE_META, type ActiveStage } from "@/lib/shipment-board/stage";
import { ClockIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { IdentityAvatar } from "../../identity-avatar";
import { useShipmentBoardStore } from "../../store";
import { TimelineBar } from "./timeline-bar";
import {
  TimelineCanvas,
  TimelineGroupRow,
  TimelineRow,
  type TimelineScrollMemory,
  type TimelineSheetItem,
} from "./timeline-canvas";
import { BAR_HEIGHT, GROUP_HEIGHT, ROW_HEIGHT, type TimelineScale } from "./timeline-scale";

/** The sheet's lines for the shipments view: a header per stage, then a row per load, most urgent first. */
function shipmentSheetItems({
  loads,
  scale,
  selectedId,
  onSelect,
  stageLabel,
}: {
  loads: readonly TimelineLoad[];
  scale: TimelineScale;
  selectedId: string | null;
  onSelect: (id: string) => void;
  stageLabel: (stage: ActiveStage) => string;
}): TimelineSheetItem[] {
  const items: TimelineSheetItem[] = [];
  for (const group of groupLoadsByStage(loads)) {
    items.push({
      key: `stage:${group.stage}`,
      height: GROUP_HEIGHT,
      header: true,
      node: (
        <TimelineGroupRow
          scale={scale}
          label={stageLabel(group.stage)}
          count={group.loads.length}
          swatchClassName={STAGE_META[group.stage].swatchClassName}
        />
      ),
    });
    for (const load of group.loads) {
      items.push({
        key: load.id,
        height: ROW_HEIGHT,
        node: (
          <ShipmentRow
            load={load}
            scale={scale}
            selected={load.id === selectedId}
            onSelect={onSelect}
          />
        ),
      });
    }
  }
  return items;
}

/** The shipments view: one row per load on the virtualized sheet. */
export function TimelineShipmentSheet({
  loads,
  scale,
  today,
  selectedId,
  onSelect,
  scrollMemory,
}: {
  loads: readonly TimelineLoad[];
  scale: TimelineScale;
  today: number;
  selectedId: string | null;
  onSelect: (id: string) => void;
  scrollMemory: TimelineScrollMemory;
}) {
  const t = useT();
  const items = shipmentSheetItems({
    loads,
    scale,
    selectedId,
    onSelect,
    stageLabel: (stage) => t(STAGE_META[stage].label),
  });

  return (
    <TimelineCanvas
      scale={scale}
      today={today}
      labelHeading={t("Shipment")}
      items={items}
      revealKey={selectedId}
      scrollMemory={scrollMemory}
      empty={
        <p className="text-muted-foreground sticky left-0 w-fit px-4 py-10 text-sm">
          {t("No loads are active in this range.")}
        </p>
      }
    />
  );
}

function ShipmentRow({
  load,
  scale,
  selected,
  onSelect,
}: {
  load: TimelineLoad;
  scale: TimelineScale;
  selected: boolean;
  onSelect: (id: string) => void;
}) {
  const t = useT();
  const highlighted = useShipmentBoardStore((s) => s.highlightId === load.id);
  const first = load.stops[0];
  const last = load.stops[load.stops.length - 1];
  const extraStops = load.stops.length - 2;

  return (
    <TimelineRow
      height={ROW_HEIGHT}
      scale={scale}
      selected={selected}
      highlighted={highlighted}
      onLabelClick={() => onSelect(load.id)}
      labelTitle={load.stops.map((stop) => stop.place).join(" → ")}
      label={
        <>
          <span className="flex min-w-0 flex-1 flex-col leading-tight">
            <span className="flex min-w-0 items-center gap-1.5 text-sm font-medium">
              <span className="truncate">{[first.city, last.city].join(" → ")}</span>
              {extraStops > 0 ? (
                <span className="text-muted-foreground shrink-0 font-mono text-xs font-normal">
                  {t("+{0}", extraStops)}
                </span>
              ) : null}
            </span>
            <span className="text-muted-foreground flex min-w-0 gap-1.5 text-xs">
              <span className="shrink-0 font-mono tabular-nums">
                {load.shipment.proNumber ?? "—"}
              </span>
              <span className="truncate">{load.shipment.customer?.name}</span>
            </span>
          </span>
          <CoverageMark load={load} />
        </>
      }
      lane={
        <TimelineBar
          load={load}
          scale={scale}
          top={(ROW_HEIGHT - BAR_HEIGHT) / 2}
          height={BAR_HEIGHT}
          caption="timing"
          selected={selected}
          onSelect={onSelect}
        />
      }
    />
  );
}

/** Who has the load, as a small avatar: a person round, a company square, nobody dashed. */
export function CoverageMark({ load }: { load: TimelineLoad }) {
  const t = useT();
  const coverage = load.coverage;
  if (coverage.kind === "driver" || coverage.kind === "carrier") {
    return (
      <span title={[coverage.name, coverage.detail].filter(Boolean).join(" · ")}>
        <IdentityAvatar
          id={coverage.id}
          initials={coverage.initials}
          shape={coverage.kind === "driver" ? "circle" : "square"}
          className="size-5.5 text-2xs"
        />
      </span>
    );
  }
  const tendered = coverage.kind === "tendered";
  return (
    <span
      title={tendered ? t("Tendered, waiting on the carrier") : t("Not covered")}
      className={
        tendered
          ? "bg-brand-subtle text-brand-subtle-foreground inline-grid size-5.5 shrink-0 place-items-center rounded-full text-2xs font-medium"
          : "border-warning text-warning inline-grid size-5.5 shrink-0 place-items-center rounded-full border-[1.5px] border-dashed text-2xs font-medium"
      }
    >
      <span className="sr-only">
        {tendered ? t("Tendered, waiting on the carrier") : t("Not covered")}
      </span>
      {tendered ? <ClockIcon aria-hidden className="size-3" /> : null}
    </span>
  );
}
