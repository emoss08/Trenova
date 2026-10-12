import type { TriageStopTimes } from "@/lib/graphql/inbox";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";

/**
 * The stop times a detention dispute turns on, read the way a biller reads
 * them: the window, when the truck arrived and by whose clock, when it was
 * checked in and when it left, and one sentence on what that means for the
 * charge. The sentence is worked out here from the times themselves, so the
 * grid and the words cannot disagree.
 */
export type StopFormat = {
  /** A time of day in the reader's zone: "10:40 AM". */
  time: (seconds: number) => string;
};

export type StopRow = {
  key: "appointment" | "arrived" | "checkedIn" | "departed";
  label: string;
  value: string;
  /** The arrival came after the window closed. */
  late: boolean;
};

const MINUTE = 60;
const NOT_RECORDED_SOURCE = "not recorded";

/**
 * Minutes between the window closing and the truck arriving: zero inside the
 * window, null with no arrival. Measured from the two times where both are
 * known; the server's own figure stands in only when there is no window.
 */
export function lateMinutesOf(stops: TriageStopTimes): number | null {
  if (stops.arrivedAt === null) {
    return null;
  }
  if (stops.appointmentEnd === null) {
    return Math.max(0, stops.lateMinutes);
  }

  return Math.max(0, Math.ceil((stops.arrivedAt - stops.appointmentEnd) / MINUTE));
}

function duration(t: TranslateFn, minutes: number): string {
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;
  if (hours === 0) {
    return t("{0, plural, one {# minute} other {# minutes}}", rest);
  }
  if (rest === 0) {
    return t("{0, plural, one {# hour} other {# hours}}", hours);
  }

  return t(
    "{0, plural, one {# hour} other {# hours}} {1, plural, one {# minute} other {# minutes}}",
    hours,
    rest,
  );
}

function arrivalLabel(t: TranslateFn, source: string): string {
  const trimmed = source.trim();
  if (trimmed === "" || trimmed.toLowerCase() === NOT_RECORDED_SOURCE) {
    return t("Arrived");
  }

  return t("Arrived ({0})", trimmed);
}

function windowText(t: TranslateFn, stops: TriageStopTimes, format: StopFormat): string {
  const { appointmentStart: start, appointmentEnd: end } = stops;
  if (start !== null && end !== null) {
    return t("{0} – {1}", format.time(start), format.time(end));
  }
  if (end !== null) {
    return t("By {0}", format.time(end));
  }
  if (start !== null) {
    return t("From {0}", format.time(start));
  }

  return t("Not recorded");
}

/** The four rows of the stop-times grid. */
export function stopRows(t: TranslateFn, stops: TriageStopTimes, format: StopFormat): StopRow[] {
  const at = (seconds: number | null) =>
    seconds === null ? t("Not recorded") : format.time(seconds);
  const late = lateMinutesOf(stops);

  return [
    {
      key: "appointment",
      label: t("Appointment window"),
      value: windowText(t, stops, format),
      late: false,
    },
    {
      key: "arrived",
      label: arrivalLabel(t, stops.arrivalSource),
      value: at(stops.arrivedAt),
      late: late !== null && late > 0,
    },
    { key: "checkedIn", label: t("Checked in at gate"), value: at(stops.checkedInAt), late: false },
    { key: "departed", label: t("Unloaded and out"), value: at(stops.departedAt), late: false },
  ];
}

/** One sentence on what the stop times mean for the detention charge. */
export function stopsReading(
  t: TranslateFn,
  stops: TriageStopTimes,
  customerName: string | null,
  format: StopFormat,
): string {
  if (stops.arrivedAt === null) {
    return t(
      "There's no arrival time on record for {0}, so the stop times can't settle it.",
      stops.stopName.trim() === "" ? t("this stop") : stops.stopName,
    );
  }
  if (stops.appointmentStart === null && stops.appointmentEnd === null) {
    return t(
      "The truck arrived at {0}, but there's no appointment window on record to measure it against.",
      format.time(stops.arrivedAt),
    );
  }

  const late = lateMinutesOf(stops) ?? 0;
  const customer = customerName?.trim() ? customerName.trim() : null;
  if (late > 0) {
    if (stops.detentionPaysWhenLate) {
      return customer === null
        ? t(
            "The truck arrived {0} after the window, but the customer pays detention on a late arrival too, so the charge can stand.",
            duration(t, late),
          )
        : t(
            "The truck arrived {0} after the window, but {1} pays detention on a late arrival too, so the charge can stand.",
            duration(t, late),
            customer,
          );
    }
    return customer === null
      ? t(
          "The truck arrived {0} after the window. The customer only pays detention on an on-time arrival, so the charge is unlikely to stand.",
          duration(t, late),
        )
      : t(
          "The truck arrived {0} after the window. {1} only pays detention on an on-time arrival, so the charge is unlikely to stand.",
          duration(t, late),
          customer,
        );
  }
  if (stops.billedDetentionMinutes > 0) {
    return t(
      "The truck arrived inside the window and was held {0}, so the charge stands.",
      duration(t, stops.billedDetentionMinutes),
    );
  }

  return t("The truck arrived inside the window.");
}
