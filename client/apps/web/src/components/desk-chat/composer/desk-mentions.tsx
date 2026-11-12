import { escapeRegExp } from "@trenova/shared/lib/regexp";
import { mentionToken } from "@/components/assistant/composer-types";
import { assistant } from "@/lib/queries/assistant";
import { useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import type { DeskEditorHandle } from "./editor/desk-editor";
import { useMenuSide } from "./use-menu-side";
import {
  isMentionKind,
  mentionSearchTypes,
  type AssistantEntityRef,
  type MentionCandidateRecord,
  type MentionSearchType,
} from "@/types/assistant";
import { MentionKindTab, type MentionKind } from "@trenova/shared/types/generated/record-kinds";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Fragment, useCallback, useEffect, useRef, useState, type RefObject } from "react";
import { DeskIcon, type DeskIconName } from "../desk-icons";
import { defineLabels } from "@trenova/shared/i18n/labels";

/** How long typing has to settle before the records are searched. */
const SEARCH_DEBOUNCE_MS = 140;

const TABS = mentionSearchTypes;

const TAB_LABELS: Record<MentionSearchType, string> = defineLabels({
  all: "All",
  shipment: "Shipments",
  customer: "Customers",
  invoice: "Invoices",
  worker: "Drivers",
  carrier: "Carriers",
  dispute: "Disputes",
  location: "Locations",
});

/** Which tab a record type sits under, as the server files it: a billing queue item is an invoice in waiting. */
function tabOf(type: string): MentionSearchType {
  return isMentionKind(type) ? MentionKindTab[type] : "all";
}

const TYPE_ICONS: Record<MentionKind, DeskIconName> = {
  shipment: "truck",
  customer: "headset",
  invoice: "receipt",
  billing_queue_item: "receipt",
  invoice_dispute: "receipt",
  worker: "route",
  carrier: "shield",
  location: "compass",
};

export function MentionGlyph({ type, size = 13 }: { type: string; size?: number }) {
  return <DeskIcon name={isMentionKind(type) ? TYPE_ICONS[type] : "link"} size={size} stroke={2} />;
}

/** How many rows Page Up and Page Down move the highlight in a composer menu. */
const PAGE_ROWS = 5;

/**
 * Where a key moves the highlight in a composer menu of `last + 1` rows, or
 * null when the key is not one that moves it. The arrows wrap; Home, End and
 * the page keys do not.
 */
export function listMove(key: string, at: number, last: number): number | null {
  if (last < 0) {
    return null;
  }
  switch (key) {
    case "ArrowDown":
      return at >= last ? 0 : at + 1;
    case "ArrowUp":
      return at <= 0 ? last : at - 1;
    case "PageDown":
      return Math.min(last, at + PAGE_ROWS);
    case "PageUp":
      return Math.max(0, at - PAGE_ROWS);
    case "Home":
      return 0;
    case "End":
      return last;
    default:
      return null;
  }
}

/** The @ token at the caret: where it starts in the draft and what follows the @. */
type MentionToken = { start: number; query: string };

function tokenAt(draft: string, caret: number): MentionToken | null {
  const match = /(^|\s)@([^\s@]{0,30})$/.exec(draft.slice(0, caret));
  return match ? { start: caret - match[2].length - 1, query: match[2] } : null;
}

export type DeskMentions = ReturnType<typeof useDeskMentions>;

/** How long an answered search is shown again without asking. */
export const MENTION_SEARCH_STALE_MS = 30_000;

const NO_RECORDS: MentionCandidateRecord[] = [];

/**
 * Reads what a bare @ lists before anyone types one, so the picker opens on
 * records and not on a skeleton. Called when the box takes the focus.
 */
export function prefetchMentions(client: QueryClient): void {
  void client.prefetchQuery({
    ...assistant.mentionSearch("", "all"),
    staleTime: MENTION_SEARCH_STALE_MS,
  });
}

/**
 * The records the @ search finds for what is typed, and which of them the
 * keyboard is on. A null query searches nothing. Words are searched once
 * typing settles, except words already answered, which are shown at once;
 * the list is `loading` from the moment the words change until their own
 * answer is in, so it never shows the last search's records as this one's.
 */
