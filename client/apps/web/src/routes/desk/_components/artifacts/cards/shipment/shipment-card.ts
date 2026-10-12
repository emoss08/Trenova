import { formatList } from "@trenova/shared/i18n/format";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { reportNumericValue } from "@trenova/shared/lib/report-format";
import { clampUnit, formatPercent, getNameInitials } from "@trenova/shared/lib/utils";
import {
  additionalChargeLineTotal,
  isActiveCarrierAssignment,
  type AssignmentStatus,
  type CarrierAssignmentStatus,
  type Shipment,
  type ShipmentBillingIneligibleDocument,
  type ShipmentBillingReadiness,
  type ShipmentBillingRequirement,
  type ShipmentEta,
  type ShipmentMove,
  type ShipmentStage,
  type ShipmentTenderStatus,
  type Stop,
  type StopType,
} from "@trenova/shared/types/shipment";
import type { CardCheckState, CardTone } from "../../record-card/record-card";
import { cardDay, cardMoment, money } from "../../record-card/record-card-values";

const SECONDS_PER_MINUTE = 60;
const MINUTES_PER_HOUR = 60;
const MINUTES_PER_DAY = 1440;

const CLOCK: Intl.DateTimeFormatOptions = { hour: "numeric", minute: "2-digit" };
const MOMENT: Intl.DateTimeFormatOptions = {
  month: "short",
  day: "numeric",
  hour: "numeric",
  minute: "2-digit",
};

/** Where a stop is, as the card names it. */
export type StopPlace = {
  name: string;
  city: string;
  state: string;
  /** "Dallas, TX", or the name */
  label: string;
};

export type StopPhase = "done" | "current" | "next" | "later" | "canceled";

export type StopPunctuality = { state: "early" | "onTime" | "late"; minutes: number } | null;

/** One stop of the journey, read for what a dispatcher asks of it. */
export type JourneyStop = {
  id: string;
  moveId: string;
  /** 0-based move position */
  moveIndex: number;
  /** 0-based across the journey */
  index: number;
  type: StopType;
  place: StopPlace;
  appointment: boolean;
  windowStart: number | null;
  windowEnd: number | null;
  arrivedAt: number | null;
  departedAt: number | null;
  phase: StopPhase;
  punctuality: StopPunctuality;
  dwellMinutes: number | null;
  pieces: number | null;
  weight: number | null;
};

function nowSeconds(): number {
  return Date.now() / 1000;
}

/** A unix second the server actually set; zero is how an unset one arrives. */
function instant(seconds: number | null | undefined): number | null {
  return typeof seconds === "number" && Number.isFinite(seconds) && seconds > 0 ? seconds : null;
}

/** A distance worth weighing by: a figure above zero, or nothing. */
function positive(value: unknown): number | null {
  const figure = reportNumericValue(value);
  return figure !== null && figure > 0 ? figure : null;
}

function wholeMinutes(seconds: number): number {
  return Math.floor(seconds / SECONDS_PER_MINUTE);
}

function inMoveOrder(moves: readonly ShipmentMove[]): ShipmentMove[] {
  return [...moves].sort((a, b) => (a.sequence ?? 0) - (b.sequence ?? 0));
}

function inStopOrder(stops: readonly Stop[] | null | undefined): Stop[] {
  return [...(stops ?? [])].sort((a, b) => (a.sequence ?? 0) - (b.sequence ?? 0));
}

function placeOf(stop: Stop): StopPlace {
  const name = stop.location?.name ?? "";
  const city = stop.location?.city ?? "";
  const state = stop.location?.state?.abbreviation ?? "";
  const label =
    city && state ? `${city}, ${state}` : city || name || (stop.addressLine ?? "").trim();
  return { name, city, state, label };
}

/** A window as it stands, with an end that comes before its start read as no end. */
function windowOf(stop: Stop): { start: number | null; end: number | null } {
  const start = instant(stop.scheduledWindowStart);
  const end = instant(stop.scheduledWindowEnd);
  return { start, end: end !== null && start !== null && end < start ? null : end };
}

/**
 * How late a stop nobody has reached yet is running: by how long its window
 * has been shut. A window still open, or none at all, says nothing.
 */
function missedWindow(
  slot: { start: number | null; end: number | null },
  now: number,
): StopPunctuality {
  const deadline = slot.end ?? slot.start;
  if (deadline === null) return null;
  const over = wholeMinutes(now - deadline);
  return over > 0 ? { state: "late", minutes: over } : null;
}

function arrivalPunctuality(
  slot: { start: number | null; end: number | null },
  arrivedAt: number,
): StopPunctuality {
  const deadline = slot.end ?? slot.start;
  if (deadline === null) return null;
  const late = wholeMinutes(arrivedAt - deadline);
  if (late > 0) return { state: "late", minutes: late };
  const early = slot.start === null ? 0 : wholeMinutes(slot.start - arrivedAt);
  if (early > 0) return { state: "early", minutes: early };
  return { state: "onTime", minutes: 0 };
}

