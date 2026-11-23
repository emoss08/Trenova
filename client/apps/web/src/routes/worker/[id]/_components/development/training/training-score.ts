import type { TranslateFn } from "@trenova/shared/i18n/runtime";

export type ScoreProjection =
  | { kind: "unscored" }
  | { kind: "waiting"; passing: number }
  | { kind: "invalid" }
  | { kind: "pass" | "fail"; passing: number; score: number };

/**
 * Where a typed score lands against the course's passing mark, before anything is saved. The
 * server decides pass or fail the same way: at or above the mark passes.
 */
export function projectScore(
  score: string | null | undefined,
  passing: string | number | null | undefined,
): ScoreProjection {
  const mark = passing == null || passing === "" ? null : Number(passing);
  const raw = score?.trim() ?? "";
  if (raw !== "") {
    const value = Number(raw);
    if (!Number.isFinite(value) || value < 0 || value > 100) return { kind: "invalid" };
    if (mark === null || !Number.isFinite(mark)) return { kind: "unscored" };
    return { kind: value >= mark ? "pass" : "fail", passing: mark, score: value };
  }
  if (mark === null || !Number.isFinite(mark)) return { kind: "unscored" };
  return { kind: "waiting", passing: mark };
}

/** The live line under the score field. */
export function scoreLine(projection: ScoreProjection, t: TranslateFn): string {
  const mark = (passing: number) => Number(passing.toFixed(0));
  switch (projection.kind) {
    case "unscored":
      return t("This course has no passing mark; a score is optional.");
    case "waiting":
      return t("Passes at {0}%.", mark(projection.passing));
    case "invalid":
      return t("Enter a score from 0 to 100.");
    case "pass":
      return t("Passes at {0}%. This would pass.", mark(projection.passing));
    case "fail":
      return t("Passes at {0}%. This would fail.", mark(projection.passing));
  }
}