export function useMentionSearch(query: string | null, tab: MentionSearchType) {
  const client = useQueryClient();
  const [highlighted, setHighlighted] = useState(0);
  const [shown, setShown] = useState("");
  const settled = useDebounce(query, SEARCH_DEBOUNCE_MS);
  const answered =
    query !== null &&
    client.getQueryData(assistant.mentionSearch(query, tab).queryKey) !== undefined;
  const asked = answered ? query : settled;
  const search = useQuery({
    ...assistant.mentionSearch(asked ?? "", tab),
    enabled: query !== null && asked !== null,
    staleTime: MENTION_SEARCH_STALE_MS,
  });
  const current = query !== null && asked === query;
  const results = (current ? search.data : undefined) ?? NO_RECORDS;
  const loading = query !== null && (!current || search.isPending);

  const key = query === null ? "" : tab + "\n" + query;
  if (key !== shown) {
    setShown(key);
    setHighlighted(0);
  }

  return { results, loading, highlighted, setHighlighted };
}

/**
 * Naming a record with @. Typing @ opens a search over the records the person
 * may read, narrowed by kind with the tabs along its top; the record picked
 * is written into the draft as @label and rides with the message as a record
 * the agent can look up. A record whose @label is deleted from the draft is
 * dropped when the message leaves.
 */
export function useDeskMentions({
  value,
  onChange,
  editorRef,
  mentions,
  onMentionsChange,
  enabled,
}: {
  value: string;
  onChange: (value: string) => void;
  editorRef: RefObject<DeskEditorHandle | null>;
  mentions: readonly AssistantEntityRef[];
  onMentionsChange: (mentions: AssistantEntityRef[]) => void;
  enabled: boolean;
}) {
  const [token, setToken] = useState<MentionToken | null>(null);
  const [tab, setTab] = useState<MentionSearchType>("all");
  const query = token?.query ?? null;
  const { results, loading, highlighted, setHighlighted } = useMentionSearch(query, tab);

  const detect = (draft: string, caret: number) => {
    if (!enabled) {
      return;
    }
    const next = tokenAt(draft, caret);
    if (next?.start !== token?.start || next?.query !== token?.query) {
      setToken(next);
    }
  };

  const close = useCallback(() => {
    setToken(null);
    setTab("all");
  }, []);

  /** The tab beside the one open, wrapping at either end. */
  const stepTab = (by: 1 | -1) => {
    const at = TABS.indexOf(tab);
    setTab(TABS[(at + by + TABS.length) % TABS.length]);
  };

  const pick = (record: MentionCandidateRecord | undefined) => {
    if (!token || !record) {
      return;
    }
    const editor = editorRef.current;
    const caret = editor ? editor.caret() : value.length;
    const inserted = `${mentionToken(record.label)} `;
    onChange(value.slice(0, token.start) + inserted + value.slice(caret));
    if (!mentions.some((mention) => mention.type === record.type && mention.id === record.id)) {
      onMentionsChange([...mentions, { type: record.type, id: record.id, label: record.label }]);
    }
    const position = token.start + inserted.length;
    close();
    requestAnimationFrame(() => editor?.setCaret(position));
  };

  /** Writes an @ at the caret and opens the search, for the button that offers it. */
  const open = () => {
    const editor = editorRef.current;
    const caret = editor ? editor.caret() : value.length;
    const before = value.slice(0, caret);
    const added = (before && !/\s$/.test(before) ? " " : "") + "@";
    const next = before + added + value.slice(caret);
    onChange(next);
    requestAnimationFrame(() => {
      const position = caret + added.length;
      editor?.setCaret(position);
      detect(next, position);
    });
  };

  /** True when the key was the picker's to handle. */
  const onKeyDown = (event: KeyboardEvent): boolean => {
    if (!token || event.isComposing) {
      return false;
    }
    const last = results.length - 1;
    const move = listMove(event.key, highlighted, last);
    if (move !== null) {
      event.preventDefault();
      setHighlighted(move);
      return true;
    }
    // With nothing typed after the @ there is no word for the caret to move
    // through, so the arrows walk the kinds instead.
    if (token.query === "" && (event.key === "ArrowLeft" || event.key === "ArrowRight")) {
      event.preventDefault();
      stepTab(event.key === "ArrowRight" ? 1 : -1);
      return true;
    }
    if ((event.key === "Enter" || event.key === "Tab") && results[highlighted] && !loading) {
      event.preventDefault();
      pick(results[highlighted]);
      return true;
    }
    if (event.key === "Escape") {
      event.preventDefault();
      close();
      return true;
    }
    return false;
  };

  return {
    open: token !== null,
    query: token?.query ?? "",
    tab,
    setTab,
    highlighted,
    setHighlighted,
    results,
    loading,
    detect,
    pick,
    openPicker: open,
    close,
    onKeyDown,
  };
}

