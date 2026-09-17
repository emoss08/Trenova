import { useT } from "@trenova/shared/i18n/use-t";
import {
  AGENT_ACCENT_ORDER,
  AGENT_ACCENTS,
  AGENT_ICON_ORDER,
  AGENT_ICONS,
  type AgentAccentName,
  type AgentIconName,
  isAgentAccentName,
  isAgentIconName,
  resolveAgentIdentity,
} from "@/components/agent-identity/agent-identity";
import { AgentTile } from "@/components/agent-identity/agent-tile";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { cn } from "@trenova/shared/lib/utils";
import { CheckIcon, ChevronDownIcon } from "lucide-react";
import { useState } from "react";

/**
 * The names a screen reader reads out. They are literal t() calls rather than a
 * lookup table because the extractor only sees literals, and a face nobody can
 * name is not a choice a keyboard user can make.
 */
function iconLabel(t: Translate, icon: AgentIconName): string {
  switch (icon) {
    case "bot":
      return t("Robot");
    case "truck":
      return t("Truck");
    case "route":
      return t("Route");
    case "receipt":
      return t("Receipt");
    case "wallet":
      return t("Wallet");
    case "shield":
      return t("Shield");
    case "headset":
      return t("Headset");
    case "clipboard":
      return t("Clipboard");
    case "compass":
      return t("Compass");
    case "radar":
      return t("Radar");
    case "gauge":
      return t("Gauge");
    case "package":
      return t("Package");
    case "file":
      return t("Document");
    case "search":
      return t("Search");
    case "bell":
      return t("Bell");
    default:
      return t("Spark");
  }
}

function accentLabel(t: Translate, accent: AgentAccentName): string {
  switch (accent) {
    case "indigo":
      return t("Indigo");
    case "teal":
      return t("Teal");
    case "amber":
      return t("Amber");
    case "rose":
      return t("Rose");
    case "emerald":
      return t("Emerald");
    case "sky":
      return t("Sky");
    case "violet":
      return t("Violet");
    default:
      return t("Slate");
  }
}

type Translate = ReturnType<typeof useT>;

type IdentityPickerProps = {
  name: string;
  template: string | null;
  icon: string;
  accent: string;
  onIconChange: (icon: string) => void;
  onAccentChange: (accent: string) => void;
};

/**
 * What the agent looks like everywhere it appears.
 *
 * The face is one field, not two grids laid out in the form. Twenty-four
 * swatches sitting open between Name and Description is the single loudest
 * thing on a page whose real subject is what the agent may do, so the choice
 * lives behind the tile it produces and the form shows only the result.
 */
export function IdentityPicker({
  name,
  template,
  icon,
  accent,
  onIconChange,
  onAccentChange,
}: IdentityPickerProps) {
  const t = useT();
  const [open, setOpen] = useState(false);

  const agent = { name, template, icon, accent };
  const resolved = resolveAgentIdentity(agent);
  const automatic = !isAgentIconName(icon) && !isAgentAccentName(accent);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <button
            type="button"
            aria-label={t("Choose a face")}
            className={cn(
              "border-input bg-muted hover:bg-muted/80 flex h-7 w-full items-center gap-1.5 rounded-md border px-1.5 text-xs transition-[border-color,box-shadow] duration-200 ease-in-out outline-hidden",
              "data-pressed:border-brand data-pressed:ring-brand/30 data-pressed:ring-4",
            )}
          />
        }
      >
        <AgentTile agent={agent} size="xs" />
        <span className="truncate">
          {iconLabel(t, resolved.icon)} · {accentLabel(t, resolved.accent)}
        </span>
        {automatic && <span className="text-muted-foreground shrink-0">{t("automatic")}</span>}
        <ChevronDownIcon className="text-muted-foreground ml-auto size-3.5 shrink-0" />
      </PopoverTrigger>

      <PopoverContent align="start" className="w-72 p-3" positionerClassName="rounded-lg">
        <div className="flex flex-col gap-3">
          <div className="flex items-center gap-2.5">
            <AgentTile agent={agent} size="xl" />
            <div className="min-w-0 flex-1">
              <p className="truncate text-sm font-medium">{name || t("This agent")}</p>
              <p className="text-muted-foreground text-xs">
                {automatic ? t("Trenova picked this and will keep it") : t("Chosen for this agent")}
              </p>
            </div>
            {!automatic && (
              <Button
                variant="ghost"
                size="xs"
                onClick={() => {
                  onIconChange("");
                  onAccentChange("");
                }}
              >
                {t("Reset")}
              </Button>
            )}
          </div>

          <div>
            <p className="text-muted-foreground mb-1.5 text-xs font-medium">{t("Symbol")}</p>
            <div role="radiogroup" aria-label={t("Symbol")} className="grid grid-cols-8 gap-1">
              {AGENT_ICON_ORDER.map((option) => {
                const Icon = AGENT_ICONS[option];
                const selected = icon === option;

                return (
                  <button
                    key={option}
                    type="button"
                    role="radio"
                    aria-checked={selected}
                    aria-label={iconLabel(t, option)}
                    onClick={() => onIconChange(selected ? "" : option)}
                    className={cn(
                      "flex aspect-square items-center justify-center rounded-md transition-colors",
                      selected
                        ? "ring-foreground/25 bg-muted text-foreground ring-1"
                        : "text-muted-foreground hover:bg-muted hover:text-foreground",
                    )}
                    style={
                      selected || resolved.icon === option
                        ? { color: AGENT_ACCENTS[resolved.accent] }
                        : undefined
                    }
                  >
                    <Icon className="size-3.5" />
                  </button>
                );
              })}
            </div>
          </div>

          <div>
            <p className="text-muted-foreground mb-1.5 text-xs font-medium">{t("Colour")}</p>
            <div role="radiogroup" aria-label={t("Colour")} className="flex gap-1.5">
              {AGENT_ACCENT_ORDER.map((option) => {
                const selected = accent === option;

                return (
                  <button
                    key={option}
                    type="button"
                    role="radio"
                    aria-checked={selected}
                    aria-label={accentLabel(t, option)}
                    onClick={() => onAccentChange(selected ? "" : option)}
                    // The tick rides on a ring of the page's own background
                    // rather than on white, which disappears on the light
                    // accents and glares on the dark ones.
                    className={cn(
                      "ring-offset-background flex size-5 flex-1 items-center justify-center rounded-full transition-transform hover:scale-110",
                      selected && "ring-foreground/40 ring-2 ring-offset-2",
                    )}
                    style={{ backgroundColor: AGENT_ACCENTS[option] }}
                  >
                    {selected && <CheckIcon className="text-background size-3" />}
                  </button>
                );
              })}
            </div>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
