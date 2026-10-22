import { translateMessage } from "@trenova/shared/i18n/message";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import type { TariffRefusal } from "@trenova/shared/types/tariff";

/**
 * Why a tariff refused to rate, as the reader sees it: the author's words from `fail` or
 * `require` are shown as written (they are the tariff's text, like a charge label); a refusal
 * the module wrote (an input nothing set) is translated.
 */
export function refusalText(refusal: TariffRefusal, t: TranslateFn): string {
  return typeof refusal.message === "string"
    ? refusal.message
    : translateMessage(refusal.message, t);
}
