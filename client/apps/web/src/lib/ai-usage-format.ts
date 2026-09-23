import { formatCurrency } from "@trenova/shared/lib/utils";

/**
 * A latency in the unit a person reads it in: milliseconds below a second,
 * seconds with one decimal above, so "640 ms" and "1.8 s" rather than "1840".
 */
export function formatLatency(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) {
    return "—";
  }
  if (ms < 1000) {
    return `${Math.round(ms)} ms`;
  }
  const seconds = ms / 1000;
  return `${seconds < 10 ? seconds.toFixed(1) : Math.round(seconds).toString()} s`;
}

/**
 * How long a step of an agent's work took, at the resolution a person reads
 * a wait in: "<1s", "14s", "2m 5s", "1h 3m". Seconds are whole because the
 * steps are stamped in whole seconds.
 */
export function formatWorkDuration(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 1) {
    return "<1s";
  }
  const whole = Math.floor(seconds);
  if (whole < 60) {
    return `${whole}s`;
  }
  if (whole < 3600) {
    const rest = whole % 60;
    const minutes = Math.floor(whole / 60);
    return rest === 0 ? `${minutes}m` : `${minutes}m ${rest}s`;
  }
  const hours = Math.floor(whole / 3600);
  const minutes = Math.floor((whole % 3600) / 60);

  return minutes === 0 ? `${hours}h` : `${hours}h ${minutes}m`;
}

/**
 * A cost in USD, keeping the cents that matter. A single turn is usually a
 * fraction of a cent, and rounding it to "$0.00" would say it was free.
 */
export function formatUsd(value: string | number | null | undefined): string | null {
  if (value === null || value === undefined || value === "") {
    return null;
  }
  const amount = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(amount)) {
    return null;
  }
  if (amount === 0) {
    return formatCurrency(0);
  }
  if (Math.abs(amount) < 0.01) {
    return `$${amount.toFixed(4)}`;
  }
  return formatCurrency(amount);
}

/** Tokens in thousands or millions once they stop being countable. */
export function formatTokens(count: number): string {
  if (!Number.isFinite(count) || count < 0) {
    return "0";
  }
  if (count >= 1_000_000) {
    return `${(count / 1_000_000).toFixed(count >= 10_000_000 ? 0 : 1)}M`;
  }
  if (count >= 10_000) {
    return `${Math.round(count / 1000)}k`;
  }
  return count.toLocaleString();
}
