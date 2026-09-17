const FALLBACK_TIMEZONES = [
  "UTC",
  "America/New_York",
  "America/Chicago",
  "America/Denver",
  "America/Phoenix",
  "America/Los_Angeles",
  "America/Anchorage",
  "Pacific/Honolulu",
  "America/Toronto",
  "America/Vancouver",
  "America/Mexico_City",
  "Europe/London",
  "Europe/Berlin",
  "Asia/Shanghai",
  "Asia/Tokyo",
  "Australia/Sydney",
];

let cached: string[] | null = null;

/** IANA time zone names the runtime knows, UTC first, sorted for a picker. */
export function listTimezones(): string[] {
  if (cached) {
    return cached;
  }

  let zones: string[] = FALLBACK_TIMEZONES;
  try {
    const supported = (
      Intl as unknown as { supportedValuesOf?: (key: string) => string[] }
    ).supportedValuesOf?.("timeZone");
    if (supported && supported.length > 0) {
      zones = supported;
    }
  } catch {
    zones = FALLBACK_TIMEZONES;
  }

  cached = ["UTC", ...zones.filter((zone) => zone !== "UTC").sort()];
  return cached;
}

/** "America/Los_Angeles" reads as "America / Los Angeles" in a list. */
export function formatTimezoneLabel(zone: string): string {
  return zone.replace(/_/g, " ").replace(/\//g, " / ");
}
