import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Separator } from "@/components/ui/separator";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { InfoIcon } from "lucide-react";

function LegendItem({ swatch, label }: { swatch: React.ReactNode; label: string }) {
  return (
    <div className="flex items-center gap-2 py-0.5">
      {swatch}
      <span className="text-xs text-foreground">{label}</span>
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
      style={{ borderColor: color, backgroundColor: `${color}20` }}
    />
  );
}

function DashedLine() {
  return (
    <svg width="20" height="8" className="shrink-0">
      <line x1="0" y1="4" x2="20" y2="4" stroke="#3b82f6" strokeWidth="2.5" strokeDasharray="4 3" />
    </svg>
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
  return (
    <Popover>
      <Tooltip>
        <TooltipTrigger
          render={
            <PopoverTrigger
              render={<Button variant="outline" size="icon" className="bg-background shadow-sm" />}
            />
          }
        >
          <InfoIcon className="size-4" />
        </TooltipTrigger>
        <TooltipContent side="left">Map legend</TooltipContent>
      </Tooltip>
      <PopoverContent side="left" sideOffset={8} className="w-48 p-3 gap-0.5">
        <span className="text-xs font-semibold tracking-wider text-muted-foreground uppercase">
          Legend
        </span>
        <div className="mt-2 flex flex-col gap-0.5">
          <LegendItem swatch={<Dot color="#000" />} label="Vehicle" />
          <LegendItem swatch={<Dot color="#3b82f6" />} label="Pickup Stop" />
          <LegendItem swatch={<Dot color="#16a34a" />} label="Delivery Stop" />
          <LegendItem swatch={<DashedLine />} label="Route" />
          <LegendItem swatch={<Ring color="#3b82f6" />} label="Geofence" />
        </div>

        <Separator className="my-2.5" />

        <span className="text-xs font-semibold tracking-wider text-muted-foreground uppercase">
          Overlays
        </span>
        <div className="mt-2 flex flex-col gap-0.5">
          <LegendItem swatch={<GradientBar from="#22c55e" to="#ef4444" />} label="Traffic" />
          <LegendItem swatch={<GradientBar from="#a3d9f5" to="#1e3a8a" />} label="Precipitation" />
          <LegendItem swatch={<GradientBar from="#dbeafe" to="#6366f1" />} label="Wind Speed" />
          <LegendItem swatch={<GradientBar from="#3b82f6" to="#ef4444" />} label="Temperature" />
        </div>
      </PopoverContent>
    </Popover>
  );
}
