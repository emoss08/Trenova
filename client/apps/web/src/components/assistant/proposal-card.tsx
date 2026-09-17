import { Button } from "@trenova/shared/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { generateDateTimeStringFromUnixTimestamp } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useT } from "@trenova/shared/i18n/use-t";
import { toneVar } from "@/components/kpi/tone";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { AssistantProposal, ProposalDecision } from "@/types/assistant";
import { useQueryClient } from "@tanstack/react-query";
import {
  CheckIcon,
  ChevronRightIcon,
  CircleAlertIcon,
  CircleCheckIcon,
  CircleSlashIcon,
  LoaderIcon,
  TriangleAlertIcon,
  XIcon,
} from "lucide-react";
import { m } from "motion/react";
import { useState } from "react";
import { presentProposal } from "./proposal-presenters";
import { classifyProposal, type ProposalPresentation } from "./proposal-state";

/**
 * A change the assistant is asking for.
 *
 * The card answers one question — should this run — so it is built around the
 * sentence that answers it and nothing else. What the tool is called, which
 * identifiers it would send and how confident the model claims to be are not
 * inputs to that decision: the first is jargon, the second is a storage key, and
 * the third is a number no model can calibrate. They are either dropped or put
 * behind Details, which still holds the literal payload for anyone who wants it.
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
  const state = classifyProposal(proposal);
  const view = presentProposal(proposal);

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

  const awaiting = state === "awaiting";

  // Once a decision is made the card is history, not a question. It keeps the
  // sentence and the outcome and drops everything that existed to help decide.
  if (!awaiting) {
    return (
      <m.div
        layout
        className="border-border/70 text-muted-foreground flex items-start gap-2 rounded-lg border px-3 py-2 text-xs"
      >
        <OutcomeIcon state={state} />
        <span className="min-w-0 flex-1">
          <span className="text-foreground block">{view.summary}</span>
          <OutcomeLine proposal={proposal} state={state} />
        </span>
      </m.div>
    );
  }

  return (
    <m.div
      layout
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      className="bg-card ring-foreground/10 flex flex-col overflow-hidden rounded-xl shadow-sm ring-1"
    >
      <div className="flex flex-col gap-2 px-3.5 pt-3 pb-2.5">
        <div className="flex items-center gap-2">
          <span
            aria-hidden
            className="size-1.5 shrink-0 rounded-full"
            style={{ backgroundColor: toneVar("warning") }}
          />
          <span className="text-muted-foreground min-w-0 flex-1 truncate text-xs tracking-wide uppercase">
            {view.title}
          </span>
          {view.severity && (
            <span
              className="shrink-0 text-[10px] font-medium tracking-wide uppercase"
              style={{ color: toneVar(view.severity.tone) }}
            >
              {view.severity.label}
            </span>
          )}
        </div>

        <p className="text-sm leading-snug">{view.summary}</p>

        {view.highlights.length > 0 && (
          <dl className="flex flex-col gap-1 text-xs">
            {view.highlights.map((entry) => (
              <div key={entry.label} className="flex gap-2">
                <dt className="text-muted-foreground w-28 shrink-0 truncate">{entry.label}</dt>
                <dd className="min-w-0 flex-1 break-words">{entry.value}</dd>
              </div>
            ))}
          </dl>
        )}

        {proposal.rationale !== "" && (
          <p className="text-muted-foreground text-xs leading-relaxed whitespace-pre-wrap">
            {proposal.rationale}
          </p>
        )}
      </div>

      <div className="flex items-center gap-2 px-3 pb-3">
        <Button
          size="sm"
          onClick={() => decideMutation.mutate("Accepted")}
          disabled={decideMutation.isPending}
          isLoading={decideMutation.isPending && decideMutation.variables === "Accepted"}
        >
          <CheckIcon className="size-3.5" />
          {t("Approve")}
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

        <span className="ml-auto flex items-center gap-2">
          {/* Reversibility is only news one way round. Saying a change can be
              undone reassures nobody; saying it cannot is the thing to read. */}
          {!view.reversible && (
            <span
              className="flex items-center gap-1 text-[11px]"
              style={{ color: toneVar("warning") }}
            >
              <TriangleAlertIcon className="size-3" />
              {t("Permanent")}
            </span>
          )}
          {view.details.length > 0 && <DetailsDisclosure details={view.details} />}
        </span>
      </div>
    </m.div>
  );
}

function DetailsDisclosure({ details }: { details: { label: string; value: string }[] }) {
  const t = useT();
  const [open, setOpen] = useState(false);

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger className="text-muted-foreground hover:text-foreground flex items-center gap-0.5 text-[11px] transition-colors">
        <ChevronRightIcon
          className={cn(
            "size-3 transition-transform duration-150 ease-[cubic-bezier(0.2,0.8,0.2,1)]",
            open && "rotate-90",
          )}
        />
        {t("Details")}
      </CollapsibleTrigger>
      <CollapsibleContent>
        <dl className="border-border/70 mt-2 flex flex-col gap-1 border-t pt-2 text-[11px]">
          {details.map((entry) => (
            <div key={entry.label} className="flex gap-2">
              <dt className="text-muted-foreground w-28 shrink-0 truncate">{entry.label}</dt>
              <dd className="min-w-0 flex-1 font-mono break-all">{entry.value}</dd>
            </div>
          ))}
        </dl>
      </CollapsibleContent>
    </Collapsible>
  );
}

function OutcomeIcon({ state }: { state: ProposalPresentation }) {
  const className = "mt-px size-3.5 shrink-0";

  switch (state) {
    case "failed":
      return <CircleAlertIcon className={className} style={{ color: toneVar("danger") }} />;
    case "done":
      return <CircleCheckIcon className={className} style={{ color: toneVar("success") }} />;
    case "running":
      return <LoaderIcon className={cn(className, "animate-spin")} />;
    default:
      return <CircleSlashIcon className={className} />;
  }
}

/**
 * What became of the decision, stated separately from the decision itself.
 *
 * "Approved" and "done" are different facts and the card never runs them
 * together: an approval whose tool failed says so, with the reason, because the
 * approver is the one who needs to know their instruction did not take effect.
 */
function OutcomeLine({
  proposal,
  state,
}: {
  proposal: AssistantProposal;
  state: ProposalPresentation;
}) {
  const t = useT();

  switch (state) {
    case "failed":
      return (
        <span className="block" style={{ color: toneVar("danger") }}>
          {proposal.executionError === ""
            ? t("Approved, but it did not run. Nothing was changed.")
            : t("Approved, but it did not run: {0}", proposal.executionError)}
        </span>
      );
    case "done":
      return (
        <span className="block">
          {proposal.executedAt
            ? t("Done {0}", generateDateTimeStringFromUnixTimestamp(proposal.executedAt))
            : t("Done")}
        </span>
      );
    case "running":
      return <span className="block">{t("Approved. Waiting for it to run.")}</span>;
    case "declined":
      return <span className="block">{t("Rejected. Nothing was changed.")}</span>;
    default:
      return <span className="block">{t("Expired without a decision.")}</span>;
  }
}
