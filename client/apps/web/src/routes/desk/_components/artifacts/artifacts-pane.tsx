import {
  ArtifactChrome,
  ArtifactKindIcon,
  ARTIFACT_KINDS,
} from "@/components/assistant/voice/artifact-chrome";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useDeskStore } from "@/stores/desk-store";
import type { AssistantArtifact } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { PanelRightCloseIcon, PinIcon } from "lucide-react";
import { useCallback, useEffect, useMemo } from "react";
import { EmailDraftArtifact } from "./email-draft-artifact";
import { EntityCardArtifact } from "./entity-card-artifact";
import { PlanArtifact } from "./plan-artifact";
import { ReportPreviewArtifact } from "./report-preview-artifact";
import { ReportRunArtifact } from "./report-run-artifact";
import { ComposedViewArtifact } from "./composed-view-artifact";
import { RateExplanationArtifact } from "./rate-explanation-artifact";
import { TableViewArtifact } from "./table-view-artifact";

/** The artifact to show when the person has not picked one: the newest. */
export function defaultArtifactId(
  artifacts: readonly AssistantArtifact[],
  remembered: string | undefined,
): string | null {
  if (remembered && artifacts.some((artifact) => artifact.id === remembered)) {
    return remembered;
  }
  const newest = [...artifacts].sort((a, b) => b.createdAt - a.createdAt)[0];

  return newest?.id ?? null;
}

/**
 * The kinds this pane holds, in the order a conversation tends to produce
 * them, with the sentence that earns each one.
 *
 * An empty pane is the first thing a person sees on a new desk, so it is
 * doing the teaching: grey boxes said only that nothing was here, which they
 * could already see. Naming what will appear says what the pane is for and,
 * more usefully, what to ask for to fill it.
 */
const ARTIFACT_PROMISES = [
  { kind: "table_view", example: "Which shipments are in transit?" },
  { kind: "report_run", example: "Run the detention report for last week." },
  { kind: "entity_card", example: "Show me shipment SEED-SHP-008." },
  { kind: "email_draft", example: "Tell the customer their load is running late." },
] as const;

