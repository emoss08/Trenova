import { useT } from "@trenova/shared/i18n/use-t";
import { SelectField } from "@/components/fields/select-field";
import { BrandLogo } from "@trenova/shared/components/brand-logo";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { SelectOptionGroup } from "@trenova/shared/types/fields";
import type { AIProviderPreset } from "@/types/ai-provider";
import { SlidersHorizontalIcon } from "lucide-react";
import { useCallback, useMemo } from "react";
import type { Control } from "react-hook-form";
import type { ProviderFormValues } from "./build-save-payload";
import {
  CUSTOM_PRESET_VALUE,
  findPreset,
  groupPresets,
  presetDisplayName,
  presetHint,
} from "./preset-options";

type PresetPickerProps = {
  control: Control<ProviderFormValues>;
  presets: readonly AIProviderPreset[];
  isLoading?: boolean;
  onSelect: (preset: AIProviderPreset | null) => void;
};

const GROUP_LABEL: Record<string, string> = {
  hosted: "Hosted endpoints",
  selfHosted: "Run it yourself",
};

/**
 * One field rather than a wall of tiles. Picking a deployment fills in the
 * endpoint and output settings; nothing is locked, and Custom keeps whatever is
 * already typed.
 */
export function PresetPicker({ control, presets, isLoading = false, onSelect }: PresetPickerProps) {
  const t = useT();

  const byKey = useMemo(() => {
    const map = new Map<string, AIProviderPreset>();
    for (const preset of presets) {
      map.set(preset.key, preset);
    }

    return map;
  }, [presets]);

  const groups = useMemo<SelectOptionGroup[]>(() => {
    const custom: SelectOptionGroup = {
      label: "",
      options: [
        {
          value: CUSTOM_PRESET_VALUE,
          label: "Custom endpoint",
          description: "Any OpenAI-compatible server",
          icon: <SlidersHorizontalIcon className="text-muted-foreground size-[18px] shrink-0" />,
        },
      ],
    };

    return [
      custom,
      ...groupPresets(presets).map((group) => ({
        label: GROUP_LABEL[group.key] ?? "",
        options: group.presets.map((preset) => ({
          value: preset.key,
          label: presetDisplayName(preset),
          description: preset.exampleModel || preset.baseUrl,
          icon: (
            <BrandLogo
              presetKey={preset.key}
              domain={preset.domain}
              name={preset.label}
              size={18}
            />
          ),
        })),
      })),
    ];
  }, [presets]);

  const handleChange = useCallback(
    (value: string) => onSelect(findPreset(presets, value)),
    [onSelect, presets],
  );

  return (
    <SelectField
      control={control}
      name="preset"
      label={t("Deployment")}
      placeholder={isLoading ? t("Loading deployments…") : t("Choose a deployment")}
      isReadOnly={isLoading}
      groups={groups}
      onValueChange={handleChange}
      renderOption={(option) => {
        const preset = byKey.get(String(option.value));
        const hint = preset ? presetHint(preset) : "none";

        return (
          <span className="flex min-w-0 flex-1 items-center gap-2">
            {option.icon}
            <span className="min-w-0 flex-1">
              <span className="flex items-center gap-1.5">
                <span className="truncate font-medium">{t(option.label)}</span>
                {hint === "local" && (
                  <Badge variant="neutral" appearance="outline" className="h-4 px-1 text-2xs">
                    {t("Local")}
                  </Badge>
                )}
                {hint === "key" && (
                  <Badge variant="danger" className="h-4 px-1 text-2xs">
                    {t("Key")}
                  </Badge>
                )}
              </span>
              {option.description && (
                <span className="text-muted-foreground block truncate font-mono text-2xs">
                  {option.description}
                </span>
              )}
            </span>
          </span>
        );
      }}
    />
  );
}
