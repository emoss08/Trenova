import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Link } from "@tanstack/react-router";
import type { DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import { recordPath } from "@/config/record-links";
import type { DispatchPlanGap, DispatchPlanLine } from "../../artifact-payloads";
import { CardAvatar } from "../../record-card/record-card";
import { cardMoment } from "../../record-card/record-card-values";
import { type LoadState, needsCheck, takenBy } from "./dispatch-plan-model";

/** The load as a ticket: its PRO number, its lane and when it picks up. */
function LoadTicket({
  proNumber,
  lane,
  pickupAt,
}: {
  proNumber: string;
  lane: string;
  pickupAt: number | null;
}) {
  const t = useT();
  return (
    <div className="dk-dpl-load">
      <div className="dk-dpl-pro">{proNumber}</div>
      {lane !== "" && <div className="dk-dpl-lane">{lane}</div>}
      <div className="dk-dpl-when">
        {pickupAt === null ? t("Pickup time not on file") : t("Pickup {0}", cardMoment(pickupAt))}
      </div>
    </div>
  );
}

/** What the connector between a load and its driver says: how the pairing stands now. */
function linkWords(state: LoadState, line: DispatchPlanLine, t: ReturnType<typeof useT>): string {
  switch (state) {
    case "assigned":
      return t("Assigned");
    case "proposed":
      return t("Proposed");
    case "taken":
      return t("Covered");
    case "closed":
      return t("Closed");
    case "unread":
      return t("Planned");
    case "planned":
      return needsCheck(line) ? t("Check") : t("Planned");
  }
}

function toneOf(state: LoadState, line: DispatchPlanLine): string {
  if (state === "assigned" || state === "taken" || state === "closed") {
    return "dk-done";
  }
  return needsCheck(line) ? "dk-warn" : "dk-ok";
}

/** A planned load beside the driver the plan put on it, joined by how the pairing stands now. */
export function PlanPair({
  line,
  state,
  move,
}: {
  line: DispatchPlanLine;
  state: LoadState;
  move: DispatchBoardMove | undefined;
}) {
  const t = useT();
  const details = [
    line.deadheadMiles === null
      ? null
      : t("{0, plural, one {# mile empty} other {# miles empty}}", Math.round(line.deadheadMiles)),
    line.driveHours === null ? t("hours unknown") : t("{0} h to drive", line.driveHours.toFixed(1)),
  ].filter((part): part is string => part !== null);
  const warnings = [
    ...line.findings
      .filter((finding) => finding.severity === "Warn" || finding.severity === "Block")
      .map((finding) => finding.message),
    ...(line.hoursNote === ""
      ? []
      : [t("No ELD has reported this driver's hours; they are unknown, not zero.")]),
    ...(line.notSendable === ""
      ? []
      : [t("The driver has no tractor on file, so this load cannot be assigned from here.")]),
  ];
  const coveredBy = state === "taken" ? takenBy(move) : "";

  return (
    <li className={cn("dk-dpl-pair", toneOf(state, line))}>
      <LoadTicket proNumber={line.proNumber} lane={line.lane} pickupAt={line.pickupAt} />
      <div className="dk-dpl-link">
        <span>{linkWords(state, line, t)}</span>
      </div>
      <div className="dk-dpl-drv">
        <CardAvatar name={line.workerName} size="sm" />
        <div className="dk-dpl-who">
          <Link className="dk-dpl-nm" to={recordPath("worker", line.workerId)}>
            {line.workerName}
          </Link>
          <span className="dk-dpl-sub">{details.join(" · ")}</span>
          {coveredBy !== "" && (
            <span className="dk-dpl-sub">{t("Now covered by {0}", coveredBy)}</span>
          )}
        </div>
        <div className="dk-dpl-cost">
          {line.cost}
          <small>{t("Plan cost")}</small>
        </div>
        {warnings.length > 0 && state === "planned" && (
          <ul className="dk-dpl-note">
            {warnings.map((warning) => (
              <li key={warning}>{warning}</li>
            ))}
          </ul>
        )}
      </div>
      {line.nextBest.length > 0 && state === "planned" && (
        <div className="dk-dpl-alts">
          <span>{t("Next best")}</span>
          {line.nextBest.map((alternative) => (
            <span key={alternative.workerName} className="dk-dpl-alt">
              {alternative.workerName}
              <b>{t("+{0}", alternative.extraCost)}</b>
            </span>
          ))}
        </div>
      )}
    </li>
  );
}

/** A load nobody can take as things stand, beside the reason. */
export function PlanGap({
  gap,
  move,
}: {
  gap: DispatchPlanGap;
  move: DispatchBoardMove | undefined;
}) {
  const t = useT();
  const coveredBy = takenBy(move);
  const lane =
    move && move.originCity && move.destinationCity
      ? t("{0} to {1}", move.originCity, move.destinationCity)
      : "";

  return (
    <li className={cn("dk-dpl-pair", coveredBy === "" ? "dk-bad" : "dk-done")}>
      <LoadTicket
        proNumber={gap.proNumber}
        lane={lane}
        pickupAt={move?.originWindowStart ?? null}
      />
      <div className="dk-dpl-link">
        <span>{coveredBy === "" ? t("None") : t("Covered")}</span>
      </div>
      <div className="dk-dpl-drv dk-dpl-none">
        {coveredBy === "" ? (
          <>
            <b>{t("No driver can take it")}</b>
            <p>{gap.reason}</p>
            {gap.findings
              .filter((finding) => !gap.reason.includes(finding.message))
              .map((finding) => (
                <p key={finding.code + finding.message}>{finding.message}</p>
              ))}
          </>
        ) : (
          <b>{t("Now covered by {0}", coveredBy)}</b>
        )}
      </div>
    </li>
  );
}
