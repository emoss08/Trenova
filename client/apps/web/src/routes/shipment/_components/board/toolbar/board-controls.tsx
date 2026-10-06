import { useT } from "@trenova/shared/i18n/use-t";
import {
  LayersThree01Icon,
  LayoutRightIcon,
  Map01Icon,
  Rows03Icon,
  TableIcon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { cn } from "@trenova/shared/lib/utils";
import type { ComponentType } from "react";
import { type BoardView, useShipmentBoardUrl } from "../url-state";

/* Labels give way to icons when the board is narrow; the buttons never shrink. */
export const TOOLBAR_LABEL_CLASS = "hidden @[1180px]/board:inline";

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
    <div
      role="radiogroup"
      aria-label={t("View")}
      className="border-input bg-card flex h-7 shrink-0 items-center rounded-md border p-0.5"
    >
      {VIEWS.map((entry) => {
        const Icon = entry.icon;
        const active = view === entry.value;
        return (
          <button
            key={entry.value}
            type="button"
            role="radio"
            aria-checked={active}
            aria-label={t(entry.label)}
            onClick={() => void setUrl({ view: entry.value })}
            className={cn(
              "ui-focus-ring text-muted-foreground hover:text-foreground inline-flex h-full items-center gap-1.5 rounded-sm px-2 text-sm font-medium",
              active && "bg-surface-active text-foreground",
            )}
          >
            <Icon className="size-3.5" />
            <span className={TOOLBAR_LABEL_CLASS}>{t(entry.label)}</span>
          </button>
        );
      })}
    </div>
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

export function GroupToggle() {
  const t = useT();
  const [{ group, view }, setUrl] = useShipmentBoardUrl();
  if (view !== "table") return null;
  return (
    <ToggleButton
      pressed={group}
      onPressedChange={(next) => void setUrl({ group: next, collapsed: [] })}
      label={t("Group")}
      icon={LayersThree01Icon}
      showLabel
    />
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
