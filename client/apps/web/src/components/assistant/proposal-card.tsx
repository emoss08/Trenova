import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Progress } from "@trenova/shared/components/ui/progress";
import { generateDateTimeStringFromUnixTimestamp } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useT } from "@trenova/shared/i18n/use-t";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { AssistantProposal, ProposalDecision } from "@/types/assistant";
import { useQueryClient } from "@tanstack/react-query";
import {
  CheckIcon,
  CircleAlertIcon,
  CircleCheckIcon,
  CircleSlashIcon,
  LoaderIcon,
  RotateCcwIcon,
  ShieldQuestionIcon,
  XIcon,
} from "lucide-react";
import { m } from "motion/react";
import { presentProposal } from "./proposal-presenters";
import { classifyProposal } from "./proposal-state";

/**
 * A proposed write, shown where the assistant asked for it.
 *
 * Nothing on this card has happened. The assistant cannot make a change on
 * its own — it can only ask — so the card says in plain words what would
 * run, lists every parameter that would be sent, and separates the model's
 * reasoning from the facts so an approver can weigh one against the other.
 */
export function ProposalCard({
  proposal,
  threadId,
}: {
  proposal: AssistantProposal;
  threadId: string;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const presentation = classifyProposal(proposal);
  const view = presentProposal(proposal);
  const confidence = Math.round(Math.min(1, Math.max(0, proposal.confidence ?? 0)) * 100);

  const decideMutation = useApiMutation({
    mutationFn: (decision: ProposalDecision) =>
      apiService.assistantService.decideProposal(proposal.id, decision),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: queries.assistant.proposals(threadId).queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: queries.assistant.messages(threadId).queryKey,
        }),
        queryClient.invalidateQueries({ queryKey: ["assistant", "pending-proposals"] }),
      ]);
    },
    resourceName: "Proposal",
  });

  const awaiting = presentation === "awaiting";

  return (
    <m.div
      layout
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      className={cn(
        "bg-card max-w-[92%] overflow-hidden rounded-xl border shadow-xs",
        awaiting ? "border-warning/50" : "border-border/70",
      )}
    >
      <div
        className={cn(
          "flex items-start gap-3 px-3.5 py-3",
          awaiting && "from-warning/10 bg-gradient-to-r to-transparent",
        )}
      >
        <span
          className={cn(
            "flex size-8 shrink-0 items-center justify-center rounded-lg",
            awaiting ? "bg-warning/20 text-warning" : "bg-muted text-muted-foreground",
          )}
        >
          <ShieldQuestionIcon className="size-4" />
        </span>
        <div className="min-w-0 flex-1">
          <p className="text-muted-foreground text-[10px] font-medium tracking-wider uppercase">
            {t("Proposed change")}
          </p>
          <p className="text-sm font-semibold">{view.title}</p>
          <p className="text-muted-foreground mt-0.5 text-xs">{view.summary}</p>
        </div>
        <ProposalStatusBadge proposal={proposal} />
      </div>

      {view.facts.length > 0 && (
        <dl className="border-border/70 grid grid-cols-[minmax(0,auto)_minmax(0,1fr)] gap-x-4 gap-y-1 border-t px-3.5 py-2.5 text-xs">
          {view.facts.map((entry) => (
            <div key={entry.label} className="contents">
              <dt className="text-muted-foreground">{entry.label}</dt>
              <dd className="break-words">{entry.value}</dd>
            </div>
          ))}
        </dl>
      )}

      {view.longText && (
        <div className="border-border/70 border-t px-3.5 py-2.5 text-xs">
          <p className="text-muted-foreground mb-1 text-[10px] font-medium tracking-wider uppercase">
            {view.longText.label}
          </p>
          <p className="whitespace-pre-wrap">{view.longText.value}</p>
        </div>
      )}

      {(proposal.rationale !== "" || awaiting) && (
        <div className="border-border/70 flex flex-col gap-2 border-t px-3.5 py-2.5 text-xs">
          {proposal.rationale !== "" && (
            <p className="text-muted-foreground border-primary/40 border-l-2 pl-2.5 whitespace-pre-wrap">
              {proposal.rationale}
            </p>
          )}
          <div className="text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1">
            <span className="flex items-center gap-1.5">
              {t("Confidence")}
              <Progress value={confidence} className="h-1.5 w-16" />
              <span className="tabular-nums">{confidence}%</span>
            </span>
            <span className="flex items-center gap-1">
              <RotateCcwIcon className="size-3" />
              {view.reversible ? t("Can be undone") : t("Cannot be undone")}
            </span>
          </div>
        </div>
      )}

      <ProposalOutcome proposal={proposal} />

      {awaiting && (
        <div className="border-border/70 flex gap-2 border-t px-3.5 py-2.5">
          <Button
            size="sm"
            onClick={() => decideMutation.mutate("Accepted")}
            disabled={decideMutation.isPending}
            isLoading={decideMutation.isPending && decideMutation.variables === "Accepted"}
          >
            <CheckIcon className="size-3.5" />
            {t("Approve and run")}
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={() => decideMutation.mutate("Rejected")}
            disabled={decideMutation.isPending}
            isLoading={decideMutation.isPending && decideMutation.variables === "Rejected"}
          >
            <XIcon className="size-3.5" />
            {t("Reject")}
          </Button>
        </div>
      )}
    </m.div>
  );
}

