import type { AIProviderPreset } from "@/types/ai-provider";

export const CUSTOM_PRESET_VALUE = "";

/** How a preset reads in one line: the model to try, and what it needs from you. */
export type PresetHint = "local" | "key" | "none";

export function presetHint(preset: AIProviderPreset): PresetHint {
  if (preset.selfHosted) {
    return "local";
  }

  return preset.requiresApiKey ? "key" : "none";
}

/** The vendor name without the "(self-hosted)" suffix the catalog carries. */
export function presetDisplayName(preset: AIProviderPreset): string {
  return preset.label.replace(/\s*\(self-hosted\)$/i, "");
}

export type PresetGroupKey = "hosted" | "selfHosted";

export type PresetGroup = {
  key: PresetGroupKey;
  presets: AIProviderPreset[];
};

/**
 * Hosted endpoints and endpoints you run yourself are different decisions, so
 * they are different groups rather than one long list.
 */
export function groupPresets(presets: readonly AIProviderPreset[]): PresetGroup[] {
  const hosted = presets.filter((preset) => !preset.selfHosted);
  const selfHosted = presets.filter((preset) => preset.selfHosted);

  const groups: PresetGroup[] = [];
  if (hosted.length > 0) {
    groups.push({ key: "hosted", presets: hosted });
  }
  if (selfHosted.length > 0) {
    groups.push({ key: "selfHosted", presets: selfHosted });
  }

  return groups;
}

export function findPreset(
  presets: readonly AIProviderPreset[],
  key: string,
): AIProviderPreset | null {
  if (key === CUSTOM_PRESET_VALUE) {
    return null;
  }

  return presets.find((preset) => preset.key === key) ?? null;
}
