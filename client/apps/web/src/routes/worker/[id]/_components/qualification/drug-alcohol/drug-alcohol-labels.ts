import type { TranslateFn } from "@trenova/shared/i18n/use-t";

/** What a test analyses, in the words the tables and panes use. */
export function substanceLabel(t: TranslateFn, substance: string): string {
  return substance === "Alcohol" ? t("Alcohol") : t("Controlled substances");
}