/**
 * Moves by sequence, each move's stops by sequence, flattened. The first stop not completed or
 * canceled is "current" when it has an arrival and no departure, else "next"; completed are
 * "done", canceled "canceled", the rest "later". A stop on a canceled move is canceled with it.
 * Punctuality: arrival vs window (end, or start when no end): late when after it, early when
 * before the start, else onTime; a stop with no arrival whose window has closed by `now` is late
 * by now − end. Dwell: departure − arrival.
 */
export function journeyStops(
  moves: readonly ShipmentMove[],
  now: number = nowSeconds(),
): JourneyStop[] {
  const journey: JourneyStop[] = [];
  let openFound = false;

  inMoveOrder(moves).forEach((move, moveIndex) => {
    const moveCanceled = move.status === "Canceled";
    inStopOrder(move.stops).forEach((stop, position) => {
      const slot = windowOf(stop);
      const arrivedAt = instant(stop.actualArrival);
      const departedAt = instant(stop.actualDeparture);

      let phase: StopPhase;
      if (moveCanceled || stop.status === "Canceled") {
        phase = "canceled";
      } else if (stop.status === "Completed") {
        phase = "done";
      } else if (openFound) {
        phase = "later";
      } else {
        openFound = true;
        phase = arrivedAt !== null && departedAt === null ? "current" : "next";
      }

      let punctuality: StopPunctuality = null;
      if (phase !== "canceled") {
        if (arrivedAt !== null) {
          punctuality = arrivalPunctuality(slot, arrivedAt);
        } else if (phase !== "done") {
          punctuality = missedWindow(slot, now);
        }
      }

      const dwellMinutes =
        arrivedAt !== null && departedAt !== null && departedAt >= arrivedAt
          ? Math.round((departedAt - arrivedAt) / SECONDS_PER_MINUTE)
          : null;

      journey.push({
        id: stop.id ?? `${moveIndex}-${position}`,
        moveId: move.id ?? "",
        moveIndex,
        index: journey.length,
        type: stop.type ?? "Pickup",
        place: placeOf(stop),
        appointment: stop.scheduleType === "Appointment",
        windowStart: slot.start,
        windowEnd: slot.end,
        arrivedAt,
        departedAt,
        phase,
        punctuality,
        dwellMinutes,
        pieces: reportNumericValue(stop.pieces),
        weight: reportNumericValue(stop.weight),
      });
    });
  });

  return journey;
}

/** How much of a run of stops is behind the truck: a stop done is whole, one it sits at half. */
function stopShare(stops: readonly JourneyStop[]): number {
  if (stops.length === 0) return 0;
  let covered = 0;
  for (const stop of stops) {
    if (stop.phase === "done") covered += 1;
    else if (stop.phase === "current") covered += 0.5;
  }
  return covered / stops.length;
}

/**
 * 0..1 along the journey: by move distance, loaded or empty, when every move still running has
 * one, else by stops (done = 1, current = 0.5 of its leg). 1 when every stop is done, 0 with no
 * stops. Canceled stops and moves are not part of the way.
 */
export function journeyProgress(
  moves: readonly ShipmentMove[],
  stops: readonly JourneyStop[],
): number {
  const live = stops.filter((stop) => stop.phase !== "canceled");
  if (live.length === 0) return 0;
  if (live.every((stop) => stop.phase === "done")) return 1;

  const running = inMoveOrder(moves)
    .map((move, moveIndex) => ({ move, moveIndex, distance: positive(move.distance) }))
    .filter(({ move }) => move.status !== "Canceled");

  if (running.length > 0 && running.every(({ distance }) => distance !== null)) {
    let total = 0;
    let covered = 0;
    for (const { move, moveIndex, distance } of running) {
      const miles = distance ?? 0;
      const own = live.filter((stop) => stop.moveIndex === moveIndex);
      const share = own.length === 0 ? (move.status === "Completed" ? 1 : 0) : stopShare(own);
      total += miles;
      covered += miles * share;
    }
    return total > 0 ? clampUnit(covered / total) : clampUnit(stopShare(live));
  }

  return clampUnit(stopShare(live));
}

function isPickup(type: StopType): boolean {
  return type === "Pickup" || type === "SplitPickup";
}

function isDelivery(type: StopType): boolean {
  return type === "Delivery" || type === "SplitDelivery";
}

/** The stops still part of the journey, or every stop when all of them were canceled. */
function standingStops(stops: readonly JourneyStop[]): readonly JourneyStop[] {
  const live = stops.filter((stop) => stop.phase !== "canceled");
  return live.length > 0 ? live : stops;
}

function lastDelivery(stops: readonly JourneyStop[]): JourneyStop | null {
  const pool = standingStops(stops);
  for (let index = pool.length - 1; index >= 0; index--) {
    if (isDelivery(pool[index].type)) return pool[index];
  }
  return pool.at(-1) ?? null;
}

/** First pickup and last delivery (falling back to first and last stop), canceled stops aside. */
export function laneOf(stops: readonly JourneyStop[]): {
  from: StopPlace | null;
  to: StopPlace | null;
} {
  const pool = standingStops(stops);
  const from = pool.find((stop) => isPickup(stop.type)) ?? pool[0] ?? null;
  const to = lastDelivery(stops);
  return { from: from?.place ?? null, to: to?.place ?? null };
}

