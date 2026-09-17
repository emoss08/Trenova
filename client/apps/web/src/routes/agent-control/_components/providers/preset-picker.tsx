import { useT } from "@trenova/shared/i18n/use-t";
import { BrandLogo } from "@trenova/shared/components/brand-logo";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import type { AIProviderPreset } from "@/types/ai-provider";
import { CheckIcon, SlidersHorizontalIcon } from "lucide-react";

type PresetPickerProps = {
  presets: readonly AIProviderPreset[];
  value: string;
  isLoading?: boolean;
  onSelect: (preset: AIProviderPreset | null) => void;
};

/**
 * A grid of known deployments. Picking one fills in the endpoint and output
 * settings; nothing is locked, and "Custom" keeps whatever is already typed.
 */
export function PresetPicker({ presets, value, isLoading = false, onSelect }: PresetPickerProps) {
  const t = useT();

  if (isLoading) {
    return (
      <div className="grid grid-cols-2 gap-2 md:grid-cols-3">
        {Array.from({ length: 6 }).map((_, index) => (
          <Skeleton key={index} className="h-16" />
        ))}
      </div>
    );
  }

  return (
    <div
      role="radiogroup"
      aria-label={t("Preset")}
      className="grid grid-cols-2 gap-2 md:grid-cols-3"
    >
      <PresetTile
        selected={value === ""}
        onClick={() => onSelect(null)}
        logo={
          <span className="bg-muted text-muted-foreground ring-border flex size-8 items-center justify-center rounded-lg ring-1">
            <SlidersHorizontalIcon className="size-4" />
          </span>
        }
        label={t("Custom")}
        caption={t("Any OpenAI-compatible server")}
      />
      {presets.map((preset) => (
        <PresetTile
          key={preset.key}
          selected={value === preset.key}
          onClick={() => onSelect(preset)}
          logo={<BrandLogo domain={preset.domain} name={preset.label} size={32} />}
          label={preset.label.replace(/\s*\(self-hosted\)$/i, "")}
          caption={preset.selfHosted ? t("Self-hosted") : preset.exampleModel || t("Hosted API")}
          badge={preset.selfHosted ? t("Local") : undefined}
        />
      ))}
    </div>
  );
}

type PresetTileProps = {
  selected: boolean;
  onClick: () => void;
  logo: React.ReactNode;
  label: string;
  caption: string;
  badge?: string;
};

function PresetTile({ selected, onClick, logo, label, caption, badge }: PresetTileProps) {
  return (
    <button
      type="button"
      role="radio"
      aria-checked={selected}
      onClick={onClick}
      className={cn(
        "bg-card hover:bg-muted/50 focus-visible:ring-ring/50 relative flex items-center gap-2.5 rounded-lg border p-2.5 text-left transition-colors outline-none focus-visible:ring-[3px]",
        selected ? "border-primary ring-primary/20 ring-2" : "border-border",
      )}
    >
      {logo}
      <span className="min-w-0 flex-1">
        <span className="flex items-center gap-1.5">
          <span className="truncate text-sm font-medium">{label}</span>
          {badge && (
            <Badge variant="outline" className="h-4 px-1 text-[10px]">
              {badge}
            </Badge>
          )}
        </span>
        <span className="text-muted-foreground block truncate font-mono text-[11px]">
          {caption}
        </span>
      </span>
      {selected && (
        <span className="bg-primary text-primary-foreground absolute top-1.5 right-1.5 flex size-4 items-center justify-center rounded-full">
          <CheckIcon className="size-3" />
        </span>
      )}
    </button>
  );
}