/** The round arrow at either end of the kinds, over the faded edge. */
const arrowClass =
  "absolute top-1/2 z-2 size-5.5 -translate-y-1/2 justify-center rounded-full bg-dsk-raised text-dsk-fg2 ring-1 ring-dsk-b hover:text-dsk-fg";

/** The kinds along the top of the picker, scrolled with arrows when they do not fit. */
function MentionTabs({
  tab,
  onChange,
}: {
  tab: MentionSearchType;
  onChange: (tab: MentionSearchType) => void;
}) {
  const t = useT();
  const ref = useRef<HTMLDivElement>(null);
  const [edges, setEdges] = useState({ left: false, right: false });
  const measure = useCallback(() => {
    const element = ref.current;
    if (!element) {
      return;
    }
    setEdges({
      left: element.scrollLeft > 2,
      right: element.scrollLeft + element.clientWidth < element.scrollWidth - 2,
    });
  }, []);

  useEffect(() => {
    const element = ref.current;
    if (!element) {
      return;
    }
    const frame = requestAnimationFrame(measure);
    const current = element.querySelector<HTMLElement>("[data-on]");
    if (current) {
      const left = current.offsetLeft - 8;
      const right = current.offsetLeft + current.offsetWidth + 8;
      if (left < element.scrollLeft) {
        element.scrollTo({ left, behavior: "smooth" });
      } else if (right > element.scrollLeft + element.clientWidth) {
        element.scrollTo({ left: right - element.clientWidth, behavior: "smooth" });
      }
    }
    return () => cancelAnimationFrame(frame);
  }, [tab, measure]);

  const nudge = (direction: number) =>
    ref.current?.scrollBy({ left: direction * 120, behavior: "smooth" });

  return (
    <div className={cn("dk-mn-fw", edges.left && "dk-l", edges.right && "dk-r")}>
      {edges.left && (
        <Button
          variant="bare"
          size="bare"
          className={cn(arrowClass, "left-1")}
          tabIndex={-1}
          aria-label={t("Earlier kinds")}
          onClick={() => nudge(-1)}
        >
          <DeskIcon name="chevL" size={11} stroke={2.4} />
        </Button>
      )}
      <div
        className="dk-mn-f"
        ref={ref}
        onScroll={measure}
        onWheel={(event) => {
          if (ref.current && Math.abs(event.deltaY) > Math.abs(event.deltaX)) {
            ref.current.scrollLeft += event.deltaY;
          }
        }}
      >
        {TABS.map((kind) => (
          <Button
            key={kind}
            variant="bare"
            size="bare"
            data-on={tab === kind || undefined}
            className={cn(
              "h-6 flex-none rounded-badge px-2.25 text-xs text-dsk-muted transition-colors duration-100 hover:bg-dsk-hover hover:text-dsk-fg",
              tab === kind && "bg-dsk-ink text-dsk-ink-fg hover:bg-dsk-ink hover:text-dsk-ink-fg",
            )}
            onClick={() => onChange(kind)}
          >
            {TAB_LABELS[kind]}
          </Button>
        ))}
      </div>
      {edges.right && (
        <Button
          variant="bare"
          size="bare"
          className={cn(arrowClass, "right-1")}
          tabIndex={-1}
          aria-label={t("More kinds")}
          onClick={() => nudge(1)}
        >
          <DeskIcon name="chevR" size={11} stroke={2.4} />
        </Button>
      )}
    </div>
  );
}

const SKELETON_WIDTHS = [
  [46, 70],
  [58, 62],
  [38, 76],
  [52, 55],
] as const;

/** The rows a record search shows while it reads, drawn in the shape of the records to come. */
export function DeskMentionLoading() {
  return (
    <div className="dk-mn-sk" aria-busy>
      {SKELETON_WIDTHS.map(([title, sub], index) => (
        <div key={index} className="dk-mn-skr" style={{ animationDelay: `${index * 80}ms` }}>
          <i className="dk-a" />
          <span>
            <i className="dk-b" style={{ width: `${title}%` }} />
            <i className="dk-c" style={{ width: `${sub}%` }} />
          </span>
        </div>
      ))}
    </div>
  );
}