/** The server's stage, or one derived from status the way the server derives it. */
export function shipmentStage(shipment: Pick<Shipment, "stage" | "status">): ShipmentStage {
  if (shipment.stage) return shipment.stage;
  switch (shipment.status) {
    case "Delayed":
      return "Late";
    case "InTransit":
    case "PartiallyCompleted":
      return "Moving";
    case "Assigned":
      return "Scheduled";
    case "Completed":
    case "ReadyToInvoice":
    case "Invoiced":
      return "Delivered";
    case "Canceled":
      return "Canceled";
    default:
      return "NeedsCoverage";
  }
}

export const STAGE_TONE: Record<ShipmentStage, CardTone> = {
  Late: "warn",
  NeedsCoverage: "draft",
  Moving: "progress",
  Scheduled: "ink",
  Delivered: "success",
  Canceled: "muted",
};

export function stageLabel(stage: ShipmentStage, t: TranslateFn): string {
  switch (stage) {
    case "Late":
      return t("Late");
    case "NeedsCoverage":
      return t("Needs coverage");
    case "Moving":
      return t("Moving");
    case "Scheduled":
      return t("Scheduled");
    case "Delivered":
      return t("Delivered");
    default:
      return t("Canceled");
  }
}

/** A span of minutes the way a dispatcher says it: "45m", "1h 20m", "2d 3h". */
export function durationLabel(minutes: number, t: TranslateFn): string {
  const total = Number.isFinite(minutes) ? Math.abs(Math.round(minutes)) : 0;
  const days = Math.floor(total / MINUTES_PER_DAY);
  if (days > 0) {
    return t("{0}d {1}h", days, Math.floor((total % MINUTES_PER_DAY) / MINUTES_PER_HOUR));
  }
  const hours = Math.floor(total / MINUTES_PER_HOUR);
  return hours > 0 ? t("{0}h {1}m", hours, total % MINUTES_PER_HOUR) : t("{0}m", total);
}

export type EtaReading = {
  state: "ok" | "warn" | "bad" | "unknown";
  label: string;
  detail: string;
  at: number | null;
};

/**
 * "On time" + "1h 20m to spare" / "At risk" + "only 15m to spare" / "Late" + "40m past the
 * window" / "No ETA yet" + reason. The server's reason follows the slack when it gave one.
 * Null when there is no eta.
 */
export function etaReading(eta: ShipmentEta | null | undefined, t: TranslateFn): EtaReading | null {
  if (!eta) return null;
  const at = instant(eta.estimatedArrival);
  const slack =
    typeof eta.slackMinutes === "number" && Number.isFinite(eta.slackMinutes)
      ? eta.slackMinutes
      : null;
  const reason = (eta.reason ?? "").trim();
  const detail = (lead: string) => [lead, reason].filter((part) => part !== "").join(" · ");
  const past = slack !== null && slack < 0 ? t("{0} past the window", durationLabel(slack, t)) : "";

  switch (eta.verdict) {
    case "OnTime":
      return {
        state: "ok",
        label: t("On time"),
        detail: detail(
          slack !== null && slack >= 0 ? t("{0} to spare", durationLabel(slack, t)) : "",
        ),
        at,
      };
    case "AtRisk":
      return {
        state: "warn",
        label: t("At risk"),
        detail: detail(
          slack !== null && slack >= 0 ? t("only {0} to spare", durationLabel(slack, t)) : past,
        ),
        at,
      };
    case "Late":
      return { state: "bad", label: t("Late"), detail: detail(past), at };
    default:
      return { state: "unknown", label: t("No ETA yet"), detail: reason, at };
  }
}

export function stopTypeLabel(type: StopType, t: TranslateFn): string {
  switch (type) {
    case "Delivery":
      return t("Delivery");
    case "SplitDelivery":
      return t("Split delivery");
    case "SplitPickup":
      return t("Split pickup");
    default:
      return t("Pickup");
  }
}

/**
 * "Appt Oct 12, 8:00 – 10:00 AM", "Open Oct 12, 8:00 AM – 4:00 PM", "Appt Oct 12, 8:00 AM" (no
 * end); spans days → "Oct 12, 8:00 AM – Oct 13, 9:00 AM" after the same lead; "" with no window.
 */
export function windowLabel(stop: JourneyStop, t: TranslateFn): string {
  const start = stop.windowStart ?? stop.windowEnd;
  if (start === null) return "";
  const end =
    stop.windowStart !== null && stop.windowEnd !== null && stop.windowEnd > stop.windowStart
      ? stop.windowEnd
      : null;
  const when =
    end === null
      ? cardMoment(start)
      : new Intl.DateTimeFormat(undefined, MOMENT).formatRange(
          new Date(start * 1000),
          new Date(end * 1000),
        );
  return stop.appointment ? t("Appt {0}", when) : t("Open {0}", when);
}

/** "On time", "Early 20m", "Late 1h 5m"; "" for null. */
export function punctualityLabel(punctuality: StopPunctuality, t: TranslateFn): string {
  if (!punctuality) return "";
  switch (punctuality.state) {
    case "early":
      return t("Early {0}", durationLabel(punctuality.minutes, t));
    case "late":
      return t("Late {0}", durationLabel(punctuality.minutes, t));
    default:
      return t("On time");
  }
}

