import { NarrativeSegments } from "@/components/narrative-segments";
import type { DetentionNarrativeSegment } from "@/lib/graphql/detention-intelligence";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, formatCurrencyWhole } from "@trenova/shared/lib/utils";
import { createContext, use, useState, type ReactNode } from "react";

/** Whole dollars: the page compares amounts, it does not reconcile them. */
export function money(value: number): string {
  return formatCurrencyWhole(Math.round(value));
}

/** A cell or bar with nothing to show, or a figure the reader cannot see: hatched. */
export const HATCH_BACKGROUND =
  "repeating-linear-gradient(135deg, var(--color-border-subtle) 0 3px, transparent 3px 6px)";

/**
 * A panel that sits beside a fixed-size one: on a wide screen it takes the height its neighbour
 * sets and scrolls inside it, so a longer list never stretches the row and leaves a gap under the
 * neighbour. Its grid cell must be `relative`. Stacked on a narrow screen it has its own height.
 */
export const BESIDE_PANEL = "h-104 xl:absolute xl:inset-0 xl:h-auto";

/**
 * A hairline that grows out from the middle when `shown` and folds back when not: the edge of a
 * list appearing as it starts to scroll, or under a heading as it sticks. Its parent must be
 * positioned; `className` places it (`top-0` or `bottom-0`).
 */
export function GrowingRule({ shown, className }: { shown: boolean; className?: string }) {
  return (
    <span
      aria-hidden
      className={cn(
        "absolute inset-x-0 h-px origin-center bg-border transition-transform duration-300 ease-settle",
        shown ? "scale-x-100" : "scale-x-0",
        className,
      )}
    />
  );
}

export function percent(share: number): string {
  return `${Math.round(share * 100)}%`;
}

const HighlightContext = createContext<(id: string | null) => void>(() => {});

/**
 * One facility lit at a time across every chart: pointing at it in the ranking, the free-time
 * rows or the movers dims the rest everywhere. The dimming is one CSS rule keyed on the marks'
 * `data-facility`, so pointing at a mark re-renders this provider alone, never the hundred-odd
 * marks; the context carries only the setter, which never changes.
 */
export function FacilityHighlightProvider({ children }: { children: ReactNode }) {
  const [hot, setHot] = useState<string | null>(null);
  return (
    <HighlightContext value={setHot}>
      {hot ? (
        // i18n-ignore: a CSS rule, not text a person reads
        <style>{`[data-facility-scope] [data-facility]:not([data-facility="${hot.replace(/["\\]/g, "\\$&")}"]){opacity:.3}`}</style>
      ) : null}
      <div data-facility-scope className="contents">
        {children}
      </div>
    </HighlightContext>
  );
}

/** The attribute that lets the highlight find a facility's mark, and the handlers that light it. */
export function useFacilityHighlight(id: string) {
  const setHot = use(HighlightContext);
  return {
    attributes: { "data-facility": id },
    handlers: {
      onPointerEnter: () => setHot(id),
      onPointerLeave: () => setHot(null),
      onFocus: () => setHot(id),
      onBlur: () => setHot(null),
    },
  };
}

export type NarrativeTargets = {
  onFacility: (locationId: string) => void;
  onCustomer: (customerId: string) => void;
  onSection: (section: string) => void;
};

/**
 * A sentence the server wrote: plain runs, strong figures, and references that open the
 * facility, the customer or the section they name.
 */
export function Narrative({
  segments,
  targets,
}: {
  segments: readonly DetentionNarrativeSegment[];
  targets: NarrativeTargets;
}) {
  return (
    <NarrativeSegments
      segments={segments}
      onTarget={(segment, id) => {
        if (segment.target === "Facility") targets.onFacility(id);
        else if (segment.target === "Customer") targets.onCustomer(id);
        else targets.onSection(id);
      }}
    />
  );
}

/** A section of the page: the question it answers, the answer in one line, then the cards. */
export function IntelligenceSection({
  id,
  title,
  answer,
  action,
  children,
}: {
  id: string;
  title: string;
  answer: ReactNode;
  action?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section id={`detention-${id}`} aria-label={title} className="flex scroll-mt-4 flex-col gap-3">
      <header className="flex flex-wrap items-end justify-between gap-x-4 gap-y-2">
        <div className="min-w-0">
          <h2 className="text-base font-semibold">{title}</h2>
          <p className="mt-0.5 max-w-3xl text-sm text-pretty text-muted-foreground">{answer}</p>
        </div>
        {action ? <div className="shrink-0">{action}</div> : null}
      </header>
      {children}
    </section>
  );
}

export function scrollToSection(section: string): void {
  document
    .getElementById(`detention-${section}`)
    ?.scrollIntoView({ behavior: "smooth", block: "start" });
}

/** A card's opening line: the finding first, the chart after it. */
export function Finding({ children }: { children: ReactNode }) {
  return <p className="mb-3 text-sm text-pretty text-muted-foreground">{children}</p>;
}

export function Strong({ children }: { children: ReactNode }) {
  return <span className="font-medium text-foreground">{children}</span>;
}

export function HoursValue({ minutes }: { minutes: number }) {
  const t = useT();
  return <>{t("{0} h", Math.round(minutes / 60).toLocaleString())}</>;
}
