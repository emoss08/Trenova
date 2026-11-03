import type { TimelineLoad } from "@/lib/shipment-board/timeline";
import { groupLoadsByStage } from "@/lib/shipment-board/timeline";
import { STAGE_META } from "@/lib/shipment-board/stage";
import { ClockIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { IdentityAvatar } from "../../identity-avatar";
import { useShipmentBoardStore } from "../../store";
import { TimelineBar } from "./timeline-bar";
import { TimelineGroupRow, TimelineRow } from "./timeline-canvas";
import type { TimelineScale } from "./timeline-scale";

const ROW_HEIGHT = 40;
const BAR_HEIGHT = 20;

/** One row per load, grouped by stage, most urgent first. */
export function TimelineShipmentRows({
  loads,
  scale,
  selectedId,
  onSelect,
}: {
  loads: readonly TimelineLoad[];
  scale: TimelineScale;
  selectedId: string | null;
  onSelect: (id: string) => void;
}) {
  const t = useT();

  return (
    <>
      {groupLoadsByStage(loads).map((group) => (
        <section key={group.stage} aria-label={t(STAGE_META[group.stage].label)}>
          <TimelineGroupRow
            scale={scale}
            label={t(STAGE_META[group.stage].label)}
            count={group.loads.length}
            swatchClassName={STAGE_META[group.stage].swatchClassName}
          />
          {group.loads.map((load) => (
            <ShipmentRow
              key={load.id}
              load={load}
              scale={scale}
              selected={load.id === selectedId}
              onSelect={onSelect}
            />
          ))}
        </section>
      ))}
    </>
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