function clock(seconds: number): string {
  return new Date(seconds * 1000).toLocaleTimeString(undefined, CLOCK);
}

function sameDay(a: number, b: number): boolean {
  return new Date(a * 1000).toDateString() === new Date(b * 1000).toDateString();
}

/** A time at a stop, with its day when that is not the day the window names. */
function stopTime(seconds: number, stop: JourneyStop): string {
  const reference = stop.windowStart ?? stop.windowEnd;
  return reference !== null && sameDay(seconds, reference) ? clock(seconds) : cardMoment(seconds);
}

/**
 * "Arrived 8:12 AM · left 9:40 AM · 1h 28m on site", "Arrived 8:12 AM", "" — day added when not
 * the window's day. A departure with no arrival on record reads "Left 9:40 AM".
 */
export function actualsLabel(stop: JourneyStop, t: TranslateFn): string {
  if (stop.arrivedAt === null) {
    return stop.departedAt === null ? "" : t("Left {0}", stopTime(stop.departedAt, stop));
  }
  const parts = [t("Arrived {0}", stopTime(stop.arrivedAt, stop))];
  if (stop.departedAt !== null) parts.push(t("left {0}", stopTime(stop.departedAt, stop)));
  if (stop.dwellMinutes !== null) {
    parts.push(t("{0} on site", durationLabel(stop.dwellMinutes, t)));
  }
  return parts.join(" · ");
}

export type MoveCoverage =
  | {
      kind: "driver";
      moveId: string;
      moveIndex: number;
      drivers: string[];
      tractor: string;
      trailer: string;
      status: AssignmentStatus;
    }
  | {
      kind: "carrier";
      moveId: string;
      moveIndex: number;
      carrier: string;
      status: CarrierAssignmentStatus;
      proNumber: string;
      cost: number | null;
      currency: string;
      driver: string;
    }
  | { kind: "none"; moveId: string; moveIndex: number };

type WorkerSummary = {
  wholeName?: string | null;
  firstName?: string | null;
  lastName?: string | null;
};

function workerName(worker: WorkerSummary | null | undefined): string {
  if (!worker) return "";
  const whole = (worker.wholeName ?? "").trim();
  if (whole) return whole;
  return [worker.firstName, worker.lastName]
    .map((part) => (part ?? "").trim())
    .filter((part) => part !== "")
    .join(" ");
}

/** A driver assignment that puts someone or something on the move, and is not called off. */
function hasDriverAssignment(move: ShipmentMove): boolean {
  const assignment = move.assignment;
  if (!assignment || assignment.status === "Canceled") return false;
  return Boolean(
    assignment.primaryWorkerId ||
    assignment.primaryWorker ||
    assignment.tractorId ||
    assignment.tractor,
  );
}

/** One per move, in move order; a canceled carrier or driver assignment counts as none. */
export function coverageOf(moves: readonly ShipmentMove[]): MoveCoverage[] {
  return inMoveOrder(moves).map((move, moveIndex): MoveCoverage => {
    const moveId = move.id ?? "";
    const carrierAssignment = move.carrierAssignment;
    const byCarrier = isActiveCarrierAssignment(carrierAssignment);
    const byDriver = hasDriverAssignment(move);

    if (byCarrier && carrierAssignment && !(byDriver && move.coverageType === "driver")) {
      return {
        kind: "carrier",
        moveId,
        moveIndex,
        carrier: carrierAssignment.carrier?.name || carrierAssignment.carrier?.code || "",
        status: carrierAssignment.status,
        proNumber: carrierAssignment.proNumber ?? "",
        cost: reportNumericValue(carrierAssignment.totalCost),
        currency: carrierAssignment.currencyCode || "USD",
        driver: (carrierAssignment.externalDriverName ?? "").trim(),
      };
    }

    if (byDriver && move.assignment) {
      const assignment = move.assignment;
      return {
        kind: "driver",
        moveId,
        moveIndex,
        drivers: [
          workerName(assignment.primaryWorker),
          workerName(assignment.secondaryWorker),
        ].filter((name) => name !== ""),
        tractor: assignment.tractor?.code ?? "",
        trailer: assignment.trailer?.code ?? "",
        status: assignment.status,
      };
    }

    return { kind: "none", moveId, moveIndex };
  });
}

/** "Ana Ruiz" → "AR", "" → "?" */
export function initialsOf(name: string): string {
  return getNameInitials(name.trim(), "?");
}

export type FreightLine = {
  id: string;
  name: string;
  pieces: number | null;
  weight: number | null;
  hazmat: boolean;
  fragile: boolean;
  stackable: boolean | null;
  freightClass: string;
};

export type FreightSummary = {
  pieces: number | null;
  weight: number | null;
  temperature: { min: number | null; max: number | null } | null;
  hazmat: boolean;
  lines: FreightLine[];
};

/** The shipment's own figure when it has one above zero, else what its lines add up to. */
function ownOrSum(own: number | null, lines: readonly (number | null)[]): number | null {
  if (own !== null && own > 0) return own;
  const counted = lines.filter((value): value is number => value !== null);
  if (counted.length === 0) return own;
  const sum = counted.reduce((total, value) => total + value, 0);
  return sum > 0 ? sum : own;
}