/** One record a search found: its kind's glyph, what it is called and what tells it apart. */
export function DeskMentionOption({
  id,
  record,
  highlighted,
  onHighlight,
  onPick,
}: {
  /** Set where an input names the option it is on with aria-activedescendant. */
  id?: string;
  record: MentionCandidateRecord;
  highlighted: boolean;
  onHighlight: () => void;
  onPick: () => void;
}) {
  return (
    <Button
      id={id}
      variant="bare"
      size="bare"
      role="option"
      aria-selected={highlighted}
      className={cn(
        "flex h-10 w-full gap-2.5 rounded-lg px-2 text-left",
        highlighted && "bg-dsk-hover",
      )}
      onMouseMove={() => !highlighted && onHighlight()}
      onClick={onPick}
    >
      <span className="dk-mn-ic">
        <MentionGlyph type={record.type} />
      </span>
      <span className="dk-mn-t">
        <b>{record.label}</b>
        <em>{record.subtitle}</em>
      </span>
      {highlighted && <span className="dk-kbd">↵</span>}
    </Button>
  );
}

/** The list over the box while an @ is being typed. */
export function DeskMentionPicker({ mentions }: { mentions: DeskMentions }) {
  const t = useT();
  const { attach, below } = useMenuSide();
  if (!mentions.open) {
    return null;
  }
  return (
    <div
      ref={attach}
      className={cn("dk-mn", below && "dk-menu-below")}
      onMouseDown={(event) => event.preventDefault()}
    >
      <MentionTabs tab={mentions.tab} onChange={mentions.setTab} />
      <div className="dk-mn-l" role="listbox">
        {mentions.loading && <DeskMentionLoading />}
        {!mentions.loading &&
          mentions.results.map((record, index) => {
            const previous = mentions.results[index - 1];
            const heading =
              mentions.tab === "all" && (!previous || tabOf(previous.type) !== tabOf(record.type));
            return (
              <Fragment key={record.type + record.id}>
                {heading && <div className="dk-mn-h">{TAB_LABELS[tabOf(record.type)]}</div>}
                <DeskMentionOption
                  record={record}
                  highlighted={mentions.highlighted === index}
                  onHighlight={() => mentions.setHighlighted(index)}
                  onPick={() => mentions.pick(record)}
                />
              </Fragment>
            );
          })}
        {!mentions.loading && mentions.results.length === 0 && (
          <div className="dk-mn-empty">
            {mentions.query
              ? t("No records match “{0}”", mentions.query)
              : t("Start typing a load, customer, invoice or driver")}
          </div>
        )}
      </div>
      <div className="dk-mn-ft">
        <span>
          <span className="dk-kbd">↑</span>
          <span className="dk-kbd">↓</span>
          {t("move")}
        </span>
        <span>
          <span className="dk-kbd">↵</span>
          {t("insert")}
        </span>
        <span>
          <span className="dk-kbd">Esc</span>
          {t("close")}
        </span>
      </div>
    </div>
  );
}

/** Splits text at each named record's @label. */
function splitMentions(
  text: string,
  mentions: readonly AssistantEntityRef[],
): Array<string | { at: number; mention: AssistantEntityRef }> {
  const named = mentions.filter((mention) => mention.label !== "");
  if (named.length === 0) {
    return [text];
  }
  const pattern = new RegExp(
    "@(" +
      named
        .map((mention) => escapeRegExp(mention.label))
        .sort((a, b) => b.length - a.length)
        .join("|") +
      ")",
    "g",
  );
  const parts: Array<string | { at: number; mention: AssistantEntityRef }> = [];
  let at = 0;
  for (const match of text.matchAll(pattern)) {
    const index = match.index ?? 0;
    if (index > at) {
      parts.push(text.slice(at, index));
    }
    const mention = named.find((candidate) => candidate.label === match[1]);
    if (mention) {
      parts.push({ at: index, mention });
    }
    at = index + match[0].length;
  }
  parts.push(text.slice(at));
  return parts;
}

/** A sent question with each named record as a chip in place of its @label. */
export function DeskMentionText({
  text,
  mentions,
}: {
  text: string;
  mentions: readonly AssistantEntityRef[];
}) {
  return (
    <>
      {splitMentions(text, mentions).map((part, index) =>
        typeof part === "string" ? (
          <Fragment key={index}>{part}</Fragment>
        ) : (
          <span key={part.at} className="dk-mn-chip">
            <MentionGlyph type={part.mention.type} size={12} />
            {part.mention.label}
          </span>
        ),
      )}
    </>
  );
}
