import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { generateDateTimeStringFromUnixTimestamp } from "@trenova/shared/lib/date";
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
  ShieldQuestionIcon,
  XIcon,
} from "lucide-react";
import { argumentRows, classifyProposal, humanizeToolName } from "./proposal-state";

/**
 * A proposed write, shown where the assistant asked for it.
 *
 * Nothing on this card has happened. The assistant cannot make a change on its
 * own — it can only ask — and the card exists so a person can read exactly what
 * would run before allowing it. That is why the parameters are shown in full
 * rather than summarized: the rationale is the model's claim about why, and the
 * parameters are the only statement of what.
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
  const rows = argumentRows(proposal.arguments);

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
      ]);
    },
    resourceName: "Proposal",
  });

  return (
    <div className="border-warning/40 bg-card max-w-[92%] rounded-lg border p-3">
      <div className="flex items-start justify-between gap-2">
        <div className="flex items-start gap-2.5">
          <span className="bg-warning/15 text-warning flex size-7 shrink-0 items-center justify-center rounded-md">
            <ShieldQuestionIcon className="size-4" />
          </span>
          <div className="flex flex-col gap-0.5">
            <span className="text-sm font-medium">{humanizeToolName(proposal.toolName)}</span>
            <span className="text-muted-foreground text-xs">
              {presentation === "awaiting"
                ? t("The assistant is asking to make this change. Nothing has run.")
                : t("A change the assistant asked to make.")}
            </span>
          </div>
        </div>
        <ProposalStatusBadge proposal={proposal} />
      </div>

      {proposal.rationale !== "" && (
        <p className="text-muted-foreground mt-2 text-xs whitespace-pre-wrap">
          {proposal.rationale}
        </p>
      )}

      {rows.length > 0 && (
        <dl className="border-border mt-2 grid grid-cols-[minmax(0,auto)_minmax(0,1fr)] gap-x-3 gap-y-1 border-t pt-2 text-xs">
          {rows.map((row) => (
            <div key={row.key} className="contents">
              <dt className="text-muted-foreground">{row.key}</dt>
              <dd className="break-all">{row.value}</dd>
            </div>
          ))}
        </dl>
      )}

      <ProposalOutcome proposal={proposal} />

      {presentation === "awaiting" && (
        <div className="mt-3 flex gap-2">
          <Button
            size="sm"
            onClick={() => decideMutation.mutate("Accepted")}
            disabled={decideMutation.isPending}
          >
            <CheckIcon className="size-3.5" />
            {t("Approve and run")}
          </Button>
          <Button
            size="sm"
            variant="outline"
            onClick={() => decideMutation.mutate("Rejected")}
            disabled={decideMutation.isPending}
          >
            <XIcon className="size-3.5" />
            {t("Reject")}
          </Button>
        </div>
      )}
    </div>
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
      <Alert variant="destructive" className="mt-3">
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

  if (presentation === "done") {
    return (
      <p className="text-muted-foreground mt-3 flex items-center gap-1.5 text-xs">
        <CircleCheckIcon className="size-3.5" />
        {proposal.executedAt
          ? t("Ran on {0}", generateDateTimeStringFromUnixTimestamp(proposal.executedAt))
          : t("This change has been made.")}
      </p>
    );
  }

  if (presentation === "running") {
    return (
      <p className="text-muted-foreground mt-3 flex items-center gap-1.5 text-xs">
        <LoaderIcon className="size-3.5" />
        {t("Approved. Waiting for it to run.")}
      </p>
    );
  }

  if (presentation === "declined") {
    return (
      <p className="text-muted-foreground mt-3 flex items-center gap-1.5 text-xs">
        <CircleSlashIcon className="size-3.5" />
        {t("Rejected. Nothing was changed.")}
      </p>
    );
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