/**
 * Shipment pieces/weight, or the commodities' sums when the shipment has none; temperature from
 * the shipment's range, else the commodities' tightest; hazmat when any commodity has one.
 */
export function freightOf(
  shipment: Pick<
    Shipment,
    "pieces" | "weight" | "temperatureMin" | "temperatureMax" | "commodities"
  >,
): FreightSummary {
  const lines = (shipment.commodities ?? []).map((line, index): FreightLine => {
    const commodity = line.commodity ?? null;
    return {
      id: line.id || `commodity-${index}`,
      name: commodity?.name ?? "",
      pieces: reportNumericValue(line.pieces),
      weight: reportNumericValue(line.weight),
      hazmat: Boolean(commodity?.hazardousMaterialId),
      fragile: commodity?.fragile === true,
      stackable: typeof commodity?.stackable === "boolean" ? commodity.stackable : null,
      freightClass: commodity?.freightClass ?? "",
    };
  });

  let temperature: FreightSummary["temperature"] = null;
  const ownMin = reportNumericValue(shipment.temperatureMin);
  const ownMax = reportNumericValue(shipment.temperatureMax);
  if (ownMin !== null || ownMax !== null) {
    temperature = { min: ownMin, max: ownMax };
  } else {
    let min: number | null = null;
    let max: number | null = null;
    for (const line of shipment.commodities ?? []) {
      const lineMin = reportNumericValue(line.commodity?.minTemperature);
      const lineMax = reportNumericValue(line.commodity?.maxTemperature);
      if (lineMin !== null) min = min === null ? lineMin : Math.max(min, lineMin);
      if (lineMax !== null) max = max === null ? lineMax : Math.min(max, lineMax);
    }
    if (min !== null || max !== null) temperature = { min, max };
  }

  return {
    pieces: ownOrSum(
      reportNumericValue(shipment.pieces),
      lines.map((line) => line.pieces),
    ),
    weight: ownOrSum(
      reportNumericValue(shipment.weight),
      lines.map((line) => line.weight),
    ),
    temperature,
    hazmat: lines.some((line) => line.hazmat),
    lines,
  };
}

/** "34–38°F", "≥ 34°F", "≤ 38°F"; one figure when both ends meet; "" with neither. */
export function temperatureLabel(
  range: { min: number | null; max: number | null },
  t: TranslateFn,
): string {
  const { min, max } = range;
  if (min !== null && max !== null) {
    return min === max ? t("{0}°F", min) : t("{0}–{1}°F", min, max);
  }
  if (min !== null) return t("≥ {0}°F", min);
  if (max !== null) return t("≤ {0}°F", max);
  return "";
}

export type ChargeLine = { key: string; label: string; detail: string; amount: number | null };

export type Charges = {
  lines: ChargeLine[];
  total: number | null;
  ratedBy: string;
  overridden: boolean;
  overrideReason: string;
  locked: boolean;
};

/** What priced the linehaul: the contract and its rule, else the formula. */
function ratingMethod(shipment: Shipment): string {
  const detail = shipment.ratingDetail;
  const contract = [detail?.agreementName, detail?.ruleLabel]
    .map((part) => (part ?? "").trim())
    .filter((part) => part !== "")
    .join(" · ");
  return contract || formulaName(shipment);
}

function formulaName(shipment: Shipment): string {
  const name = shipment.ratingDetail?.formulaTemplateName || shipment.formulaTemplate?.name;
  return (name ?? "").trim();
}

/**
 * Linehaul first (label "Linehaul", detail the rating method: ratingDetail.agreementName ·
 * ruleLabel, else formulaTemplateName / formulaTemplate.name), then each additional charge
 * (label accessorialCharge.description || code, detail "2 × $45.00" when unit > 1 or
 * "12% of linehaul" for Percentage, amount additionalChargeLineTotal), total from
 * totalChargeAmount. `ratedBy` names the contract or formula alone, and says when it was
 * applied automatically.
 */
export function chargesOf(shipment: Shipment, t: TranslateFn): Charges {
  const lines: ChargeLine[] = [
    {
      key: "linehaul",
      label: t("Linehaul"),
      detail: ratingMethod(shipment),
      amount: reportNumericValue(shipment.freightChargeAmount),
    },
  ];

  (shipment.additionalCharges ?? []).forEach((charge, index) => {
    const accessorial = charge.accessorialCharge;
    const label = (accessorial?.description || accessorial?.code || "").trim() || t("Charge");
    const unit = typeof charge.unit === "number" && Number.isFinite(charge.unit) ? charge.unit : 1;
    let detail = "";
    if (charge.method === "Percentage") {
      const percent = reportNumericValue(charge.amount) ?? 0;
      detail = t("{0} of linehaul", formatPercent(percent, Number.isInteger(percent) ? 0 : 2));
    } else if (unit > 1) {
      detail = t("{0} × {1}", unit, money(charge.amount));
    }
    lines.push({
      key: charge.id || `charge-${index}`,
      label,
      detail,
      amount: additionalChargeLineTotal({
        method: charge.method,
        amount: reportNumericValue(charge.amount),
        unit,
      }),
    });
  });

  const contract = (shipment.ratingDetail?.agreementName ?? "").trim();
  const method = contract || formulaName(shipment);
  let ratedBy = "";
  if (method) {
    ratedBy = shipment.autoRated ? t("Auto-rated by {0}", method) : t("Rated by {0}", method);
  }

  return {
    lines,
    total: reportNumericValue(shipment.totalChargeAmount),
    ratedBy,
    overridden: reportNumericValue(shipment.rateOverrideAmount) !== null,
    overrideReason: (shipment.rateOverrideReason ?? "").trim(),
    locked: shipment.rateLocked === true,
  };
}

