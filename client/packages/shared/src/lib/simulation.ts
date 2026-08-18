import type {
  RateSimulation,
  RateSimulationSummary,
  RuleCoverage,
  RuleOutcome,
} from "../types/rate";

/**
 * Reading a simulation.
 *
 * A run answers two questions and they are read differently. The revenue delta
 * is a number somebody weighs; rule coverage is a list somebody acts on. What
 * lives here is the translation of each into the words and shapes the panel
 * shows — nothing here recomputes what the server measured.
 */

/** Whether the run has finished, whatever the outcome. */
export function isTerminal(simulation: RateSimulation | undefined): boolean {
  const status = simulation?.status;

  return status === "Completed" || status === "Failed" || status === "Canceled";
}

/**
 * Whether a status screen should keep asking.
 *
 * A run takes minutes, so the panel polls. Polling a finished run forever is
 * how a background job quietly becomes a load on the database.
 */
export function shouldPoll(simulation: RateSimulation | undefined): boolean {
  if (!simulation) return false;

  return !isTerminal(simulation);
}

/**
 * Whether a simulation actually measured anything.
 *
 * A completed run over zero shipments reports a zero delta, which reads as
 * "this change costs nothing" rather than "this measured nothing". They are
 * opposite conclusions and the panel has to tell them apart.
 */
export function measuredAnything(simulation: RateSimulation | undefined): boolean {
  return (simulation?.summary?.evaluatedCount ?? 0) > 0;
}

export type DeltaDirection = "up" | "down" | "flat";

export function deltaDirection(summary: RateSimulationSummary | null | undefined): DeltaDirection {
  const delta = summary?.totalDelta ?? 0;

  if (delta > 0) return "up";
  if (delta < 0) return "down";

  return "flat";
}

/**
 * How far through a run is, as a fraction.
 *
 * A run whose total is not known yet reports nothing rather than zero: an empty
 * bar reads as "no progress", which is wrong for a job that has not been told
 * how much there is to do.
 */
export function runProgress(simulation: RateSimulation | undefined): number | null {
  const summary = simulation?.summary;
  if (!summary || summary.shipmentCount <= 0) return null;

  const done = summary.evaluatedCount + summary.errorCount;

  return Math.min(1, done / summary.shipmentCount);
}

const OUTCOME_LABEL: Record<RuleOutcome, string> = {
  Won: "Priced shipments",
  Lost: "Always outranked",
  NeverFired: "Never matched",
};

export function ruleOutcomeLabel(outcome: RuleOutcome): string {
  return OUTCOME_LABEL[outcome] ?? outcome;
}

/**
 * What to tell somebody about one rule's run.
 *
 * The two failure states have different fixes, and saying "this rule did
 * nothing" for both would send somebody looking in the wrong place. A lane that
 * never matched is written for freight that does not exist; a lane that always
 * lost is shadowed by something narrower.
 */
export function ruleCoverageNote(row: RuleCoverage): string {
  switch (row.outcome) {
    case "NeverFired":
      return "No shipment in this window matched this lane. It may be written for freight you do not move.";
    case "Lost":
      return row.lostToLabel
        ? `Matched ${row.lostCount} shipments and never won — ${row.lostToLabel} covers the same freight more narrowly.`
        : `Matched ${row.lostCount} shipments and never won: something narrower covers the same freight.`;
    case "Won":
      return row.lostCount > 0
        ? `Priced ${row.wonCount} shipments, outranked on ${row.lostCount}.`
        : `Priced ${row.wonCount} shipments.`;
    default:
      return "";
  }
}

/**
 * The rules worth acting on, which is what the panel leads with.
 *
 * A working rule is there for completeness; the ones that did nothing are why
 * somebody opened the list.
 */
export function problemRules(coverage: readonly RuleCoverage[] | null | undefined): RuleCoverage[] {
  return (coverage ?? []).filter((row) => row.outcome !== "Won");
}

/**
 * A one-line reading of the whole run, in the terms somebody decides on.
 *
 * A run that priced nothing says so rather than reporting a delta of zero,
 * because the two look identical as numbers and mean opposite things.
 */
export function summaryHeadline(simulation: RateSimulation | undefined): string {
  if (!simulation) return "";

  if (simulation.status === "Failed") {
    return simulation.error || "This simulation did not finish.";
  }

  if (!isTerminal(simulation)) {
    return "Replaying shipments…";
  }

  const summary = simulation.summary;
  if (!summary || summary.evaluatedCount === 0) {
    return "No shipments in this window could be priced, so there is nothing to compare.";
  }

  const direction = deltaDirection(summary);
  if (direction === "flat") {
    return `Across ${summary.evaluatedCount} shipments this contract charges exactly what they were billed.`;
  }

  const verb = direction === "up" ? "more" : "less";
  const magnitude = Math.abs(summary.totalDeltaPct);

  return `Across ${summary.evaluatedCount} shipments this contract charges ${magnitude}% ${verb} than they were billed.`;
}
