import type { AssistantArtifact } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { ArtifactNotice } from "@/components/assistant/voice/artifact-chrome";
import { ArrowUpRightIcon, TriangleAlertIcon } from "lucide-react";
import { useMemo } from "react";
import { Link } from "react-router";
import { composedViewFrom } from "./artifact-payloads";

/**
 * A view somebody described, as something to open.
 *
 * The rows are deliberately not here. A table opened from this link is live,
 * sortable, exportable and re-checked against permissions on every page;
 * rows copied into the pane are a snapshot that is wrong by the time anybody
 * reads them. So the pane shows what it was narrowed to, what it could not
 * be, and the way in.
 */
export function ComposedViewArtifact({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const view = useMemo(() => composedViewFrom(artifact), [artifact]);

  if (view === null) {
    return (
      <ArtifactNotice kind={artifact.kind}>{t("This view has no table to open.")}</ArtifactNotice>
    );
  }

  return (
    <div className="animate-rise flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-4">
      <p className="text-sm leading-relaxed">{view.explanation}</p>

      {view.terms.length > 0 && (
        <ul className="flex flex-wrap gap-1.5">
          {view.terms.map((term) => (
            <li
              key={term}
              className="bg-sunken text-foreground-muted rounded-full px-2 py-0.5 text-xs"
            >
              {term}
            </li>
          ))}
        </ul>
      )}

      {/* A view that quietly lost a condition looks like an answer, so what it
          could not express sits next to what it could. */}
      {view.unresolved.length > 0 && (
        <div className="space-y-1.5">
          <p className="text-warning flex items-center gap-1.5 text-xs font-medium">
            <TriangleAlertIcon className="size-3" />
            {t("Not included")}
          </p>
          <ul className="space-y-1.5">
            {view.unresolved.map((entry) => (
              <li key={entry.phrase} className="text-xs">
                <span className="block">{entry.phrase}</span>
                <span className="text-muted-foreground block">{entry.reason}</span>
              </li>
            ))}
          </ul>
        </div>
      )}

      <Button size="sm" className="self-start" render={<Link to={view.path} />}>
        <ArrowUpRightIcon className="size-3.5" />
        {t("Open the table")}
      </Button>
    </div>
  );
}
