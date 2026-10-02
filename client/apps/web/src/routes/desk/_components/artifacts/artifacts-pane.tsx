import {
  ArtifactNotice,
  ARTIFACT_KINDS,
} from "@/components/assistant/voice/artifact-chrome";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { EASE_SETTLE } from "@/lib/motion";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useDeskStore } from "@/stores/desk-store";
import type { LiveArtifacts } from "../desk-layout";
import type { AssistantArtifact } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertAction, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { PanelRightCloseIcon } from "lucide-react";
import { AnimatePresence, m, useReducedMotion } from "motion/react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { ArtifactFilmstrip } from "./artifact-filmstrip";
import { orderArtifacts } from "./artifact-order";
import { DeskArtifactsEmpty } from "./desk-artifacts-empty";
import { ArtifactSwitcher } from "./artifact-switcher";
import { ComposedViewArtifact } from "./composed-view-artifact";
import { DecisionRequestArtifact } from "./decision-request-artifact";
import { DocumentArtifact } from "./document-artifact";
import { EmailDraftArtifact } from "./email-draft-artifact";
import { EntityCardArtifact } from "./entity-card-artifact";
import { NavigationArtifact } from "./navigation-artifact";
import { PlanArtifact } from "./plan-artifact";
import { RateExplanationArtifact } from "./rate-explanation-artifact";
import { ReportPreviewArtifact } from "./report-preview-artifact";
import { ReportRunArtifact } from "./report-run-artifact";
import { RunDiffArtifact } from "./run-diff-artifact";
import { TableViewArtifact } from "./table-view-artifact";

/** How long the header wears the mark of a new artifact landing, before it fades. */
const ARRIVAL_MS = 1200;

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


/** The pane's top line when there is no artifact to switch between: its name and the way to hide it. */
function PaneHeader({ onClose }: { onClose: () => void }) {
  const t = useT();

  return (
    <div className="border-border-subtle flex h-12 shrink-0 items-center gap-2 border-b pr-1.5 pl-3">
      <span className="text-sm font-medium">{t("Artifacts")}</span>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              variant="ghost"
              size="icon-sm"
              className="text-foreground-subtle hover:text-foreground ml-auto"
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
  );
}

/** The pane's shape while the list loads: the switcher, the strip and one body. */
function ArtifactsLoading() {
  const t = useT();

  return (
    <div role="status" aria-label={t("Loading artifacts")} className="flex min-h-0 flex-1 flex-col">
      <div className="border-border-subtle flex h-12 shrink-0 items-center gap-2 border-b pr-1.5 pl-2">
        <Skeleton className="size-7 rounded-md" />
        <div className="flex flex-1 flex-col gap-1.5">
          <Skeleton className="h-3 w-40" />
          <Skeleton className="h-2 w-24" />
        </div>
        <Skeleton className="h-3 w-8" />
        <Skeleton className="size-7 rounded-md" />
        <Skeleton className="size-7 rounded-md" />
      </div>
      <div className="border-border-subtle flex h-10 shrink-0 items-center gap-1 border-b px-2">
        <Skeleton className="size-8 rounded-md" />
        <Skeleton className="size-8 rounded-md" />
        <Skeleton className="size-8 rounded-md" />
      </div>
      <div className="flex flex-col gap-2 p-4">
        <Skeleton className="h-6" />
        <Skeleton className="h-6" />
        <Skeleton className="h-6 w-3/4" />
      </div>
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
    case "run_diff":
      return <RunDiffArtifact artifact={artifact} />;
    case "document":
      return <DocumentArtifact artifact={artifact} />;
    case "navigation":
      return <NavigationArtifact artifact={artifact} />;
    case "decision_request":
      return <DecisionRequestArtifact artifact={artifact} />;
    default:
      return (
        <ArtifactNotice kind={artifact.kind}>
          {t("This kind of artifact cannot be shown here yet.")}
        </ArtifactNotice>
      );
  }
}

/**
 * What the conversation produced, beside it. A switcher over everything
 * there is, pinned first and newest first, a strip of their marks, and the
 * one that is open rendered whole underneath. The transcript refers to
 * these; this is where they are read.
 */