/**
 * The outcome of a decision, stated separately from the decision itself.
 *
 * "Approved" and "done" are different facts and the card never runs them
 * together: an approval whose tool failed says so, with the reason, because the
 * approver is the one who needs to know their instruction did not take effect.
 */
function ProposalOutcome({ proposal }: { proposal: AssistantProposal }) {
  const t = useT();
  const presentation = classifyProposal(proposal);

  if (presentation === "failed") {
    return (
      <Alert variant="destructive" className="rounded-none border-x-0 border-b-0">
        <CircleAlertIcon className="size-4" />
        <AlertTitle>{t("Approved, but it did not run")}</AlertTitle>
        <AlertDescription>
          {proposal.executionError === ""
            ? t("The change was approved but could not be completed. Nothing was changed.")
            : proposal.executionError}
        </AlertDescription>
      </Alert>
    );
  }

  const line = (icon: React.ReactNode, text: string) => (
    <p className="text-muted-foreground border-border/70 flex items-center gap-1.5 border-t px-3.5 py-2 text-xs">
      {icon}
      {text}
    </p>
  );

  if (presentation === "done") {
    return line(
      <CircleCheckIcon className="size-3.5 text-emerald-600 dark:text-emerald-400" />,
      proposal.executedAt
        ? t("Ran on {0}", generateDateTimeStringFromUnixTimestamp(proposal.executedAt))
        : t("This change has been made."),
    );
  }

  if (presentation === "running") {
    return line(
      <LoaderIcon className="size-3.5 animate-spin" />,
      t("Approved. Waiting for it to run."),
    );
  }

  if (presentation === "declined") {
    return line(<CircleSlashIcon className="size-3.5" />, t("Rejected. Nothing was changed."));
  }

  return null;
}

function ProposalStatusBadge({ proposal }: { proposal: AssistantProposal }) {
  const t = useT();

  switch (classifyProposal(proposal)) {
    case "awaiting":
      return <Badge variant="warning">{t("Needs your approval")}</Badge>;
    case "running":
      return <Badge variant="info">{t("Approved")}</Badge>;
    case "done":
      return <Badge variant="active">{t("Done")}</Badge>;
    case "failed":
      return <Badge variant="inactive">{t("Failed")}</Badge>;
    case "declined":
      return <Badge variant="outline">{t("Rejected")}</Badge>;
    default:
      return <Badge variant="outline">{t("No longer available")}</Badge>;
  }
}
