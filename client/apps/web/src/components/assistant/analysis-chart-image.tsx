import { agentExtension } from "@/lib/queries/agent-extension";
import { useQuery } from "@tanstack/react-query";
import { useObjectUrl } from "@trenova/shared/hooks/use-object-url";
import { useT } from "@trenova/shared/i18n/use-t";

/**
 * One chart an analysis drew, read from the run under the session's own
 * credentials. The server sends it only to the person the analysis ran for
 * and to those who oversee agents.
 */
export function AnalysisChartImage({
  runId,
  position,
  title,
}: {
  runId: string;
  position: number;
  title: string;
}) {
  const t = useT();
  const image = useQuery({
    ...agentExtension.analysisChart(runId, position),
    staleTime: Number.POSITIVE_INFINITY,
    retry: 1,
  });
  const url = useObjectUrl(image.data ?? null);
  const label = title || t("Chart {0}", position);

  if (image.isError) {
    return (
      <p className="text-muted-foreground rounded-md border p-3 text-xs">
        {t("{0} could not be loaded.", label)}
      </p>
    );
  }

  return (
    <figure className="flex flex-col gap-1">
      {url ? (
        <img
          src={url}
          alt={label}
          className="bg-background w-full rounded-md border object-contain"
          loading="lazy"
        />
      ) : (
        <div
          aria-busy="true"
          aria-label={t("Loading {0}", label)}
          className="ui-shimmer aspect-[4/3] w-full rounded-md"
        />
      )}
      {title !== "" && <figcaption className="text-muted-foreground text-xs">{title}</figcaption>}
    </figure>
  );
}