function ArtifactsEmpty() {
  const t = useT();

  return (
    <div className="flex min-h-0 flex-1 flex-col justify-center gap-4 px-4 py-6">
      <div className="space-y-1">
        <p className="text-sm font-medium">{t("Nothing here yet")}</p>
        <p className="text-muted-foreground text-xs">
          {t(
            "What a turn makes — a table, a report, a record, a draft — opens here beside the conversation.",
          )}
        </p>
      </div>

      <ul className="space-y-1.5">
        {ARTIFACT_PROMISES.map(({ kind, example }, index) => (
          <li
            key={kind}
            style={{ animationDelay: `${index * 45}ms` }}
            className={cn(
              "animate-land border-desk-hairline flex items-start gap-2.5",
              "rounded-surface border px-2.5 py-2",
            )}
          >
            <ArtifactKindIcon
              kind={kind}
              className="text-muted-foreground mt-0.5 size-3.5 shrink-0"
            />
            <span className="min-w-0 flex-1">
              <span className="block text-xs">{t(ARTIFACT_KINDS[kind].label)}</span>
              <span className="text-muted-foreground block text-2xs">
                {t("“{example}”", { example })}
              </span>
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function ArtifactBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();

  switch (artifact.kind) {
    case "report_preview":
      return <ReportPreviewArtifact artifact={artifact} />;
    case "report_run":
      return <ReportRunArtifact artifact={artifact} />;
    case "email_draft":
      return <EmailDraftArtifact artifact={artifact} />;
    case "plan":
      return <PlanArtifact artifact={artifact} />;
    case "entity_card":
      return <EntityCardArtifact artifact={artifact} />;
    case "table_view":
      // Two things arrive as a table_view: a list result, which carries its
      // rows, and a described view, which carries the link that opens them
      // live. The payload says which.
      return "path" in artifact.payload ? (
        <ComposedViewArtifact artifact={artifact} />
      ) : (
        <TableViewArtifact artifact={artifact} />
      );
    case "rate_explanation":
      return <RateExplanationArtifact artifact={artifact} />;
    default:
      return (
        <p className="text-muted-foreground p-4 text-sm">
          {t("This kind of artifact cannot be shown here yet.")}
        </p>
      );
  }
}

/**
 * What the conversation produced, beside it. A row of what there is, pinned
 * first, and the one that is open rendered whole underneath. The transcript
 * refers to these; this is where they are read.
 */
export function ArtifactsPane({
  threadId,
  liveArtifactIds,
  onClose,
  className,
}: {
  threadId: string;
  /** Artifacts a streaming turn has announced, so the pane opens the newest as it lands. */
  liveArtifactIds: readonly string[];
  onClose: () => void;
  className?: string;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const artifactsQuery = useQuery(queries.assistant.artifacts(threadId));
  const artifacts = useMemo(() => artifactsQuery.data?.results ?? [], [artifactsQuery.data]);

  const remembered = useDeskStore((state) => state.activeArtifactByThread[threadId]);
  const setActiveArtifact = useDeskStore((state) => state.setActiveArtifact);
  const activeId = defaultArtifactId(artifacts, remembered);
  const active = artifacts.find((artifact) => artifact.id === activeId) ?? null;

  // A turn that just produced something opens it: the reader asked for a
  // table and the table is what they are waiting for.
  const newestLive = liveArtifactIds.at(-1);
  useEffect(() => {
    if (!newestLive) {
      return;
    }
    void queryClient.invalidateQueries({
      queryKey: queries.assistant.artifacts(threadId).queryKey,
    });
    setActiveArtifact(threadId, newestLive);
  }, [newestLive, queryClient, setActiveArtifact, threadId]);

  const pinMutation = useApiMutation({
    mutationFn: ({ id, pinned }: { id: string; pinned: boolean }) =>
      apiService.assistantService.pinArtifact(threadId, id, pinned),
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: queries.assistant.artifacts(threadId).queryKey }),
    resourceName: "Artifact",
  });

  const open = useCallback(
    (id: string) => setActiveArtifact(threadId, id),
    [setActiveArtifact, threadId],
  );

  return (
    <aside
      data-slot="artifacts-pane"
      aria-label={t("Artifacts")}
      className={cn("bg-desk-canvas flex min-h-0 min-w-0 flex-col", className)}
    >
      <div className="flex h-11 shrink-0 items-center gap-2 pr-1.5 pl-3">
        <span className="text-sm font-medium">{t("Artifacts")}</span>
        {artifacts.length > 0 && (
          <span className="text-muted-foreground text-xs tabular-nums">{artifacts.length}</span>
        )}
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="ghost"
                size="icon-sm"
                className="text-muted-foreground hover:text-foreground ml-auto"
                aria-label={t("Hide artifacts")}
                onClick={onClose}
              />
            }
          >
            <PanelRightCloseIcon className="size-4" />
          </TooltipTrigger>
          <TooltipContent>{t("Hide artifacts")}</TooltipContent>
        </Tooltip>
      </div>

      {artifactsQuery.isLoading ? (
        <div className="flex flex-col gap-2 px-3">
          <Skeleton className="h-8" />
          <Skeleton className="h-48" />
        </div>
      ) : artifacts.length === 0 ? (
        <ArtifactsEmpty />
      ) : (
        <>
          <div className="scrollbar-overlay shrink-0 overflow-x-auto">
            <div role="tablist" aria-label={t("Artifacts")} className="flex gap-1 px-3 pb-2">
              {artifacts.map((artifact, index) => (
                <button
                  key={artifact.id}
                  type="button"
                  role="tab"
                  aria-selected={artifact.id === activeId}
                  onClick={() => open(artifact.id)}
                  // Staggered so a conversation's output reads as a row
                  // being dealt rather than a block appearing.
                  style={{ animationDelay: `${Math.min(index, 8) * 35}ms` }}
                  className={cn(
                    "animate-land ui-focus-ring flex h-7 max-w-56 shrink-0 items-center gap-1.5",
                    "rounded-full px-2.5 text-xs transition-colors",
                    artifact.id === activeId
                      ? "bg-foreground text-background"
                      : "bg-card text-muted-foreground hover:text-foreground ring-foreground/10 ring-1",
                  )}
                >
                  {artifact.pinned && <PinIcon className="size-3 shrink-0" />}
                  <ArtifactKindIcon kind={artifact.kind} className="size-3 shrink-0" />
                  <span className="truncate">{artifact.title}</span>
                </button>
              ))}
            </div>
          </div>

          <div className="flex min-h-0 flex-1 flex-col px-3 pb-3">
            {active && (
              <ArtifactChrome
                // Keyed on the artifact so opening one replays the arrival:
                // it comes in from the conversation that made it rather than
                // fading in where it stands, which is what makes the two
                // columns read as one motion instead of two panes.
                key={active.id}
                kind={active.kind}
                title={active.title}
                status={active.status}
                pinned={active.pinned}
                onPin={(pinned) => pinMutation.mutate({ id: active.id, pinned })}
                className="animate-materialise min-h-0 flex-1"
              >
                <ArtifactBody artifact={active} />
              </ArtifactChrome>
            )}
          </div>
        </>
      )}
    </aside>
  );
}

/** The kinds the pane can render, for a chip that promises to open one. */
export function isRenderableArtifactKind(kind: AssistantArtifact["kind"]): boolean {
  return (
    kind in ARTIFACT_KINDS &&
    [
      "report_preview",
      "report_run",
      "email_draft",
      "plan",
      "entity_card",
      "table_view",
      "rate_explanation",
    ].includes(kind)
  );
}
