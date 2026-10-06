import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { SectionPanel } from "@/components/section-panel";
import type { AgentRunRow } from "@/lib/graphql/agent-activity-tables";
import { queries } from "@/lib/queries";
import { downloadAgentRunTranscript } from "@/services/agent-run";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import {
  AlertCircleIcon,
  ChevronRightIcon,
  Download01Icon,
} from "@trenova/shared/components/icons";
import { useState } from "react";
import { agentTypeLabel, RunStatusBadge, TriggerBadge } from "./agent-badges";
import { RunTranscriptView } from "./run-transcript-view";

/**
 * One run, read: how it ended and what it said, and behind a disclosure the
 * transcript of how it got there. The transcript is read only when opened,
 * because a long run's record is the largest thing on the row.
 */
export function AgentRunPanel({ open, onOpenChange, row }: DataTablePanelProps<AgentRunRow>) {
  const t = useT();

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={row ? agentTypeLabel(row.agentType, t) : t("Agent run")}
      description={row?.createdAt ? formatUnixDateTimeMedium(row.createdAt) : undefined}
      size="lg"
    >
      {row ? (
        <div className="flex flex-col gap-3">
          <SectionPanel title={t("Run")}>
            <div className="p-3">
              <DescriptionList layout="inline">
                <DescriptionItem label={t("Status")}>
                  <RunStatusBadge value={row.status} t={t} />
                </DescriptionItem>
                <DescriptionItem label={t("Started by")}>
                  <TriggerBadge value={row.trigger} t={t} />
                </DescriptionItem>
                <DescriptionItem label={t("Model")}>
                  {row.modelIdentifier || <DescriptionEmpty />}
                </DescriptionItem>
                <DescriptionItem label={t("Summary")}>
                  {row.summary ? (
                    <span className="break-words whitespace-pre-wrap">{row.summary}</span>
                  ) : (
                    <DescriptionEmpty />
                  )}
                </DescriptionItem>
                {row.errorMessage ? (
                  <DescriptionItem label={t("Error")}>
                    <span className="text-danger break-words">{row.errorMessage}</span>
                  </DescriptionItem>
                ) : null}
              </DescriptionList>
            </div>
          </SectionPanel>
          <TranscriptDisclosure runId={row.id} active={open} />
        </div>
      ) : (
        <div className="flex flex-col gap-3" aria-busy>
          <Skeleton className="h-24" />
          <Skeleton className="h-10" />
        </div>
      )}
    </DataTablePanelContainer>
  );
}

function TranscriptDisclosure({ runId, active }: { runId: string; active: boolean }) {
  const t = useT();
  const [expanded, setExpanded] = useState(false);
  const transcript = useQuery({
    ...queries.agentRun.transcript(runId),
    enabled: active && expanded,
  });

  return (
    <Collapsible
      open={expanded}
      onOpenChange={setExpanded}
      className="border-border min-w-0 rounded-surface border"
    >
      <CollapsibleTrigger className="group/transcript ui-focus-ring hover:bg-surface-hover flex w-full items-center gap-2 rounded-surface px-3 py-2 text-left text-sm font-medium transition-colors">
        <ChevronRightIcon
          aria-hidden
          className={cn(
            "text-foreground-subtle size-3.5 transition-transform duration-200",
            expanded && "rotate-90",
          )}
        />
        {t("Transcript")}
      </CollapsibleTrigger>
      <CollapsibleContent className="h-(--collapsible-panel-height) overflow-hidden transition-[height] duration-200 ease-settle data-ending-style:h-0 data-starting-style:h-0">
        <div className="border-border border-t p-3">
          {transcript.isError ? (
            <Alert variant="destructive" size="sm">
              <AlertCircleIcon />
              <AlertDescription>
                {t("The transcript could not be loaded. Try again shortly.")}
              </AlertDescription>
            </Alert>
          ) : transcript.isSuccess ? (
            transcript.data === null ? (
              <p className="text-foreground-muted text-sm">
                {t(
                  "This run kept no transcript. Runs filed before transcripts were kept, and runs that said nothing, have only their summary.",
                )}
              </p>
            ) : (
              <div className="flex flex-col gap-2">
                <div className="flex justify-end">
                  <Button
                    variant="ghost"
                    size="xs"
                    aria-label={t("Download transcript")}
                    onClick={() => downloadAgentRunTranscript(runId)}
                  >
                    <Download01Icon className="size-3" />
                    {t("Download")}
                  </Button>
                </div>
                <RunTranscriptView runId={runId} transcript={transcript.data} />
              </div>
            )
          ) : (
            <div className="flex flex-col gap-2" aria-busy>
              <Skeleton className="h-6" />
              <Skeleton className="h-16" />
            </div>
          )}
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}