export type Margin = {
  profit: number;
  percent: number | null;
  target: number | null;
  cost: number;
  loadedMiles: number;
  ratePerMile: number | null;
  missingDistance: boolean;
};

/** From profitabilityEstimate; null without one. ratePerMile = total / loadedMiles when > 0. */
export function marginOf(
  shipment: Pick<Shipment, "profitabilityEstimate" | "totalChargeAmount">,
): Margin | null {
  const estimate = shipment.profitabilityEstimate;
  if (!estimate) return null;
  const loadedMiles = reportNumericValue(estimate.loadedMiles) ?? 0;
  const total = reportNumericValue(shipment.totalChargeAmount);
  return {
    profit: reportNumericValue(estimate.profit) ?? 0,
    percent: reportNumericValue(estimate.marginPercent),
    target: reportNumericValue(estimate.targetMarginPercent),
    cost: reportNumericValue(estimate.estimatedCost) ?? 0,
    loadedMiles,
    ratePerMile: loadedMiles > 0 && total !== null ? total / loadedMiles : null,
    missingDistance: estimate.missingDistance === true,
  };
}

/** Sum of the distances of the moves still running, or null when none of them has one. */
export function totalMiles(moves: readonly ShipmentMove[]): number | null {
  let total: number | null = null;
  for (const move of moves) {
    if (move.status === "Canceled") continue;
    const distance = positive(move.distance);
    if (distance !== null) total = (total ?? 0) + distance;
  }
  return total;
}

/**
 * "readiness" when delivered and not yet invoiced (status Completed | ReadyToInvoice, or stage
 * Delivered without billedAt), "invoiced" for Invoiced or billedAt, else "notYet".
 */
export function billingPhase(
  shipment: Pick<Shipment, "status" | "stage" | "billedAt">,
): "notYet" | "readiness" | "invoiced" {
  if (shipment.status === "Invoiced" || instant(shipment.billedAt) !== null) return "invoiced";
  if (shipment.status === "Completed" || shipment.status === "ReadyToInvoice") return "readiness";
  if (shipment.stage === "Delivered") return "readiness";
  return "notYet";
}

export type ReadinessCheck = { key: string; state: CardCheckState; title: string; detail: string };

const STANDING_ORDER = ["rejected", "expired", "pending_review", "inactive"] as const;

function standingCount(
  standing: (typeof STANDING_ORDER)[number],
  count: number,
  t: TranslateFn,
): string {
  switch (standing) {
    case "rejected":
      return t("{0} rejected", count);
    case "expired":
      return t("{0} expired", count);
    case "pending_review":
      return t("{0} pending review", count);
    default:
      return t("{0} inactive", count);
  }
}

/** Why the documents on file do not count: "2 rejected, 1 pending review". */
function standingSummary(
  documents: readonly ShipmentBillingIneligibleDocument[],
  t: TranslateFn,
): string {
  return STANDING_ORDER.map((standing) => {
    const count = documents.filter((document) => document.standing === standing).length;
    return count > 0 ? standingCount(standing, count, t) : "";
  })
    .filter((part) => part !== "")
    .join(", ");
}

function requirementDetail(
  requirement: ShipmentBillingRequirement,
  satisfied: boolean,
  t: TranslateFn,
): string {
  const standing = standingSummary(requirement.ineligibleDocuments ?? [], t);
  if (!satisfied) return standing || t("Not on file");
  const onFile = t(
    "{0, plural, one {# document on file} other {# documents on file}}",
    requirement.documentCount ?? 0,
  );
  return [onFile, standing].filter((part) => part !== "").join(" · ");
}

/**
 * One per requirement (ok satisfied / bad missing, detail names ineligible documents' standing),
 * one per validation failure (bad), one for unresolved service failures (warn), one per warning
 * (warn). A requirement listed only as missing still gets its check.
 */