export function ArtifactsPane({
  threadId,
  liveArtifacts,
  onClose,
  className,
}: {
  threadId: string;
  /** What a streaming turn has produced so far, so the pane opens the newest as it lands and follows the set. */
  liveArtifacts: LiveArtifacts;
  onClose: () => void;
  className?: string;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const artifactsQuery = useQuery(queries.assistant.artifacts(threadId));
  const artifacts = useMemo(
    () => orderArtifacts(artifactsQuery.data?.results ?? []),
    [artifactsQuery.data],
  );

  const remembered = useDeskStore((state) => state.activeArtifactByThread[threadId]);
  const setActiveArtifact = useDeskStore((state) => state.setActiveArtifact);
  const activeId = defaultArtifactId(artifacts, remembered);
  const active = artifacts.find((artifact) => artifact.id === activeId) ?? null;

  // A turn that just produced something opens it: the reader asked for a
  // table and the table is what they are waiting for. Every revision of the
  // set re-reads the list, so a table that grew with a later read shows its
  // new rows and a card that read folded away leaves the list. The header
  // marks the landing for a moment, once, and then rests.
  const newestLive = liveArtifacts.ids.at(-1);
  const liveRevision = liveArtifacts.revision;
  useEffect(() => {
    if (liveRevision === 0) {
      return;
    }
    void queryClient.invalidateQueries({
      queryKey: queries.assistant.artifacts(threadId).queryKey,
    });
    if (newestLive) {
      setActiveArtifact(threadId, newestLive);
    }
  }, [liveRevision, newestLive, queryClient, setActiveArtifact, threadId]);
  // A revision the header has not yet settled on is one that just landed;
  // the revision the pane mounted with is already settled, so reopening a
  // conversation mid-turn does not flash.
  const [settledRevision, setSettledRevision] = useState(liveRevision);
  const arrived =
    liveRevision !== 0 && newestLive !== undefined && liveRevision !== settledRevision;
  useEffect(() => {
    if (!arrived) {
      return;
    }
    const timer = window.setTimeout(() => setSettledRevision(liveRevision), ARRIVAL_MS);

    return () => window.clearTimeout(timer);
  }, [arrived, liveRevision]);

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
  const pin = useCallback(
    (id: string, pinned: boolean) => pinMutation.mutate({ id, pinned }),
    [pinMutation],
  );

  const reduceMotion = useReducedMotion();

  return (
    <aside
      data-slot="artifacts-pane"
      aria-label={t("Artifacts")}
      className={cn("bg-card flex min-h-0 min-w-0 flex-col", className)}
    >
      {artifactsQuery.isLoading ? (
        <ArtifactsLoading />
      ) : artifactsQuery.isError ? (
        <>
          <PaneHeader onClose={onClose} />
          <div className="p-3">
            <Alert size="sm" variant="destructive">
              <AlertDescription>
                {t("This conversation's artifacts could not be loaded.")}
              </AlertDescription>
              <AlertAction>
                <Button variant="outline" size="xs" onClick={() => void artifactsQuery.refetch()}>
                  {t("Try again")}
                </Button>
              </AlertAction>
            </Alert>
          </div>
        </>
      ) : active === null ? (
        <DeskArtifactsEmpty onClose={onClose} />
      ) : (
        <>
          <ArtifactSwitcher
            artifacts={artifacts}
            active={active}
            arrived={arrived}
            onOpen={open}
            onPin={pin}
            onClose={onClose}
          />
          <ArtifactFilmstrip
            artifacts={artifacts}
            activeId={active.id}
            threadId={threadId}
            onOpen={open}
          />
          <div className="relative flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
            {/* Keyed on the artifact so opening one replays the arrival: the
                one leaving lifts away and the one picked settles into place,
                which is what makes the switch read as turning a page rather
                than repainting a panel. */}
            <AnimatePresence mode="wait" initial={false}>
              <m.div
                key={active.id}
                data-slot="artifact-body"
                initial={reduceMotion ? false : { opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                exit={
                  reduceMotion
                    ? { opacity: 0, transition: { duration: 0 } }
                    : { opacity: 0, y: -8, transition: { duration: 0.12, ease: EASE_SETTLE } }
                }
                transition={{ duration: 0.22, ease: EASE_SETTLE }}
                className="flex min-h-0 min-w-0 flex-1 flex-col"
              >
                <ArtifactBody artifact={active} />
              </m.div>
            </AnimatePresence>
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
      "run_diff",
      "document",
      "navigation",
      "decision_request",
    ].includes(kind)
  );
}
