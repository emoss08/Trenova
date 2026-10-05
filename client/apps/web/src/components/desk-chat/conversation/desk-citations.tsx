import { describeActivity, groupActivity, type ToolStep } from "@/components/assistant/activity";
import { ArtifactKindIcon } from "@/components/assistant/voice/artifact-chrome";
import { MarkdownLink } from "@/components/elements/ai-markdown";
import type { AssistantArtifact, StepRationale } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ComponentProps,
  type ReactNode,
} from "react";
import { createPortal } from "react-dom";
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
/** In a compact column, how far a popover keeps inside the scroll area, and the room it wants above. */
const DENSE_EDGE = 10;
const DENSE_ROOM = 150;

/** Where a mark's popover hangs: the mark's top centre, or its bottom centre when there is no room above. */
export type PopoverPlace = {
  below: boolean;
  /** How far the popover is nudged sideways to stay on screen. */
  shift: number;
  /** The point it hangs from, in the viewport. */
  x: number;
  y: number;
  /** The surface it is drawn in, so it keeps that surface's tokens. */
  root: Element;
};

/** The frame a popover stays inside, in viewport coordinates, and the room it wants above its mark. */
export type PopoverBounds = { left: number; right: number; top: number; room: number };

/**
 * Whether a popover opens below its mark, and how far it is nudged sideways,
 * so all of it stays inside `bounds`. A popover wider than the frame is
 * centred in it rather than pushed off one side to keep the other.
 */
export function popoverPlacement(
  anchor: { left: number; top: number; width: number },
  width: number,
  bounds: PopoverBounds,
): { below: boolean; shift: number } {
  const center = anchor.left + anchor.width / 2;
  const half = width / 2;
  let shift = 0;
  if (bounds.right - bounds.left < width) {
    shift = (bounds.left + bounds.right) / 2 - center;
  } else if (center - half < bounds.left) {
    shift = bounds.left - (center - half);
  } else if (center + half > bounds.right) {
    shift = bounds.right - (center + half);
  }

  return { below: anchor.top - bounds.top < bounds.room, shift };
}

/**
 * Where a popover goes so all of it stays in view: inside the window at the
 * Desk, and inside the conversation's scroll area in a compact column, where
 * the window is far wider than what can be seen of the thread.
 */
export function placePopover(anchor: HTMLElement, width: number = POPOVER_WIDTH): PopoverPlace {
  const rect = anchor.getBoundingClientRect();
  const scroller = anchor.closest(".dk-dense")?.querySelector(".dk-scroll") ?? null;
  const frame = scroller?.contains(anchor) ? scroller.getBoundingClientRect() : null;
  const bounds: PopoverBounds = frame
    ? {
        left: frame.left + DENSE_EDGE,
        right: frame.right - DENSE_EDGE,
        top: frame.top,
        room: DENSE_ROOM,
      }
    : { left: EDGE, right: window.innerWidth - EDGE, top: 0, room: POPOVER_ROOM };
  const { below, shift } = popoverPlacement(rect, width, bounds);

  return {
    below,
    shift,
    x: rect.left + rect.width / 2,
    y: below ? rect.bottom : rect.top,
    root: anchor.closest(".dsk, .dk-chat") ?? document.body,
  };
}

/**
 * A mark's popover, open while the mark is rested on or focused. It closes
 * when the page scrolls, since it no longer hangs from the mark.
 */
export function useMarkPopover(width: number = POPOVER_WIDTH) {
  const [place, setPlace] = useState<PopoverPlace | null>(null);
  const anchorRef = useRef<HTMLSpanElement>(null);
  const show = useCallback(() => {
    if (anchorRef.current) setPlace(placePopover(anchorRef.current, width));
  }, [width]);
  const hide = useCallback(() => setPlace(null), []);
  const open = place !== null;
  useEffect(() => {
    if (!open) return;
    window.addEventListener("scroll", hide, true);
    return () => window.removeEventListener("scroll", hide, true);
  }, [hide, open]);

  return { anchorRef, place, show, hide };
}

/**
 * Hangs a mark's popover outside the reply's text. Drawn inside the text, an
 * open popover changes where the line may break, so the mark can wrap away
 * from the pointer that opened it, close, wrap back and open again.
 */
export function MarkPopoverLayer({
  place,
  children,
}: {
  place: PopoverPlace;
  children: ReactNode;
}) {
  return createPortal(
    <span className="dk-pop-at" style={{ left: place.x, top: place.y }}>
      {children}
    </span>,
    place.root,
  );
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
      {citation.step.why && <StepWhyToggle why={citation.step.why} />}
    </>
  );
}

/**
 * "Why this step?": the agent's own account of the step, as it gave it with
 * the call. A step it gave none for offers nothing rather than a made-up
 * reason.
 */
function StepWhyToggle({ why }: { why: StepRationale }) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const rows: [string, string][] = [
    [t("Saw"), why.saw],
    [t("Because"), why.because],
    [t("Instead of"), why.insteadOf],
  ];
  if (rows.every(([, text]) => text === "")) return null;

  return (
    <>
      <button
        type="button"
        className={cn("dk-fnp-why", open && "dk-on")}
        aria-expanded={open}
        onClick={(event) => {
          event.stopPropagation();
          setOpen((value) => !value);
        }}
      >
        <DeskIcon name="why" size={12} />
        {open ? t("Hide reasoning") : t("Why this step?")}
      </button>
      {open && <StepWhy rows={rows} />}
    </>
  );
}

export function StepWhy({ rows }: { rows: [string, string][] }) {
  return (
    <span className="dk-why">
      {rows
        .filter(([, text]) => text !== "")
        .map(([label, text]) => (
          <span key={label}>
            <em>{label}</em>
            {text}
          </span>
        ))}
    </span>
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
  const { anchorRef, place, show, hide } = useMarkPopover();
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
        <MarkPopoverLayer place={place}>
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
        </MarkPopoverLayer>
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
