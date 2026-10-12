import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { translate } from "@trenova/shared/i18n/runtime";
import { formatDateInUserTimezone } from "@trenova/shared/lib/date";

function formatElapsedAgo(elapsedSeconds: number): string {
  if (elapsedSeconds < 60) return formatRelativeTime(-elapsedSeconds, "narrow");
  if (elapsedSeconds < 3600)
    return formatRelativeTime(-Math.floor(elapsedSeconds / 60) * 60, "narrow");
  if (elapsedSeconds < 86400)
    return formatRelativeTime(-Math.floor(elapsedSeconds / 3600) * 3600, "narrow");
  return formatRelativeTime(-Math.floor(elapsedSeconds / 86400) * 86400, "narrow");
}

export function formatElapsedTime(timestamp: number, now = Date.now()) {
  if (timestamp <= 0) return translate("just now");

  const elapsedSeconds = Math.max(0, Math.floor((now - timestamp) / 1000));
  if (elapsedSeconds < 1) return translate("just now");
  if (elapsedSeconds < 60) return formatElapsedAgo(elapsedSeconds);

  return formatRelativeTime(-Math.floor(elapsedSeconds / 60) * 60, "narrow");
}

export function formatPreciseTimeAgo(timestamp: number, now = Date.now()): string {
  const elapsedSeconds = Math.max(0, Math.floor((now - timestamp) / 1000));
  return formatElapsedAgo(elapsedSeconds);
}

export function formatTimeAgo(timestamp: number, now = Date.now()): string {
  const elapsedSeconds = Math.max(0, Math.floor((now - timestamp) / 1000));
  if (elapsedSeconds < 60) return translate("just now");
  if (elapsedSeconds < 604800) return formatElapsedAgo(elapsedSeconds);
  return formatDateInUserTimezone(new Date(timestamp), {
    month: "numeric",
    day: "numeric",
    year: "numeric",
  });
}
