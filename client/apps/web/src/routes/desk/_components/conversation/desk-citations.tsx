import { describeActivity, groupActivity, type ToolStep } from "@/components/assistant/activity";
import { ArtifactKindIcon } from "@/components/assistant/voice/artifact-chrome";
import { MarkdownLink } from "@/components/elements/ai-markdown";
import type { AssistantArtifact } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo, useState, type ComponentProps } from "react";
import type { Components } from "react-markdown";
import { DeskIcon } from "../desk-icons";
import { CITATION_HREF, type Citation } from "./citations";

function stepDuration(step: ToolStep): string {
  if (step.durationSeconds === null) {
    return "";
  }
  return step.durationSeconds < 10
    ? `${step.durationSeconds.toFixed(1)}s`
    : `${Math.round(step.durationSeconds)}s`;
}

/**
 * A step's number in a reply and, on hover, what the step did: its number,
 * the tool, how long it took, the step in a sentence, what it was asked and
 * what it came back with, and the artifact it made when it made one. The
 * number opens that artifact.
 */
function CitationNumber({
  citation,
  artifact,
  onOpenArtifact,
}: {
  citation: Citation;
  artifact: Pick<AssistantArtifact, "id" | "kind" | "title"> | null;
  onOpenArtifact: (id: string) => void;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const line = useMemo(() => {
    const group = groupActivity([citation.step])[0];
    return group ? describeActivity(group, t) : null;
  }, [citation.step, t]);

  return (
    <span
      className="dk-fnw"
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
      onFocus={() => setOpen(true)}
      onBlur={() => setOpen(false)}
    >
      <button
        type="button"
        className={cn("dk-fn", open && "dk-hot")}
        aria-label={t("Step {0}: {1}", citation.n, line?.phrase ?? citation.step.name)}
        onClick={() => artifact && onOpenArtifact(artifact.id)}
      >
        {citation.n}
      </button>
      {open && (
        <span className="dk-fnp" role="tooltip">
          <span className="dk-fnp-in">
            <span className="dk-fnp-h">
              <span>{String(citation.n).padStart(2, "0")}</span>
              <span>{citation.step.name}</span>
              <span className="dk-t">{stepDuration(citation.step)}</span>
            </span>
            <span className="dk-fnp-b">{line?.phrase ?? citation.step.name}</span>
            {(line?.detail || citation.step.summary) && (
              <span className="dk-fnp-q">
                {line?.detail && <span>{line.detail}</span>}
                {line?.detail && citation.step.summary && <DeskIcon name="chevR" size={10} />}
                {citation.step.summary && <b>{citation.step.summary}</b>}
              </span>
            )}
            {artifact && (
              <button
                type="button"
                className="dk-abadge dk-sm"
                onClick={() => onOpenArtifact(artifact.id)}
              >
                <ArtifactKindIcon kind={artifact.kind} className="dk-abadge-i" />
                {artifact.title}
              </button>
            )}
          </span>
        </span>
      )}
    </span>
  );
}

/**
 * The reply's markdown links, with the citation links drawn as step numbers
 * and every other link drawn as the app draws links.
 */
export function useCitationOverrides(
  citations: readonly Citation[],
  artifacts: readonly Pick<AssistantArtifact, "id" | "kind" | "title" | "sourceToolCallId">[],
  onOpenArtifact: (id: string) => void,
): Components | undefined {
  return useMemo(() => {
    if (citations.length === 0) {
      return undefined;
    }
    const byNumber = new Map(citations.map((citation) => [citation.n, citation]));

    return {
      a: ({ href, children, ...rest }: ComponentProps<"a">) => {
        if (href?.startsWith(CITATION_HREF)) {
          const citation = byNumber.get(Number(href.slice(CITATION_HREF.length)));
          if (citation) {
            const artifact =
              artifacts.find((candidate) => candidate.sourceToolCallId === citation.step.id) ??
              null;
            return (
              <CitationNumber
                citation={citation}
                artifact={artifact}
                onOpenArtifact={onOpenArtifact}
              />
            );
          }
        }
        return (
          <MarkdownLink href={href} {...rest}>
            {children}
          </MarkdownLink>
        );
      },
    };
  }, [artifacts, citations, onOpenArtifact]);
}
