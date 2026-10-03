import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo } from "react";
import { Link, useNavigate, useParams } from "react-router";
import {
  ArtIcon,
  DeskArtKindIcon,
  deskArtKind,
  deskArtKindName,
} from "./_components/artifacts/desk-art-kinds";
import { groupLineages } from "./_components/artifacts/desk-lineage";
import { ArtifactBody, artifactLink } from "./_components/artifacts/desk-workspace";
import "./_styles/desk-v2.css";

/**
 * An artifact on a page of its own: the whole of it, at reading width, with
 * the way back to the conversation that made it. It is what "Open on its own
 * page" opens, and what a report too long for the pane opens as.
 */
export function DeskArtifactPage() {
  const t = useT();
  const navigate = useNavigate();
  const { threadId = "", slug = "" } = useParams<{ threadId: string; slug: string }>();
  const lineageQuery = useQuery({
    ...queries.assistant.artifactBySlug(threadId, slug),
    enabled: threadId !== "" && slug !== "",
    retry: false,
  });
  const lineage = useMemo(
    () => groupLineages(lineageQuery.data?.results ?? [])[0] ?? null,
    [lineageQuery.data],
  );

  if (lineageQuery.isPending) {
    return <div className="dsk dk-apage" aria-busy />;
  }
  if (!lineage) {
    return (
      <div className="dsk dk-apage">
        <div className="dk-apage-in">
          <p className="dk-ax-oldnote">{t("This artifact is not here.")}</p>
          <Link className="dk-ax-btn dk-ghost" to="/desk">
            {t("Back to the desk")}
          </Link>
        </div>
      </div>
    );
  }

  const artifact = lineage.latest;
  const kind = deskArtKind(artifact);
  const previous =
    lineage.versions.length > 1 ? lineage.versions[lineage.versions.length - 2] : null;

  return (
    <div className="dsk dk-apage">
      <div className="dk-apage-in">
        <header className="dk-apage-h">
          <span className={`dk-ax-ki dk-k-${kind}`}>
            <DeskArtKindIcon kind={kind} />
          </span>
          <span className="dk-ax-ct">
            <b>{artifact.title}</b>
            <span>
              {deskArtKindName(kind, t)}
              {lineage.versions.length > 1 ? ` · v${artifact.lineageSeq}` : ""}
            </span>
          </span>
          <Link className="dk-ax-btn dk-ghost" to={artifactLink(artifact)}>
            <ArtIcon name="ext" size={13} />
            {t("Open in conversation")}
          </Link>
        </header>
        <div className="dk-apage-body">
          <ArtifactBody
            artifact={artifact}
            previous={previous}
            versions={null}
            lineage={lineage}
            onOpenArtifact={(id) => void navigate(`/desk/t/${threadId}?a=${id}`)}
          />
        </div>
      </div>
    </div>
  );
}
