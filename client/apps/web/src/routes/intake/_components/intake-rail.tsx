import { RailItem, RailSection } from "@/components/navigation/queue-rail";
import type { CaptureSource } from "@/lib/graphql/capture";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Switch } from "@trenova/shared/components/ui/switch";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  ArchiveXIcon,
  CheckCheckIcon,
  InboxIcon,
  LayersIcon,
  LoaderIcon,
  PrinterIcon,
  ScanLineIcon,
  type LucideIcon,
} from "lucide-react";
import { useId } from "react";
import { INTAKE_VIEWS, viewLabel, type IntakeFilter, type IntakeView } from "./queue-filter";

const VIEW_ICON: Record<IntakeView, LucideIcon> = {
  waiting: InboxIcon,
  working: LoaderIcon,
  filed: CheckCheckIcon,
  closed: ArchiveXIcon,
  all: LayersIcon,
};

const SOURCES: { value: CaptureSource | null; icon: LucideIcon }[] = [
  { value: null, icon: LayersIcon },
  { value: "Scan", icon: ScanLineIcon },
  { value: "Print", icon: PrinterIcon },
];

/**
 * The queue's views and filters. What is waiting to be filed is the loud
 * count; the others say only what they hold when opened.
 */
export function IntakeRail({
  filter,
  waiting,
  onChange,
}: {
  filter: IntakeFilter;
  waiting: number | undefined;
  onChange: (filter: IntakeFilter) => void;
}) {
  const t = useT();
  const mineId = useId();

  const sourceLabel = (source: CaptureSource | null) =>
    source === null ? t("Scans and prints") : source === "Scan" ? t("Scanned") : t("Printed");

  return (
    <nav aria-label={t("Intake views")} className="flex min-h-0 flex-col">
      <ScrollArea className="min-h-0 flex-1">
        <div className="flex flex-col gap-5 p-3">
          <RailSection>
            {INTAKE_VIEWS.map((view) => (
              <RailItem
                key={view}
                icon={VIEW_ICON[view]}
                label={viewLabel(t, view)}
                active={filter.view === view}
                count={view === "waiting" ? waiting : undefined}
                loud={view === "waiting" && (waiting ?? 0) > 0}
                onClick={() => onChange({ ...filter, view })}
              />
            ))}
          </RailSection>

          <RailSection title={t("Came from")}>
            {SOURCES.map(({ value, icon }) => (
              <RailItem
                key={value ?? "any"}
                icon={icon}
                label={sourceLabel(value)}
                active={filter.source === value}
                onClick={() => onChange({ ...filter, source: value })}
              />
            ))}
          </RailSection>

          <div className="flex items-center justify-between gap-2 px-2">
            <label htmlFor={mineId} className="text-foreground-muted text-sm">
              {t("Only mine")}
            </label>
            <Switch
              id={mineId}
              checked={filter.mine}
              onCheckedChange={(mine) => onChange({ ...filter, mine })}
            />
          </div>
        </div>
      </ScrollArea>
    </nav>
  );
}
