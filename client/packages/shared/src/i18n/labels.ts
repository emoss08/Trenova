// labels.ts makes a module-level label map read in the language on screen.
//
// Status badges, select options and enum captions are declared once at module scope:
//
//   export const TRAINING_DELIVERY_LABELS = defineLabels({ Online: "Online", OnTheJob: "On the job" });
//
// A module-level translate() would freeze whatever language was active when the module was
// imported, and a plain map never reaches a catalog at all, so both rendered English to every
// reader. defineLabels keeps the English source as the declaration — the extractor collects
// it from the call — and looks the translation up each time a value is read, so
// `TRAINING_DELIVERY_LABELS[delivery]` in a render, Object.entries over the map for a select,
// and a language switch all see the current language.
//
// Read a label where it is shown. Copying values into another module-level structure (a
// select's options built at import) takes them in the language active at import; such data
// holds the English source instead — sourceLabels(map) — and its renderer translates it, the
// same contract as a `label: "…"` literal in an options array.
import { getLocale, lookupIn } from "@trenova/shared/i18n/runtime";

export type LabelMap<T extends Readonly<Record<string, string>>> = {
  readonly [K in keyof T]: string;
};

const sources = new WeakMap<object, Readonly<Record<string, string>>>();

export function defineLabels<const T extends Readonly<Record<string, string>>>(
  labels: T,
): LabelMap<T> {
  const map = {} as Record<string, string>;
  for (const key of Object.keys(labels)) {
    const source = labels[key];
    Object.defineProperty(map, key, {
      enumerable: true,
      get: () => lookupIn(getLocale(), source),
    });
  }
  Object.freeze(map);
  sources.set(map, Object.freeze({ ...labels }));
  return map as LabelMap<T>;
}

/**
 * sourceLabels returns a label map's English source, for data built once at import whose
 * renderer translates at render.
 */
export function sourceLabels<T extends Readonly<Record<string, string>>>(
  map: LabelMap<T>,
): { readonly [K in keyof T]: string } {
  const source = sources.get(map);
  if (source === undefined) {
    throw new Error("sourceLabels() needs a map declared with defineLabels()");
  }
  return source as { readonly [K in keyof T]: string };
}

/**
 * translateLabel looks up a caption computed at runtime — a field key or an enum value turned
 * into words ("In transit") — and returns it as written when the catalog has no entry. The
 * set such a helper produces is open, so a value nobody has rendered before stays English
 * rather than failing; every caption the catalog already holds reads translated.
 */
export function translateLabel(label: string): string {
  return label === "" ? label : lookupIn(getLocale(), label);
}
