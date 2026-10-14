/**
 * One calendar event as an iCalendar (.ics) file, which every calendar app
 * imports: a delivery appointment, a pickup window, an expected arrival.
 */
export type CalendarEvent = {
  /** Stable across downloads, so importing twice updates the event rather than adding a copy. */
  uid: string;
  title: string;
  /** Unix seconds. */
  start: number;
  /** Unix seconds; defaults to half an hour after start. */
  end?: number;
  location?: string;
  description?: string;
  url?: string;
};

function stamp(unix: number): string {
  return new Date(unix * 1000).toISOString().replace(/[-:]/g, "").replace(/\.\d{3}Z$/, "Z");
}

/** Escapes text as RFC 5545 requires: backslash, semicolon, comma and line breaks. */
function escapeText(value: string): string {
  return value
    .replace(/\\/g, "\\\\")
    .replace(/;/g, "\\;")
    .replace(/,/g, "\\,")
    .replace(/\r?\n/g, "\\n");
}

/** Folds a content line at 75 octets, continuing with a leading space. */
function fold(line: string): string {
  const parts: string[] = [];
  let rest = line;
  while (rest.length > 75) {
    parts.push(rest.slice(0, 75));
    rest = ` ${rest.slice(75)}`;
  }
  parts.push(rest);
  return parts.join("\r\n");
}

export function calendarFile(event: CalendarEvent, now: number = Date.now() / 1000): string {
  const lines = [
    "BEGIN:VCALENDAR",
    "VERSION:2.0",
    "PRODID:-//Trenova//Shipment tracking//EN",
    "CALSCALE:GREGORIAN",
    "METHOD:PUBLISH",
    "BEGIN:VEVENT",
    `UID:${escapeText(event.uid)}`,
    `DTSTAMP:${stamp(Math.floor(now))}`,
    `DTSTART:${stamp(event.start)}`,
    `DTEND:${stamp(event.end ?? event.start + 1800)}`,
    `SUMMARY:${escapeText(event.title)}`,
  ];
  if (event.location) lines.push(`LOCATION:${escapeText(event.location)}`);
  if (event.description) lines.push(`DESCRIPTION:${escapeText(event.description)}`);
  if (event.url) lines.push(`URL:${event.url}`);
  lines.push("END:VEVENT", "END:VCALENDAR");
  return `${lines.map(fold).join("\r\n")}\r\n`;
}

/** Hands the event to the browser as a download. */
export function downloadCalendarFile(event: CalendarEvent, fileName: string): void {
  const blob = new Blob([calendarFile(event)], { type: "text/calendar;charset=utf-8" });
  const href = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = href;
  anchor.download = fileName.endsWith(".ics") ? fileName : `${fileName}.ics`;
  anchor.click();
  window.setTimeout(() => URL.revokeObjectURL(href), 0);
}
