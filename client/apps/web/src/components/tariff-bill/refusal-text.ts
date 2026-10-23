import { translateMessage } from "@trenova/shared/i18n/message";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import type { CatalogMessage } from "@trenova/shared/i18n/message";
import type { TariffRefusal } from "@trenova/shared/types/tariff";

/** A message the server or the module sent: text already in the reader's words, or a catalog message. */
export function tariffMessageText(message: string | CatalogMessage, t: TranslateFn): string {
  return typeof message === "string" ? message : translateMessage(message, t);
}

/**
 * Why a tariff refused to rate, as the reader sees it: the author's words from `fail` or
 * `require` are shown as written (they are the tariff's text, like a charge label); a refusal
 * the module wrote (an input nothing set) is translated.
 */
export function refusalText(refusal: TariffRefusal, t: TranslateFn): string {
  return tariffMessageText(refusal.message, t);
}
