import { describeActivity, groupActivity, type ToolStep } from "@/components/assistant/activity";
import { ArtifactKindIcon } from "@/components/assistant/voice/artifact-chrome";
import { MarkdownLink } from "@/components/elements/ai-markdown";
import type { AssistantArtifact } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo, useRef, useState, type ComponentProps } from "react";
import type { Components } from "react-markdown";
import { DeskIcon } from "../desk-icons";
import { CITATION_HREF, groupCitations, type Citation, type CitationGroup } from "./citations";

function stepDuration(step: ToolStep): string {
  if (step.durationSeconds === null) {
    return "";
  }
  return step.durationSeconds < 10
    ? `${step.durationSeconds.toFixed(1)}s`
    : `${Math.round(step.durationSeconds)}s`;
}

/** How far a popover keeps from the window's edges. */
const EDGE = 12;
/** Room a popover wants above its mark before it opens below instead. */
const POPOVER_ROOM = 230;
const POPOVER_WIDTH = 250;

/** Where a popover goes so all of it stays on screen: below when there is no room above, and nudged in from the sides. */
function placePopover(anchor: HTMLElement): { below: boolean; shift: number } {
  const rect = anchor.getBoundingClientRect();
  const center = rect.left + rect.width / 2;
  const half = POPOVER_WIDTH / 2;
  let shift = 0;
  if (center - half < EDGE) shift = EDGE - (center - half);
  else if (center + half > window.innerWidth - EDGE)
    shift = window.innerWidth - EDGE - (center + half);
  return { below: rect.top < POPOVER_ROOM, shift };
}

/** One step as a citation's popover describes it. */
function StepSummary({
  citation,
  artifact,
  onOpenArtifact,
}: {
  citation: Citation;
  artifact: Pick<AssistantArtifact, "id" | "kind" | "title"> | null;
  onOpenArtifact: (id: string) => void;
}) {
  const t = useT();
  const line = useMemo(() => {
    const group = groupActivity([citation.step])[0];
    return group ? describeActivity(group, t) : null;
  }, [citation.step, t]);

  return (
    <>
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
    </>
  );
}

/**
 * A reply's mark for the steps behind a phrase: one number, or a range when
 * several steps back the same words. On hover it says what each step did,
 * and opens where it fits on screen. The mark opens the artifact the first
 * step made, when it made one.
 */
function CitationNumber({
  group,
  artifactOf,
  onOpenArtifact,
}: {
  group: CitationGroup;
  artifactOf: (citation: Citation) => Pick<AssistantArtifact, "id" | "kind" | "title"> | null;
  onOpenArtifact: (id: string) => void;
}) {
  const t = useT();
  const [place, setPlace] = useState<{ below: boolean; shift: number } | null>(null);
  const anchorRef = useRef<HTMLSpanElement>(null);
  const show = () => {
    if (anchorRef.current) setPlace(placePopover(anchorRef.current));
  };
  const hide = () => setPlace(null);
  const first = group.citations[0];
  const artifact = artifactOf(first);
  const shown = group.citations.slice(0, MAX_LISTED);
  const more = group.citations.length - shown.length;

  return (
    <span
      ref={anchorRef}
      className="dk-fnw"
      onMouseEnter={show}
      onMouseLeave={hide}
      onFocus={show}
      onBlur={hide}
    >
      <button
        type="button"
        className={cn("dk-fn", place && "dk-hot")}
        aria-label={
          group.citations.length > 1
            ? t("Steps {0}", group.label)
            : t("Step {0}: {1}", first.n, first.step.name)
        }
        onClick={() => artifact && onOpenArtifact(artifact.id)}
      >
        {group.label}
      </button>
      {place && (
        <span
          className={cn("dk-fnp", place.below && "dk-below")}
          role="tooltip"
          style={place.shift ? { left: `calc(50% + ${place.shift}px)` } : undefined}
        >
          <span className={cn("dk-fnp-in", group.citations.length > 1 && "dk-many")}>
            {shown.map((citation, index) => (
              <span key={citation.step.id} className="dk-fnp-step">
                {index > 0 && <span className="dk-fnp-sep" />}
                <StepSummary
                  citation={citation}
                  artifact={artifactOf(citation)}
                  onOpenArtifact={onOpenArtifact}
                />
              </span>
            ))}
            {more > 0 && (
              <span className="dk-fnp-more">
                {t("{0, plural, one {and # more step} other {and # more steps}}", more)}
              </span>
            )}
          </span>
        </span>
      )}
    </span>
  );
}

/** The most steps one citation's popover lists before it counts the rest. */
const MAX_LISTED = 4;

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
    const byNumber = new Map(
      groupCitations(citations).map((group) => [group.citations[0].n, group]),
    );
    const artifactOf = (citation: Citation) =>
      artifacts.find((candidate) => candidate.sourceToolCallId === citation.step.id) ?? null;

    return {
      a: ({ href, children, ...rest }: ComponentProps<"a">) => {
        if (href?.startsWith(CITATION_HREF)) {
          const group = byNumber.get(Number(href.slice(CITATION_HREF.length)));
          if (group) {
            return (
              <CitationNumber
                group={group}
                artifactOf={artifactOf}
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
