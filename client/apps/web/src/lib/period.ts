import type { TranslateFn } from "@trenova/shared/i18n/use-t";

const MINUTE = 60;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

/**
 * A configured period in words, in the largest unit that divides it exactly: a session lifetime
 * of 43,200 seconds is "12 hours", an idle timeout of 1,800 is "30 minutes" and 90 seconds stays
 * "90 seconds". Settings are written in round units, so this reads them back the way they were
 * set rather than as a mixed "1 hour 30 minutes" a countdown would want.
 */
export function formatPeriod(t: TranslateFn, seconds: number): string {
  const whole = Math.max(0, Math.floor(seconds));
  if (whole > 0 && whole % DAY === 0) {
    return t("{0, plural, one {# day} other {# days}}", whole / DAY);
  }
  if (whole > 0 && whole % HOUR === 0) {
    return t("{0, plural, one {# hour} other {# hours}}", whole / HOUR);
  }
  if (whole > 0 && whole % MINUTE === 0) {
    return t("{0, plural, one {# minute} other {# minutes}}", whole / MINUTE);
  }

  return t("{0, plural, one {# second} other {# seconds}}", whole);
}
