import { assistant } from "@/lib/queries/assistant";
import type { AssistantArtifact, AssistantProposal } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { humanizeToolName } from "@/components/assistant/proposal-state";
import { useMemo } from "react";
import { planFrom } from "../../artifact-payloads";
import { ArtIcon } from "../../desk-art-kinds";

function stepState(proposal: AssistantProposal | undefined): "done" | "wait" | "next" | "fail" {
  switch (proposal?.status) {
    case "Executed":
    case "Simulated":
      return "done";
    case "Pending":
      return "wait";
    case "Rejected":
    case "ExecutionFailed":
    case "Expired":
      return "fail";
    default:
      return "next";
  }
}

/** A plan as a checklist with its progress, each step ticking as it runs. It is decided in the approval box. */
export function DeskPlanBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const plan = useMemo(() => planFrom(artifact), [artifact]);
  const proposalsQuery = useQuery(assistant.proposals(artifact.threadId));
  const byId = new Map(
    (proposalsQuery.data?.results ?? []).map((proposal) => [proposal.id, proposal]),
  );
  const steps = plan.steps.map((step) => ({
    ...step,
    state: stepState(byId.get(step.proposalId)),
  }));
  const done = steps.filter((step) => step.state === "done").length;
  const total = Math.max(steps.length, plan.stepCount, 1);

  return (
    <div className="dk-ax-pad dk-ax-plan">
      {plan.summary !== "" && <p className="dk-ax-lead">{plan.summary}</p>}
      <div className="dk-ax-prog">
        <span style={{ width: `${(done / total) * 100}%` }} />
      </div>
      <div className="dk-ax-prog-l">{t("{0} of {1} steps done", done, total)}</div>
      <ol>
        {steps.map((step, index) => (
          <li
            key={step.proposalId || step.step}
            className={`dk-st-${step.state}`}
            style={{ animationDelay: `${index * 70}ms` }}
          >
            <span className="dk-ax-sn">
              {step.state === "done" ? <ArtIcon name="check" size={11} stroke={2.8} /> : index + 1}
            </span>
            <div>
              <b>{step.rationale || humanizeToolName(step.toolName)}</b>
              <span>
                <code>{step.toolName}</code>
                {step.state === "wait"
                  ? ` · ${t("waiting on your approval")}`
                  : step.state === "next"
                    ? ` · ${t("up next")}`
                    : step.state === "fail"
                      ? ` · ${t("did not run")}`
                      : ""}
              </span>
            </div>
          </li>
        ))}
      </ol>
      {steps.length === 0 && (
        <p className="dk-ax-note">{t("The plan behind this is no longer in the conversation.")}</p>
      )}
    </div>
  );
}
