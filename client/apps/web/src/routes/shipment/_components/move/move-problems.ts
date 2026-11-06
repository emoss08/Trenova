import { defineLabels } from "@trenova/shared/i18n/labels";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";

/** Something wrong with a move or one of its stops that keeps the shipment from saving. */
export type MoveProblem = {
  /** The stop it is on, by its place in the move, or null when it is on the move itself. */
  stopIndex: number | null;
  /** The field of the stop (or of the move) it is on, such as `locationId`; empty for the whole. */
  field: string;
  /** The form path of the field, for focusing it in the move editor. */
  path: string;
  message: string;
};

const STOP_FIELD_LABELS: Readonly<Record<string, string>> = defineLabels({
  locationId: "Location",
  location: "Location",
  addressLine: "Address",
  type: "Stop type",
  status: "Status",
  sequence: "Sequence",
  scheduleType: "Schedule type",
  handling: "Live or drop",
  pieces: "Pieces",
  weight: "Weight (lbs)",
  scheduledWindowStart: "Scheduled start",
  scheduledWindowEnd: "Scheduled end",
  actualArrival: "Actual arrival",
  actualDeparture: "Actual departure",
});

const MOVE_FIELD_LABELS: Readonly<Record<string, string>> = defineLabels({
  status: "Status",
  loaded: "Loaded",
  distance: "Distance (mi)",
  stops: "Stops",
});

// Keys of a react-hook-form error node that describe it rather than name a child field.
const ERROR_NODE_KEYS = new Set(["message", "type", "types", "ref"]);

function isObject(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object";
}

function collect(node: unknown, path: string, out: { path: string; message: string }[]) {
  if (!isObject(node)) return;
  const message = node.message;
  if (typeof message === "string" && message.trim() !== "") {
    out.push({ path, message });
  }
  for (const [key, child] of Object.entries(node)) {
    if (ERROR_NODE_KEYS.has(key)) continue;
    // A field array keeps its own error under `root`; it belongs to the array itself.
    collect(child, key === "root" ? path : `${path}.${key}`, out);
  }
}

/**
 * Every problem the form holds for one move, each placed on the stop and field it belongs to,
 * in the order the form holds them. `errors` is the move's own branch of the form's errors
 * (`errors.moves[moveIndex]`): field errors from the schema and from the server alike, and
 * errors on the move or its list of stops as a whole.
 */
export function moveProblems(moveIndex: number, errors: unknown): MoveProblem[] {
  const base = `moves.${moveIndex}`;
  const found: { path: string; message: string }[] = [];
  collect(errors, base, found);

  const seen = new Set<string>();
  const problems: MoveProblem[] = [];
  for (const { path, message } of found) {
    const key = `${path}\n${message}`;
    if (seen.has(key)) continue;
    seen.add(key);

    const segments = path === base ? [] : path.slice(base.length + 1).split(".");
    const onStop = segments[0] === "stops" && /^\d+$/.test(segments[1] ?? "");
    problems.push(
      onStop
        ? { stopIndex: Number(segments[1]), field: segments[2] ?? "", path, message }
        : { stopIndex: null, field: segments[0] ?? "", path, message },
    );
  }
  return problems;
}

/** The label of the field a problem is on, or undefined for one the editor has no name for. */
export function problemFieldLabel(problem: MoveProblem): string | undefined {
  const labels = problem.stopIndex === null ? MOVE_FIELD_LABELS : STOP_FIELD_LABELS;
  return Object.hasOwn(labels, problem.field) ? labels[problem.field] : undefined;
}

/** A problem as a person reads it: the field it is on, then what is wrong with it. */
export function problemText(problem: MoveProblem, t: TranslateFn): string {
  const label = problemFieldLabel(problem);
  return label ? t("{0}: {1}", label, problem.message) : problem.message;
}

/** A problem with the stop it is on, for a list that covers the whole move. */
export function problemTextInMove(problem: MoveProblem, t: TranslateFn): string {
  const text = problemText(problem, t);
  return problem.stopIndex === null ? text : t("Stop {0} · {1}", problem.stopIndex + 1, text);
}
