/**
 * Gradient avatars for people without a photo. The default is drawn from the
 * person's username, so it is stable and different for everyone (the way
 * avatar.vercel.sh works); a person may pick one of the named presets instead.
 */

export const AVATAR_STYLE_AUTO = "auto";

export type AvatarHues = readonly [number, number, number];

export type AvatarPreset = {
  id: string;
  label: string;
  hues: AvatarHues;
  chroma?: number;
};

export const AVATAR_PRESETS: readonly AvatarPreset[] = [
  { id: "dusk", label: "Dusk", hues: [258, 306, 20] },
  { id: "lagoon", label: "Lagoon", hues: [190, 232, 148] },
  { id: "citrus", label: "Citrus", hues: [76, 40, 120] },
  { id: "orchid", label: "Orchid", hues: [306, 340, 258] },
  { id: "ember", label: "Ember", hues: [25, 55, 350] },
  { id: "mint", label: "Mint", hues: [160, 120, 200] },
  { id: "slate", label: "Slate", hues: [240, 270, 210], chroma: 0.06 },
];

export const AVATAR_STYLES: readonly string[] = [
  AVATAR_STYLE_AUTO,
  ...AVATAR_PRESETS.map((preset) => preset.id),
];

const DEFAULT_CHROMA = 0.17;

export function isAvatarStyle(value: unknown): value is string {
  return typeof value === "string" && AVATAR_STYLES.includes(value);
}

/** FNV-1a over UTF-16 code units, as an unsigned 32-bit integer. */
export function fnv1a(input: string): number {
  let hash = 2166136261;
  for (let index = 0; index < input.length; index++) {
    hash ^= input.charCodeAt(index);
    hash = Math.imul(hash, 16777619);
  }
  return hash >>> 0;
}

export function avatarSeedHues(seed: string): AvatarHues {
  const hash = fnv1a(seed);
  const a = hash % 360;
  const b = (a + 40 + ((hash >>> 9) % 50)) % 360;
  const c = (a + 300 - ((hash >>> 17) % 60)) % 360;
  return [a, b, c];
}

/** Four radial washes over a 135° base: the mesh every gradient avatar is drawn with. */
export function avatarMesh([a, b, c]: AvatarHues, chroma: number = DEFAULT_CHROMA): string {
  const k = chroma;
  const mid = (a + b) / 2;
  return [
    // i18n-ignore: CSS gradient syntax, not interface text
    `radial-gradient(at 18% 22%, oklch(0.82 ${k} ${a}) 0, transparent 58%)`,
    // i18n-ignore: CSS gradient syntax, not interface text
    `radial-gradient(at 86% 18%, oklch(0.74 ${round(k + 0.02)} ${b}) 0, transparent 55%)`,
    // i18n-ignore: CSS gradient syntax, not interface text
    `radial-gradient(at 70% 88%, oklch(0.66 ${round(k + 0.02)} ${c}) 0, transparent 60%)`,
    // i18n-ignore: CSS gradient syntax, not interface text
    `radial-gradient(at 12% 92%, oklch(0.88 ${round(k * 0.7)} ${mid}) 0, transparent 50%)`,
    // i18n-ignore: CSS gradient syntax, not interface text
    `linear-gradient(135deg, oklch(0.8 ${k} ${a}), oklch(0.68 ${k} ${c}))`,
  ].join(",");
}

function round(value: number): number {
  return Math.round(value * 1000) / 1000;
}

export function avatarPreset(style: string | null | undefined): AvatarPreset | undefined {
  return AVATAR_PRESETS.find((preset) => preset.id === style);
}

/** The background for a person's avatar: their preset, or the gradient their username seeds. */
export function avatarBackground(seed: string, style?: string | null): string {
  const preset = avatarPreset(style);
  return preset ? avatarMesh(preset.hues, preset.chroma) : avatarMesh(avatarSeedHues(seed));
}