export function readinessChecks(
  readiness: ShipmentBillingReadiness,
  t: TranslateFn,
): ReadinessCheck[] {
  const checks: ReadinessCheck[] = [];
  const missing = readiness.missingRequirements ?? [];
  const missingIds = new Set(missing.map((requirement) => requirement.documentTypeId));
  const requirements = [...(readiness.requirements ?? [])];
  const listed = new Set(requirements.map((requirement) => requirement.documentTypeId));
  for (const requirement of missing) {
    if (!listed.has(requirement.documentTypeId)) {
      listed.add(requirement.documentTypeId);
      requirements.push(requirement);
    }
  }

  for (const requirement of requirements) {
    const satisfied = requirement.satisfied && !missingIds.has(requirement.documentTypeId);
    checks.push({
      key: `requirement:${requirement.documentTypeId}`,
      state: satisfied ? "ok" : "bad",
      title: requirement.documentTypeName || requirement.documentTypeCode,
      detail: requirementDetail(requirement, satisfied, t),
    });
  }

  (readiness.validationFailures ?? []).forEach((failure, index) => {
    checks.push({
      key: `validation:${failure.field}:${failure.code}:${index}`,
      state: "bad",
      title: failure.message,
      detail: "",
    });
  });

  const failures = readiness.serviceFailureContext;
  if (failures?.hasUnresolved) {
    const count = failures.unresolvedCount || (failures.serviceFailureIds ?? []).length;
    checks.push({
      key: "service-failures",
      state: "warn",
      title: t(
        "{0, plural, one {# unresolved service failure} other {# unresolved service failures}}",
        count,
      ),
      detail: t("Review before invoicing"),
    });
  }

  (readiness.warnings ?? []).forEach((warning, index) => {
    checks.push({
      key: `warning:${warning.code}:${index}`,
      state: "warn",
      title: warning.message,
      detail: "",
    });
  });

  return checks;
}

export type NowReading = { tone: CardTone; title: string; detail: string };

function canceledReading(shipment: Shipment, t: TranslateFn): NowReading {
  return {
    tone: STAGE_TONE.Canceled,
    title: t("Canceled"),
    detail: [(shipment.cancelReason ?? "").trim(), cardDay(instant(shipment.canceledAt))]
      .filter((part) => part !== "")
      .join(" · "),
  };
}

function deliveredReading(
  shipment: Shipment,
  stops: readonly JourneyStop[],
  t: TranslateFn,
): NowReading {
  const last = lastDelivery(stops);
  const deliveredAt =
    instant(shipment.actualDeliveryDate) ?? last?.departedAt ?? last?.arrivedAt ?? null;
  const punctuality = last?.punctuality ?? null;
  let detail = "";
  if (punctuality?.state === "late") {
    detail = t("{0} late at the last stop", durationLabel(punctuality.minutes, t));
  } else if (punctuality?.state === "early") {
    detail = t("{0} early at the last stop", durationLabel(punctuality.minutes, t));
  } else if (punctuality?.state === "onTime") {
    detail = t("On time at the last stop");
  }
  return {
    tone: STAGE_TONE.Delivered,
    title: deliveredAt === null ? t("Delivered") : t("Delivered {0}", cardMoment(deliveredAt)),
    detail,
  };
}

function coverageReading(shipment: Shipment, t: TranslateFn): NowReading {
  const running = (shipment.moves ?? []).filter((move) => move.status !== "Canceled");
  const uncovered = coverageOf(running).filter((coverage) => coverage.kind === "none").length;
  let detail = "";
  if (running.length === 0) {
    detail = t("No moves planned yet");
  } else if (uncovered > 0) {
    detail = t(
      "{0} of {1, plural, one {# move} other {# moves}} {0, plural, one {has} other {have}} no driver or carrier",
      uncovered,
      running.length,
    );
  }
  return { tone: STAGE_TONE.NeedsCoverage, title: t("Needs coverage"), detail };
}

/**
 * The live line under the head:
 * Canceled → "Canceled" + reason/day; Delivered → "Delivered {day, time}" + on time/late at the
 * last stop; NeedsCoverage → "Needs coverage" + "{n} of {m} moves have no driver or carrier";
 * otherwise → "At {place}" when a stop is current, else "Next: {stop type} at {place}" + window,
 * with the ETA reading for the final stop when there is one. A next stop whose window has shut by
 * `now` says how late it is running, and a late stop or a worrying ETA turns the line to warn.
 */
export function nowReading(
  shipment: Shipment,
  stops: readonly JourneyStop[],
  t: TranslateFn,
  now: number = nowSeconds(),
): NowReading {
  const stage = shipmentStage(shipment);
  if (stage === "Canceled") return canceledReading(shipment, t);
  if (stage === "Delivered") return deliveredReading(shipment, stops, t);
  if (stage === "NeedsCoverage") return coverageReading(shipment, t);

  const current = stops.find((stop) => stop.phase === "current") ?? null;
  const next = current ? null : (stops.find((stop) => stop.phase === "next") ?? null);
  const stop = current ?? next;

  let title = stageLabel(stage, t);
  let punctuality: StopPunctuality = null;
  if (current) {
    title = current.place.label ? t("At {0}", current.place.label) : t("At a stop");
    punctuality = current.punctuality;
  } else if (next) {
    const type = stopTypeLabel(next.type, t);
    title = next.place.label ? t("Next: {0} at {1}", type, next.place.label) : t("Next: {0}", type);
    punctuality = missedWindow({ start: next.windowStart, end: next.windowEnd }, now);
  }

  const reading = etaReading(shipment.eta, t);
  const eta =
    reading && reading.state !== "unknown"
      ? [reading.label, reading.detail].filter((part) => part !== "").join(", ")
      : "";

  const detail = [stop ? windowLabel(stop, t) : "", punctualityLabel(punctuality, t), eta]
    .filter((part) => part !== "")
    .join(" · ");

  const worrying =
    punctuality?.state === "late" || reading?.state === "bad" || reading?.state === "warn";
  return { tone: worrying ? "warn" : STAGE_TONE[stage], title, detail };
}

