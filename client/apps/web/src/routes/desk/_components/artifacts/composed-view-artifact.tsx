import type { AssistantArtifact } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { toneVar } from "@/components/kpi/tone";
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
      <p className="text-muted-foreground p-4 text-sm">{t("This view has no table to open.")}</p>
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-3 p-3">
      <p className="text-sm">{view.explanation}</p>

      {view.terms.length > 0 && (
        <ul className="flex flex-wrap gap-1.5">
          {view.terms.map((term) => (
            <li
              key={term}
              className="bg-card ring-foreground/10 rounded-full px-2 py-0.5 text-xs ring-1"
            >
              {term}
            </li>
          ))}
        </ul>
      )}

      {/* A view that quietly lost a condition looks like an answer, so what it
          could not express sits next to what it could. */}
      {view.unresolved.length > 0 && (
        <div className="space-y-1">
          <p
            className="flex items-center gap-1.5 text-xs font-medium"
            style={{ color: toneVar("warning") }}
          >
            <TriangleAlertIcon className="size-3" />
            {t("Not included")}
          </p>
          {view.unresolved.map((entry) => (
            <p key={entry.phrase} className="text-xs">
              <span className="block">{entry.phrase}</span>
              <span className="text-muted-foreground block">{entry.reason}</span>
            </p>
          ))}
        </div>
      )}

      <Button size="sm" className="self-start" render={<Link to={view.path} />}>
        <ArrowUpRightIcon className="size-3.5" />
        {t("Open the table")}
      </Button>
    </div>
  );
}
