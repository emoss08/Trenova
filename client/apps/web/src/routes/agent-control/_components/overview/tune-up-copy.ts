import type { AITuneUp } from "@/lib/graphql/ai-control";
import {
  Award01Icon,
  ClockIcon,
  Dataflow03Icon,
  EyeOffIcon,
  SwitchVertical01Icon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { tierLabel } from "../safety/safety-model";

const SECONDS_PER_DAY = 86_400;

export type TuneUpTone = "ok" | "warn" | "brand" | "neutral";

/** A run of the title; a strong part is the thing the change is about. */
export type TuneUpTitlePart = { text: string; strong?: boolean };

export type TuneUpCopy = {
  icon: IconComponent;
  title: TuneUpTitlePart[];
  evidence: string;
  gain: string;
  tone: TuneUpTone;
  action: string;
  /** What the toast says once it is applied. */
  applied: string;
};

export type TuneUpLabels = {
  tool: (name: string) => string;
  task: (task: string) => string;
  day: (unix: number) => string;
  now: number;
};

/** Splits a translated sentence on its one {0} placeholder so the name it holds can be bold. */
function titled(template: string, strong: string): TuneUpTitlePart[] {
  const [before, after = ""] = template.split("{0}");
  return [
    ...(before ? [{ text: before }] : []),
    { text: strong, strong: true },
    ...(after ? [{ text: after }] : []),
  ];
}

function percent(rate: number): string {
  return `${Math.round(rate * 100)}%`;
}

/** Every tune-up's sentence, its evidence, what it gains and what applying it is called. */
export function tuneUpCopy(tuneUp: AITuneUp, t: TranslateFn, labels: TuneUpLabels): TuneUpCopy {
  const { evidence } = tuneUp;
  const agentName = tuneUp.agent?.name ?? "";
  const providerName = tuneUp.provider?.name ?? "";
  const otherName = tuneUp.otherProvider?.name ?? "";

  switch (tuneUp.kind) {
    case "RaiseToolTier": {
      const tool = labels.tool(tuneUp.toolName ?? "");
      const automatic = evidence.toTier === "AutoExecute";
      const perWeek = Math.round(evidence.approvalsPerWeek);
      return {
        icon: Award01Icon,
        title: automatic
          ? titled(t("Let {0} run on its own for {1}", "{0}", agentName), tool)
          : titled(
              t(
                "Move {0} to {1} for {2}",
                "{0}",
                tierLabel(t, evidence.toTier ?? "ActWithApproval"),
                agentName,
              ),
              tool,
            ),
        evidence:
          evidence.rejections === 0
            ? t("Approved unchanged {0} times in a row · never rejected", evidence.streak)
            : evidence.rejections === 1
              ? t("Approved unchanged {0} times in a row · 1 rejection before that", evidence.streak)
              : t(
                  "Approved unchanged {0} times in a row · {1} rejections before that",
                  evidence.streak,
                  evidence.rejections,
                ),
        gain:
          perWeek >= 1
            ? perWeek === 1
              ? t("About 1 approval a week")
              : t("About {0} approvals a week", perWeek)
            : t("Fewer approvals to make"),
        tone: "ok",
        action: t("Raise it"),
        applied: t("{0} moved up a tier for {1}", tool, agentName),
      };
    }
    case "ReorderProviders": {
      const tasks = evidence.tasks.length;
      return {
        icon: SwitchVertical01Icon,
        title: titled(t("Put {0} ahead of {1}", "{0}", otherName), providerName),
        evidence: t(
          "{0} failed {1} times in {2} days; {3} answered {4} of them",
          otherName,
          evidence.failed.toLocaleString(),
          30,
          providerName,
          evidence.rescued.toLocaleString(),
        ),
        gain:
          tasks === 1
            ? t("Fewer retries on {0}", labels.task(evidence.tasks[0]))
            : t("Fewer retries on {0} tasks", tasks),
        tone: "warn",
        action: t("Reorder"),
        applied: t("{0} now goes before {1}", providerName, otherName),
      };
    }
    case "LeaveShadow":
      return {
        icon: EyeOffIcon,
        title: titled(t("Take {0} out of shadow", "{0}"), agentName),
        evidence:
          evidence.wouldFail === 0
            ? t(
                "{0} recorded proposals · {1} match what people did · none would have failed",
                evidence.recorded,
                percent(evidence.matchRate),
              )
            : t(
                "{0} recorded proposals · {1} match what people did · {2} would have failed",
                evidence.recorded,
                percent(evidence.matchRate),
                evidence.wouldFail,
              ),
        gain: t("Proposals start reaching Desk"),
        tone: "brand",
        action: t("Go live"),
        applied: t("{0} is live", agentName),
      };
    case "AssignTask": {
      const task = labels.task(tuneUp.task ?? "");
      const embedding = tuneUp.task === "Embedding";
      const gap = embedding
        ? t("Nothing handles it, so search matches words only.")
        : t("Nothing handles it now.");
      return {
        icon: Dataflow03Icon,
        title: titled(t("Give {0} to {1}", "{0}", providerName), task),
        evidence: evidence.model
          ? t("{0} {1} already serves {2}", gap, providerName, evidence.model)
          : t("{0} {1} can take it", gap, providerName),
        gain: embedding ? t("Search by meaning") : t("{0} starts working", task),
        tone: "warn",
        action: t("Assign"),
        applied: t("{0} now goes to {1}", task, providerName),
      };
    }
    case "TurnOffIdleAgent": {
      const days = Math.max(1, Math.floor((labels.now - evidence.idleSince) / SECONDS_PER_DAY));
      const since = labels.day(evidence.idleSince);
      return {
        icon: ClockIcon,
        title: evidence.lastRunAt
          ? titled(t("{0} hasn't run in {1} days", "{0}", days), agentName)
          : titled(t("{0} has never run", "{0}"), agentName),
        evidence:
          evidence.tools === 1
            ? t("Nobody has asked it anything since {0} · it holds 1 tool", since)
            : t("Nobody has asked it anything since {0} · it holds {1} tools", since, evidence.tools),
        gain: t("Fewer agents to choose from in Desk"),
        tone: "neutral",
        action: t("Turn off"),
        applied: t("{0} is off", agentName),
      };
    }
  }
}
