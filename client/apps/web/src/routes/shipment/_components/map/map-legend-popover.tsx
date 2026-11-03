import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Separator } from "@trenova/shared/components/ui/separator";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { InfoCircleIcon, Truck01Icon } from "@trenova/shared/components/icons";

function LegendItem({ swatch, label }: { swatch: React.ReactNode; label: string }) {
  return (
    <div className="flex items-center gap-2 py-0.5">
      {swatch}
      <span className="text-foreground text-xs">{label}</span>
    </div>
  );
}

function Dot({ color }: { color: string }) {
  return (
    <span
      className="inline-block size-3 shrink-0 rounded-full"
      style={{ backgroundColor: color }}
    />
  );
}

function Ring({ color }: { color: string }) {
  return (
    <span
      className="inline-block size-3 shrink-0 rounded-full border-2"
      style={{
        borderColor: color,
        backgroundColor: `color-mix(in oklab, ${color} 12%, transparent)`,
      }}
    />
  );
}

function RouteLine({ dotted }: { dotted?: boolean }) {
  return (
    <svg width="20" height="8" className="text-info shrink-0" aria-hidden>
      <line
        x1="1"
        y1="4"
        x2="19"
        y2="4"
        stroke="currentColor"
        strokeWidth="2.5"
        strokeLinecap="round"
        strokeDasharray={dotted ? "1 4" : undefined}
      />
    </svg>
  );
}

function ClusterChip() {
  return (
    <span
      className="bg-card text-foreground inline-flex size-3.5 shrink-0 items-center justify-center rounded-full font-mono text-2xs"
      style={{ boxShadow: "0 0 0 2px var(--info)" }}
    >
      3
    </span>
  );
}

function TruckChip() {
  return (
    <span
      className="inline-flex size-3.5 shrink-0 items-center justify-center rounded-full"
      style={{
        background: "var(--brand)",
        border: "1px solid var(--card)",
        boxShadow: "0 0 0 1px rgba(255,255,255,0.85)",
      }}
    >
      <Truck01Icon className="size-2" style={{ color: "var(--brand-foreground)" }} />
    </span>
  );
}

function GradientBar({ from, to }: { from: string; to: string }) {
  return (
    <span
      className="inline-block h-3 w-5 shrink-0 rounded-sm"
      style={{ background: `linear-gradient(to right, ${from}, ${to})` }}
    />
  );
}

export function MapLegendPopover() {
  const t = useT();

  return (
    <Popover>
      <Tooltip>
        <TooltipTrigger
          render={
            <PopoverTrigger
              render={<Button variant="outline" size="icon" className="bg-background size-7" />}
            />
          }
        >
          <InfoCircleIcon className="size-4" />
        </TooltipTrigger>
        <TooltipContent side="bottom">{t("Map legend")}</TooltipContent>
      </Tooltip>
      <PopoverContent side="bottom" sideOffset={8} className="w-48 gap-0.5 p-3">
        <span className="text-muted-foreground text-xs font-semibold">{t("Legend")}</span>
        <div className="mt-2 flex flex-col gap-0.5">
          <LegendItem swatch={<Dot color="var(--info)" />} label={t("Shipment at its next stop")} />
          <LegendItem swatch={<ClusterChip />} label={t("Shipments close together")} />
          <LegendItem swatch={<RouteLine />} label={t("Driven")} />
          <LegendItem swatch={<RouteLine dotted />} label={t("Still to drive")} />
          <LegendItem swatch={<TruckChip />} label={t("Live vehicle")} />
          <LegendItem swatch={<Ring color="var(--info)" />} label={t("Geofence")} />
        </div>

        <Separator className="my-2.5" />

        <span className="text-muted-foreground text-xs font-semibold">{t("Overlays")}</span>
        <div className="mt-2 flex flex-col gap-0.5">
          <LegendItem swatch={<GradientBar from="#22c55e" to="#ef4444" />} label={t("Traffic")} />
          <LegendItem
            swatch={<GradientBar from="#a3d9f5" to="#1e3a8a" />}
            label={t("Precipitation")}
          />
          <LegendItem
            swatch={<GradientBar from="#dbeafe" to="#6366f1" />}
            label={t("Wind speed")}
          />
          <LegendItem
            swatch={<GradientBar from="#3b82f6" to="#ef4444" />}
            label={t("Temperature")}
          />
        </div>
      </PopoverContent>
    </Popover>
  );
}