export function tenderStatusLabel(status: ShipmentTenderStatus, t: TranslateFn): string {
  switch (status) {
    case "Tendered":
      return t("Tendered");
    case "Accepted":
      return t("Tender accepted");
    case "Rejected":
      return t("Tender rejected");
    case "Expired":
      return t("Tender expired");
    case "Canceled":
      return t("Tender canceled");
  }
}

export function assignmentStatusLabel(status: AssignmentStatus, t: TranslateFn): string {
  switch (status) {
    case "New":
      return t("Assigned");
    case "InProgress":
      return t("Under way");
    case "Completed":
      return t("Completed");
    case "Canceled":
      return t("Canceled");
  }
}

export function carrierStatusLabel(status: CarrierAssignmentStatus, t: TranslateFn): string {
  switch (status) {
    case "Pending":
      return t("Pending");
    case "Confirmed":
      return t("Confirmed");
    case "Canceled":
      return t("Canceled");
  }
}

/** Who covers a move in a few words, for the line that opens the move on the rail. */
export function coverageSummary(coverage: MoveCoverage, t: TranslateFn): string {
  switch (coverage.kind) {
    case "driver":
      return coverage.drivers.length > 0 ? formatList(coverage.drivers) : t("Driver assigned");
    case "carrier":
      return coverage.carrier || t("Carrier assigned");
    case "none":
      return t("No driver or carrier yet");
  }
}

/**
 * Where each stop sits along the strip, from 0 to 1. When every move has a
 * distance the moves take the share of the line their miles do, and a move's
 * stops are spread across its share; otherwise the stops are spaced evenly.
 */
export function stopOffsets(
  moves: readonly ShipmentMove[],
  stops: readonly JourneyStop[],
): number[] {
  if (stops.length < 2) return stops.map(() => 0);
  const distances = [...moves]
    .sort((a, b) => a.sequence - b.sequence)
    .map((move) => move.distance ?? 0);
  const total = distances.reduce((sum, distance) => sum + distance, 0);
  if (distances.length === 0 || total <= 0 || distances.some((distance) => distance <= 0)) {
    return stops.map((_, index) => index / (stops.length - 1));
  }

  const starts: number[] = [];
  let run = 0;
  for (const distance of distances) {
    starts.push(run);
    run += distance;
  }
  const counts = new Map<number, number>();
  const firsts = new Map<number, number>();
  stops.forEach((stop, index) => {
    counts.set(stop.moveIndex, (counts.get(stop.moveIndex) ?? 0) + 1);
    if (!firsts.has(stop.moveIndex)) firsts.set(stop.moveIndex, index);
  });

  return stops.map((stop, index) => {
    const start = (starts[stop.moveIndex] ?? 0) / total;
    const span = (distances[stop.moveIndex] ?? 0) / total;
    const count = counts.get(stop.moveIndex) ?? 1;
    const within = index - (firsts.get(stop.moveIndex) ?? index);
    return clampUnit(count < 2 ? start : start + (span * within) / (count - 1));
  });
}

/**
 * Where the truck stands on the strip: at the stop it is at, or at the
 * journey's progress held between the last stop it left and the next one,
 * so the marker never sits past a stop it has not reached.
 */
export function truckOffset(
  stops: readonly JourneyStop[],
  offsets: readonly number[],
  progress: number,
): number {
  const current = stops.findIndex((stop) => stop.phase === "current");
  if (current !== -1) return offsets[current] ?? 0;
  let lower = 0;
  let upper = 1;
  stops.forEach((stop, index) => {
    if (stop.phase === "done") lower = Math.max(lower, offsets[index] ?? 0);
  });
  const next = stops.findIndex((stop) => stop.phase === "next");
  if (next !== -1) upper = offsets[next] ?? 1;
  return Math.min(Math.max(clampUnit(progress), lower), Math.max(lower, upper));
}

/** How a margin reads against its target: under water, under target, on it, or unknown. */
export type MarginTone = "ok" | "warn" | "bad" | "unknown";

/**
 * The margin meter: how far the bar fills and where the target's tick
 * stands, each from 0 to 1 of a scale with room for twice the target and at
 * least 40 points, and the tone the margin reads in.
 */
export function marginGauge(margin: Margin): {
  fill: number;
  tick: number | null;
  tone: MarginTone;
} {
  const scale = Math.min(100, Math.max(40, (margin.target ?? 0) * 2, margin.percent ?? 0));
  return {
    fill: margin.percent === null ? 0 : clampUnit(margin.percent / scale),
    tick: margin.target === null ? null : clampUnit(margin.target / scale),
    tone:
      margin.percent === null
        ? "unknown"
        : margin.percent < 0
          ? "bad"
          : margin.target !== null && margin.percent < margin.target
            ? "warn"
            : "ok",
  };
}

/** A freight class as the NMFC writes it: "Class77_5" reads "Class 77.5". */
export function freightClassLabel(freightClass: string, t: TranslateFn): string {
  const value = freightClass.replace(/^Class/u, "").replaceAll("_", ".");
  return value === "" ? "" : t("Class {0}", value);
}
