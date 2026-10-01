import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { ProposalPresentation } from "./proposal-state";

export type RecordStatusInput = {
  state: ProposalPresentation;
  /** When it was decided, already formatted for the reader; empty when not known. */
  time: string;
  /** The reader is the person who decided it. */
  decidedByMe: boolean;
  /** It ran without waiting on anyone, at a tier the agent may act on alone. */
  own: boolean;
};

function approved({ time, decidedByMe }: RecordStatusInput, t: TranslateFn): string {
  if (decidedByMe) {
    return time !== "" ? t("Approved by you {0}", time) : t("Approved by you");
  }

  return time !== "" ? t("Approved {0}", time) : t("Approved");
}

function rejected({ time, decidedByMe }: RecordStatusInput, t: TranslateFn): string {
  if (decidedByMe) {
    return time !== "" ? t("Rejected by you {0}", time) : t("Rejected by you");
  }

  return time !== "" ? t("Rejected {0}", time) : t("Rejected");
}

/**
 * The second half of a decision's one-line record: where it stands, who
 * decided it and when. "Approved" and "done" stay two facts, so an approval
 * that is still running or did not go through says so after the approval.
 * A write that ran on its own was never a question and never reads as
 * approved. One still waiting points at the approval box below, the only
 * place it is decided.
 */
export function recordStatus(input: RecordStatusInput, t: TranslateFn): string {
  switch (input.state) {
    case "awaiting":
      return t("Waiting — decide below");
    case "held":
      return t("On hold");
    case "running":
      return input.own ? t("Running on its own") : `${approved(input, t)} · ${t("running")}`;
    case "done":
      return input.own ? t("Done on its own") : approved(input, t);
    case "failed":
      return input.own
        ? t("Ran on its own · did not go through")
        : `${approved(input, t)} · ${t("did not run")}`;
    case "simulated":
      return `${approved(input, t)} · ${t("simulated")}`;
    case "declined":
      return rejected(input, t);
    default:
      return t("Expired without a decision");
  }
}
