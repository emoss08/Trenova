import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort, formatUnixTime } from "@trenova/shared/lib/date";

/** When a turn happened, as its row says it: the time today, the date and time before. */
export function turnTime(at: number, timezone: string, t: TranslateFn): string {
  const today = new Date().toLocaleDateString("en-CA", { timeZone: timezone });
  const day = new Date(at * 1000).toLocaleDateString("en-CA", { timeZone: timezone });
  if (day === today) {
    return formatUnixTime(at, { timezone }) || t("Now");
  }

  return formatUnixDateTimeShort(at, { timezone });
}
