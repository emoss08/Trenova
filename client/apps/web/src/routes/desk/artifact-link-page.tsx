import { assistant } from "@/lib/queries/assistant";
import { useQuery } from "@tanstack/react-query";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { buttonVariants } from "@trenova/shared/lib/variants/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { Link, Navigate, useParams } from "react-router";

/**
 * A link to one artifact, `desk/c/{conversation}/a/{slug}`: the slug is read
 * on the server, and the conversation opens with that artifact beside it.
 */
export function DeskArtifactLinkPage() {
  const t = useT();
  const { threadId = "", slug = "" } = useParams<{ threadId: string; slug: string }>();
  const lineageQuery = useQuery({
    ...assistant.artifacts(threadId)._ctx.slug(slug),
    enabled: threadId !== "" && slug !== "",
    retry: false,
  });
  const root = lineageQuery.data?.results[0];

  if (root) {
    return <Navigate to={`/desk/t/${threadId}?a=${root.lineageId || root.id}`} replace />;
  }
  if (lineageQuery.isError) {
    return (
      <EmptySheet
        className="my-auto"
        title={t("This artifact is not here")}
        description={t("It may have been deleted, or it belongs to someone else's conversation.")}
        action={
          <Link
            to="/desk"
            data-slot="button"
            className={buttonVariants({ variant: "outline", size: "sm" })}
          >
            {t("Back to the desk")}
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
  return null;
}
