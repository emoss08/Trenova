import { describe, expect, it } from "vitest";
import { calendarFile } from "./calendar-file";

const START = Date.UTC(2026, 9, 14, 21, 44, 0) / 1000;

function lines(ics: string): string[] {
  return ics.replace(/\r\n /g, "").split("\r\n").filter(Boolean);
}

describe("calendarFile", () => {
  it("writes one event with UTC times and CRLF line endings", () => {
    const ics = calendarFile(
      { uid: "S2610-0398-3@trenova", title: "Delivery of load S2610-0398", start: START },
      START,
    );

    expect(ics.startsWith("BEGIN:VCALENDAR\r\n")).toBe(true);
    expect(ics.endsWith("END:VCALENDAR\r\n")).toBe(true);
    expect(lines(ics)).toEqual(
      expect.arrayContaining([
        "UID:S2610-0398-3@trenova",
        "DTSTART:20261014T214400Z",
        "DTEND:20261014T221400Z",
        "SUMMARY:Delivery of load S2610-0398",
      ]),
    );
  });

  it("ends at the appointment's end when one is given", () => {
    const ics = calendarFile({ uid: "u", title: "t", start: START, end: START + 3600 }, START);
    expect(lines(ics)).toContain("DTEND:20261014T224400Z");
  });

  it("escapes the characters iCalendar reserves in text", () => {
    const ics = calendarFile(
      {
        uid: "u",
        title: "Pickup; dock 4, door B",
        start: START,
        location: "North Star Grocers DC, 1200 Industrial Blvd NE",
        description: "Line one\nLine two \\ done",
      },
      START,
    );
    const body = lines(ics);
    expect(body).toContain("SUMMARY:Pickup\\; dock 4\\, door B");
    expect(body).toContain("LOCATION:North Star Grocers DC\\, 1200 Industrial Blvd NE");
    expect(body).toContain("DESCRIPTION:Line one\\nLine two \\\\ done");
  });

  it("folds long lines at 75 characters", () => {
    const ics = calendarFile({ uid: "u", title: "x".repeat(200), start: START }, START);
    for (const raw of ics.split("\r\n")) {
      expect(raw.length).toBeLessThanOrEqual(75);
    }
    expect(lines(ics)).toContain(`SUMMARY:${"x".repeat(200)}`);
  });
});
