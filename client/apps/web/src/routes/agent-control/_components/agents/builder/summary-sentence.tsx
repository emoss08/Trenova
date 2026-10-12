import type { AgentFormValues } from "../agent-form-schema";
import type { AutonomyTier } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatList } from "@trenova/shared/i18n/format";
import type { KeyboardEvent, ReactNode } from "react";
import type { BlockId } from "./block";

type SummarySentenceProps = {
  values: AgentFormValues;
  reads: number;
  byTier: Record<AutonomyTier, number>;
  /** What the chosen schedule says, when it is one of the presets. */
  schedule: string | null;
  roleNames: readonly string[];
  eventLabel: (kind: string) => string;
  onJump: (block: BlockId) => void;
  onMode: () => void;
};

/** The whole agent in a sentence, each clause opening the part it describes. */
export function SummarySentence({
  values,
  reads,
  byTier,
  schedule,
  roleNames,
  eventLabel,
  onJump,
  onMode,
}: SummarySentenceProps) {
  const t = useT();

  const trigger: ReactNode =
    values.triggerMode === "Chat" ? (
      <>
        {t("Answers")}{" "}
        <b>{values.accessMode === "Roles" && roleNames.length ? formatList(roleNames) : t("anyone")}</b>{" "}
        {t("in Desk")}
      </>
    ) : values.triggerMode === "Scheduled" ? (
      <>
        {t("Runs")} <b>{schedule ? lower(schedule) : t("on a custom schedule")}</b>
      </>
    ) : values.triggerMode === "Event" ? (
      <>
        {t("Wakes when")}{" "}
        <b>
          {values.eventKinds.length
            ? formatList(
                values.eventKinds.slice(0, 2).map((kind) => lower(eventLabel(kind))),
                "disjunction",
              )
            : t("an event you pick")}
        </b>
      </>
    ) : (
      <>
        {t("Keeps watch every")}{" "}
        <b>{t("{0} minutes", Math.max(1, Math.round(values.intervalSeconds / 60)))}</b>
      </>
    );

  const clauses: ReactNode[] = [
    <>
      {t("reads with")} <b>{reads === 1 ? t("1 tool") : t("{0} tools", reads)}</b>
    </>,
  ];
  if (byTier.AutoExecute) {
    clauses.push(
      <>
        {t("does")}{" "}
        <b>
          {byTier.AutoExecute === 1
            ? t("1 thing on its own")
            : t("{0} things on its own", byTier.AutoExecute)}
        </b>
      </>,
    );
  }
  if (byTier.ActWithApproval) {
    clauses.push(
      <>
        {t("asks a person before")} <b>{byTier.ActWithApproval}</b>
      </>,
    );
  }
  if (byTier.Propose) {
    clauses.push(
      <>
        {t("only proposes")} <b>{byTier.Propose}</b>
      </>,
    );
  }

  const mode = values.simulationMode
    ? t("It's in simulation, so nothing is written.")
    : values.shadowMode
      ? t("It runs in shadow, so nothing is offered yet.")
      : t("It's live.");

  return (
    <p className="sent">
      <Clause onActivate={() => onJump("trig")}>{trigger}</Clause>,{" "}
      <Clause onActivate={() => onJump("tools")}>
        {clauses.map((clause, index) => (
          <span key={index}>
            {index === 0 ? "" : index === clauses.length - 1 ? ` ${t("and")} ` : ", "}
            {clause}
          </span>
        ))}
      </Clause>
      .{" "}
      <Clause onActivate={onMode}>{mode}</Clause>
    </p>
  );
}

function lower(text: string): string {
  return text.charAt(0).toLowerCase() + text.slice(1);
}

function Clause({ onActivate, children }: { onActivate: () => void; children: ReactNode }) {
  const onKeyDown = (event: KeyboardEvent<HTMLSpanElement>) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      onActivate();
    }
  };
  return (
    <span className="sj" role="button" tabIndex={0} onClick={onActivate} onKeyDown={onKeyDown}>
      {children}
    </span>
  );
}
