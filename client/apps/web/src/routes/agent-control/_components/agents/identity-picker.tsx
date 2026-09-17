import { useT } from "@trenova/shared/i18n/use-t";
import {
  AGENT_ACCENT_ORDER,
  AGENT_ACCENTS,
  AGENT_ICON_ORDER,
  AGENT_ICONS,
  type AgentAccentName,
  type AgentIconName,
  resolveAgentIdentity,
} from "@/components/agent-identity/agent-identity";
import { AgentTile } from "@/components/agent-identity/agent-tile";
import { cn } from "@trenova/shared/lib/utils";
import { CheckIcon } from "lucide-react";

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
 * What the agent looks like everywhere it appears. Leaving both alone is a real
 * choice: the preview shows the face Trenova would pick, so a person can see
 * what they are overriding before they override it.
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
  const resolved = resolveAgentIdentity({ name, template, icon, accent });

  return (
    <div className="flex items-start gap-3">
      <AgentTile agent={{ name, template, icon, accent }} size="xl" />

      <div className="flex min-w-0 flex-1 flex-col gap-2">
        <div role="radiogroup" aria-label={t("Icon")} className="flex flex-wrap gap-1">
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
                  "flex size-7 items-center justify-center rounded-md border transition-colors",
                  selected
                    ? "border-foreground/30 bg-muted text-foreground"
                    : "border-transparent text-muted-foreground hover:bg-muted hover:text-foreground",
                  !selected && resolved.icon === option && "text-foreground",
                )}
              >
                <Icon className="size-3.5" />
              </button>
            );
          })}
        </div>

        <div role="radiogroup" aria-label={t("Colour")} className="flex flex-wrap gap-1.5">
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
                className="ring-offset-background flex size-5 items-center justify-center rounded-full transition-transform hover:scale-110"
                style={{ backgroundColor: AGENT_ACCENTS[option] }}
              >
                {selected && <CheckIcon className="size-3 text-white" />}
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
}
