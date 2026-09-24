import { MarkdownLinkContext, type MarkdownLinkRenderer } from "@/components/elements/ai-markdown";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@trenova/shared/components/ui/hover-card";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatISODateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { ArrowUpRightIcon, ChevronRightIcon, GlobeIcon, LandmarkIcon } from "lucide-react";
import { useCallback, useMemo, useState, type ReactNode } from "react";
import { indexSources, sourceFor, type WebSource } from "./web-sources";

/** How many sites the closed sources line names before it counts the rest. */
const NAMED_SITES = 2;

/** A government page wears a landmark; anything else, a globe. */
function SourceMark({ source, className }: { source: WebSource; className?: string }) {
  const Icon = source.official ? LandmarkIcon : GlobeIcon;

  return <Icon aria-hidden className={cn("text-foreground-subtle size-3 shrink-0", className)} />;
}

/** Where a page stands: from an agency or not, and when it was published. */
function SourceFacts({ source }: { source: WebSource }) {
  const t = useT();
  const published = formatISODateMedium(source.published);
  const facts = [
    source.official ? t("Official source") : "",
    published !== "" ? t("Published {0}", published) : t("No publish date"),
  ].filter((fact) => fact !== "");

  return <>{facts.join(" · ")}</>;
}

/**
 * A link in the answer to a page the agent found, drawn as the site it cites
 * so the reader can see at a glance where a sentence came from. Resting on
 * it shows the page — its title, whether an agency published it, when, and
 * the passage the agent read.
 */
function CitationChip({ source }: { source: WebSource }) {
  const t = useT();

  return (
    <HoverCard>
      <HoverCardTrigger
        delay={150}
        closeDelay={100}
        href={source.url}
        target="_blank"
        rel="noopener noreferrer"
        aria-label={t("{0} on {1}, opens in a new tab", source.title, source.site)}
        className="ui-focus-ring bg-muted text-foreground-muted hover:bg-surface-hover hover:text-foreground mx-0.5 inline-flex h-4.5 max-w-44 items-center gap-1 rounded-full px-1.5 align-middle text-2xs no-underline transition-colors"
      >
        <SourceMark source={source} className="size-2.5" />
        <span className="truncate">{source.site}</span>
      </HoverCardTrigger>
      <HoverCardContent side="top" align="start" className="flex w-72 flex-col gap-1.5">
        <span className="text-foreground-muted flex items-center gap-1.5 text-xs">
          <SourceMark source={source} />
          <span className="truncate">{source.site}</span>
        </span>
        <span className="line-clamp-2 text-sm font-medium">{source.title}</span>
        <span className="text-foreground-subtle text-xs">
          <SourceFacts source={source} />
        </span>
        {source.excerpt !== "" && (
          <span className="text-foreground-muted line-clamp-3 text-xs leading-relaxed">
            {source.excerpt}
          </span>
        )}
      </HoverCardContent>
    </HoverCard>
  );
}

/**
 * Lets the answer's links to pages the agent found draw themselves as
 * citations. A link to anything else stays a plain link: a model can write an
 * address it never searched, and that should not look like a source.
 */
export function CitationProvider({
  sources,
  children,
}: {
  sources: readonly WebSource[];
  children: ReactNode;
}) {
  const index = useMemo(() => indexSources(sources), [sources]);
  const renderLink = useCallback<MarkdownLinkRenderer>(
    (href) => {
      const source = sourceFor(index, href);

      return source ? <CitationChip source={source} /> : null;
    },
    [index],
  );

  if (sources.length === 0) {
    return children;
  }

  return <MarkdownLinkContext value={renderLink}>{children}</MarkdownLinkContext>;
}

function SourceRow({ source }: { source: WebSource }) {
  const t = useT();

  return (
    <li>
      <a
        href={source.url}
        target="_blank"
        rel="noopener noreferrer"
        className="group/source ui-focus-ring hover:bg-surface-hover -mx-2 flex min-w-0 items-start gap-2.5 rounded-control px-2 py-1.5 transition-colors"
      >
        <SourceMark source={source} className="mt-0.5 size-3.5" />
        <span className="flex min-w-0 flex-1 flex-col">
          <span className="truncate text-xs">{source.title}</span>
          <span className="text-foreground-subtle truncate text-2xs">
            {source.site} · <SourceFacts source={source} />
          </span>
        </span>
        <ArrowUpRightIcon
          aria-label={t("Opens in a new tab")}
          className="text-foreground-subtle mt-0.5 size-3 shrink-0 opacity-0 transition-opacity group-hover/source:opacity-100 group-focus-visible/source:opacity-100"
        />
      </a>
    </li>
  );
}

/**
 * Everything the reply drew on from the web, under the answer. Closed, it is
 * one quiet line naming the first sites; open, the pages the answer cites
 * come first and what the search also turned up follows, so the reader can
 * check a claim or see what the agent weighed and left out.
 */
export function SourcesFooter({ sources }: { sources: readonly WebSource[] }) {
  const t = useT();
  const [open, setOpen] = useState(false);

  if (sources.length === 0) {
    return null;
  }

  const cited = sources.filter((source) => source.cited);
  const found = sources.filter((source) => !source.cited);
  const sites = [...new Set(sources.map((source) => source.site))];
  const named = sites.slice(0, NAMED_SITES).join(", ");
  const hidden = sites.length - NAMED_SITES;
  const retrievedOn = formatISODateMedium(sources[0].retrievedOn);

  return (
    <Collapsible open={open} onOpenChange={setOpen} className="min-w-0">
      <CollapsibleTrigger className="text-foreground-muted hover:text-foreground ui-focus-ring -mx-1.5 flex max-w-full min-w-0 items-center gap-1.5 rounded-control px-1.5 py-0.5 text-xs transition-colors">
        <GlobeIcon aria-hidden className="size-3 shrink-0" />
        <span className="shrink-0">
          {t("{0, plural, one {# source} other {# sources}}", sources.length)}
        </span>
        <span className="text-foreground-subtle min-w-0 truncate">
          {hidden > 0 ? `${named} +${hidden}` : named}
        </span>
        <ChevronRightIcon
          aria-hidden
          className={cn("size-3 shrink-0 transition-transform duration-200", open && "rotate-90")}
        />
      </CollapsibleTrigger>
      <CollapsibleContent className="h-(--collapsible-panel-height) overflow-hidden transition-[height] duration-200 ease-settle data-ending-style:h-0 data-starting-style:h-0">
        <div className="flex flex-col gap-2 pt-1.5 pb-1 pl-4.5">
          {cited.length > 0 && <WebSourceList sources={cited} />}
          {found.length > 0 && (
            <WebSourceList label={cited.length > 0 ? t("Also found") : undefined} sources={found} />
          )}
          {retrievedOn !== "" && (
            <p className="text-foreground-subtle text-2xs">
              {t("Retrieved from the web on {0}", retrievedOn)}
            </p>
          )}
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}

/** Pages as rows that open them: the mark, the title, the site and its facts. */
export function WebSourceList({
  label,
  sources,
}: {
  label?: string;
  sources: readonly WebSource[];
}) {
  return (
    <div className="flex flex-col gap-0.5">
      {label && <span className="text-foreground-subtle text-2xs">{label}</span>}
      <ul className="flex flex-col">
        {sources.map((source) => (
          <SourceRow key={source.url} source={source} />
        ))}
      </ul>
    </div>
  );
}
