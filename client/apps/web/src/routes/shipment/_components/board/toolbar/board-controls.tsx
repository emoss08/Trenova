import { useT } from "@trenova/shared/i18n/use-t";
import {
  Building07Icon,
  CalendarCheck01Icon,
  CalendarIcon,
  CheckIcon,
  LayersThree01Icon,
  LayoutRightIcon,
  Map01Icon,
  Rows03Icon,
  SlashCircle01Icon,
  TableIcon,
  User01Icon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { cn } from "@trenova/shared/lib/utils";
import { type BoardGrouping, GROUPING_LABELS } from "@/lib/shipment-board/grouping";
import { type ComponentType, useState } from "react";
import { type BoardView, useShipmentBoardUrl } from "../url-state";

/* Labels give way to icons when the board is narrow; the buttons never shrink. */
const TOOLBAR_LABEL_CLASS = "hidden @[1180px]/board:inline";

/* A view's icon alone still has to name it, so its label stays for screen readers. */
const VIEW_LABEL_CLASS = "sr-only @[1180px]/board:not-sr-only";

/* The display menu and saved views are the first to go on a narrow board. */
export const TOOLBAR_RESPONSIVE = {
  label: TOOLBAR_LABEL_CLASS,
  secondary: "hidden @[1000px]/board:contents",
};

const VIEWS: Array<{
  value: BoardView;
  label: string;
  icon: ComponentType<{ className?: string }>;
}> = [
  { value: "table", label: "Table", icon: TableIcon },
  { value: "timeline", label: "Timeline", icon: Rows03Icon },
  { value: "map", label: "Map", icon: Map01Icon },
];

export function ViewSwitch() {
  const t = useT();
  const [{ view }, setUrl] = useShipmentBoardUrl();
  return (
    <SegmentedControl<BoardView>
      aria-label={t("View")}
      className="h-7 shrink-0"
      labelClassName={VIEW_LABEL_CLASS}
      value={view}
      onValueChange={(next) => void setUrl({ view: next })}
      items={VIEWS.map((entry) => ({
        value: entry.value,
        label: t(entry.label),
        icon: entry.icon,
      }))}
    />
  );
}

function ToggleButton({
  pressed,
  onPressedChange,
  label,
  icon: Icon,
  showLabel,
}: {
  pressed: boolean;
  onPressedChange: (pressed: boolean) => void;
  label: string;
  icon: ComponentType<{ className?: string }>;
  showLabel: boolean;
}) {
  const button = (
    <Button
      variant="outline"
      size="sm"
      aria-pressed={pressed}
      aria-label={label}
      onClick={() => onPressedChange(!pressed)}
      className="aria-pressed:bg-brand-subtle aria-pressed:text-brand-subtle-foreground aria-pressed:border-brand/30 shrink-0"
    >
      <Icon className="size-3.5" />
      {showLabel ? <span className={TOOLBAR_LABEL_CLASS}>{label}</span> : null}
    </Button>
  );
  if (showLabel) return button;
  return (
    <Tooltip>
      <TooltipTrigger render={button} />
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}

const GROUPING_OPTIONS: Array<{
  value: BoardGrouping;
  icon: ComponentType<{ className?: string }>;
}> = [
  { value: "stage", icon: LayersThree01Icon },
  { value: "shipDate", icon: CalendarIcon },
  { value: "deliveryDate", icon: CalendarCheck01Icon },
  { value: "customer", icon: Building07Icon },
  { value: "owner", icon: User01Icon },
  { value: "none", icon: SlashCircle01Icon },
];

/** Picks what the table groups by. Changing it opens every group again. */
export function GroupMenu() {
  const t = useT();
  const [open, setOpen] = useState(false);
  const [{ group, view }, setUrl] = useShipmentBoardUrl();
  if (view !== "table") return null;

  const grouped = group !== "none";
  const label = grouped ? t(GROUPING_LABELS[group]) : t("Group");

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button
            variant="outline"
            size="sm"
            aria-label={t("Group by")}
            data-grouped={grouped}
            className="data-[grouped=true]:bg-brand-subtle data-[grouped=true]:text-brand-subtle-foreground data-[grouped=true]:border-brand/30 shrink-0"
          >
            <LayersThree01Icon className="size-3.5" />
            <span className={TOOLBAR_LABEL_CLASS}>{label}</span>
          </Button>
        }
      />
      <PopoverContent align="start" className="w-56 gap-1 p-1">
        <span className="text-muted-foreground px-2 pt-1 pb-0.5 text-xs font-medium">
          {t("Group by")}
        </span>
        <div role="radiogroup" aria-label={t("Group by")} className="flex flex-col">
          {GROUPING_OPTIONS.map((option) => {
            const selected = group === option.value;
            const Icon = option.icon;
            return (
              <button
                key={option.value}
                type="button"
                role="radio"
                aria-checked={selected}
                onClick={() => {
                  setOpen(false);
                  if (!selected) void setUrl({ group: option.value, collapsed: [] });
                }}
                className={cn(
                  "ui-focus-ring hover:bg-surface-active flex h-7 cursor-pointer items-center gap-2 rounded-md px-2 text-left text-sm",
                  selected ? "text-foreground" : "text-muted-foreground hover:text-foreground",
                )}
              >
                <Icon className="size-3.5 shrink-0" />
                <span className="flex-1 truncate">{t(GROUPING_LABELS[option.value])}</span>
                {selected ? <CheckIcon className="size-3.5 shrink-0" /> : null}
              </button>
            );
          })}
        </div>
      </PopoverContent>
    </Popover>
  );
}

export function PanelToggle({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  return (
    <ToggleButton
      pressed={open}
      onPressedChange={onOpenChange}
      label={t("Toggle side panel")}
      icon={LayoutRightIcon}
      showLabel={false}
    />
  );
}
