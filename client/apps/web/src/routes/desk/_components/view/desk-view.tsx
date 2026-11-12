import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { conversationPath } from "@/lib/conversation-path";
import { assistant } from "@/lib/queries/assistant";
import { useDeskMarginStore } from "@/stores/desk-margin-store";
import { useQuery } from "@tanstack/react-query";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort, resolveUserTimezone } from "@trenova/shared/lib/date";
import { buttonVariants } from "@trenova/shared/lib/variants/button";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useEffect, useMemo } from "react";
import { Link, useNavigate } from "react-router";
import { deskArtKind, deskArtKindName } from "../artifacts/desk-art-kinds";
import { groupLineages } from "../artifacts/desk-lineage";
import { ArtifactBody } from "../artifacts/desk-workspace";
import { useDesk } from "../desk-layout";
import { DeskViewSkeleton } from "./view-skeleton";
import { viewPath } from "./view-path";
import "./view.css";

/** The width from which Nova's margin sits beside the canvas rather than over it. */
const MARGIN_BESIDE = "(min-width: 901px)";

/**
 * What an agent laid out in answer to a question, on the canvas: the question
 * it answers, what it is called, who composed it and when its figures were
 * read, then the whole of it. The conversation that made it sits in Nova's
 * margin, so a follow-up is asked beside the work rather than in front of it.
 */
export function DeskView({ threadId, slug }: { threadId: string; slug: string }) {
  const t = useT();
  const navigate = useNavigate();
  const desk = useDesk();
  const timezone = resolveUserTimezone(useAuthStore((state) => state.user?.timezone));
  const lineageQuery = useQuery({
    ...assistant.artifacts(threadId)._ctx.slug(slug),
    enabled: threadId !== "" && slug !== "",
    retry: false,
  });
  const threadQuery = useQuery({ ...assistant.thread(threadId), enabled: threadId !== "" });
  const lineage = useMemo(
    () => groupLineages(lineageQuery.data?.results ?? [])[0] ?? null,
    [lineageQuery.data],
  );
  const artifact = lineage?.latest ?? null;
  const title = artifact?.title ?? null;
  const setPageTitle = useDeskMarginStore((state) => state.setPageTitle);
  const noteOpenWork = useDeskMarginStore((state) => state.noteOpenWork);
  const showThread = useDeskMarginStore((state) => state.showThread);

  // The conversation opens beside the view once, when the view does; closing
  // the margin after that is the person's choice and stays made. In a window
  // too narrow for both, the margin would cover the view, so it stays shut
  // until asked for.
  useEffect(() => {
    if (threadId !== "" && window.matchMedia(MARGIN_BESIDE).matches) {
      showThread(threadId, null);
    }
  }, [showThread, threadId]);
  useEffect(() => {
    setPageTitle(title);
    if (title !== null) {
      noteOpenWork({
        kind: "view",
        id: `${threadId}/${slug}`,
        label: title,
        path: viewPath(threadId, slug),
      });
    }
    return () => setPageTitle(null);
  }, [noteOpenWork, setPageTitle, slug, threadId, title]);

  if (lineageQuery.isPending) {
    return <DeskViewSkeleton />;
  }
  if (lineage === null || artifact === null) {
    return (
      <EmptySheet
        className="my-auto"
        title={t("This view is not here")}
        description={t("It may have been deleted, or it belongs to someone else's conversation.")}
        action={
          <Link
            to="/desk"
            data-slot="button"
            className={buttonVariants({ variant: "outline", size: "sm" })}
          >
            {t("Back to Today")}
          </Link>
        }
        sketch={
          <div className="flex flex-col gap-3">
            <GhostLine className="w-2/3" />
            <GhostLine className="w-1/2" />
          </div>
        }
      />
    );
  }

  const kind = deskArtKind(artifact);
  const agent = threadQuery.data
    ? (desk.agentsById.get(threadQuery.data.agentDefinitionId) ?? null)
    : null;
  const previous =
    lineage.versions.length > 1 ? lineage.versions[lineage.versions.length - 2] : null;
  const read = formatUnixDateTimeShort(artifact.updatedAt, { timezone });

  return (
    <div className="dk-vw-scroll">
      <article className="dk-vw">
        {artifact.turn !== "" && (
          <div className="dk-vw-ask">
            <span>{t("You asked")}</span>
            <span className="dk-vw-q" title={artifact.turn}>
              {artifact.turn}
            </span>
          </div>
        )}
        <h1 className="dk-vw-h">{artifact.title}</h1>
        <div className="dk-vw-prov">
          <span className="dk-vw-live">
            <i aria-hidden />
            {t("Read {0}", read)}
          </span>
          {agent && <DeskAgentTile agent={agent} size="xs" />}
          <span>
            {agent
              ? t("Composed by {0} from your records.", agent.name)
              : t("Composed from your records.")}{" "}
            {lineage.versions.length > 1
              ? t("{0}, version {1}", deskArtKindName(kind, t), artifact.lineageSeq)
              : deskArtKindName(kind, t)}
          </span>
          <span className="dk-vw-sp" />
          <Link
            className={buttonVariants({ variant: "ghost", size: "sm" })}
            to={`${conversationPath(threadId)}?a=${encodeURIComponent(artifact.lineageId || artifact.id)}`}
          >
            <DeskIcon name="chat" size={12} />
            {t("Open in the conversation")}
          </Link>
        </div>
        <div className="dk-vw-body">
          <ArtifactBody
            artifact={artifact}
            previous={previous}
            versions={null}
            lineage={lineage}
            onOpenArtifact={(id) => void navigate(viewPath(threadId, id))}
          />
        </div>
      </article>
    </div>
  );
}
