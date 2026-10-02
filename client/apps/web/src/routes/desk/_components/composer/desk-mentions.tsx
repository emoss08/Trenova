import { apiService } from "@/services/api";
import type {
  AssistantEntityRef,
  MentionCandidateRecord,
  MentionSearchType,
} from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  Fragment,
  useCallback,
  useEffect,
  useRef,
  useState,
  type KeyboardEvent,
  type RefObject,
} from "react";
import { DeskIcon, type DeskIconName } from "../desk-icons";

/** How long typing has to settle before the records are searched. */
const SEARCH_DEBOUNCE_MS = 140;

const TABS: readonly MentionSearchType[] = ["all", "shipment", "customer", "invoice", "worker", "carrier"];

const TAB_LABELS: Record<MentionSearchType, string> = {
  all: "All",
  shipment: "Shipments",
  customer: "Customers",
  invoice: "Invoices",
  worker: "Drivers",
  carrier: "Carriers",
};

/** Which tab a record type sits under: a billing queue item is an invoice in waiting. */
function tabOf(type: string): MentionSearchType {
  if (type === "billing_queue_item") {
    return "invoice";
  }
  return (TABS as readonly string[]).includes(type) ? (type as MentionSearchType) : "all";
}

const TYPE_ICONS: Record<string, DeskIconName> = {
  shipment: "truck",
  customer: "headset",
  invoice: "receipt",
  billing_queue_item: "receipt",
  worker: "route",
  carrier: "shield",
};

export function MentionGlyph({ type, size = 13 }: { type: string; size?: number }) {
  return <DeskIcon name={TYPE_ICONS[type] ?? "link"} size={size} stroke={2} />;
}

/** The @ token at the caret: where it starts in the draft and what follows the @. */
type MentionToken = { start: number; query: string };

function tokenAt(draft: string, caret: number): MentionToken | null {
  const match = /(^|\s)@([^\s@]{0,30})$/.exec(draft.slice(0, caret));
  return match ? { start: caret - match[2].length - 1, query: match[2] } : null;
}

export type DeskMentions = ReturnType<typeof useDeskMentions>;

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
  textareaRef,
  mentions,
  onMentionsChange,
  enabled,
}: {
  value: string;
  onChange: (value: string) => void;
  textareaRef: RefObject<HTMLTextAreaElement | null>;
  mentions: readonly AssistantEntityRef[];
  onMentionsChange: (mentions: AssistantEntityRef[]) => void;
  enabled: boolean;
}) {
  const [token, setToken] = useState<MentionToken | null>(null);
  const [tab, setTab] = useState<MentionSearchType>("all");
  const [highlighted, setHighlighted] = useState(0);
  const [results, setResults] = useState<MentionCandidateRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const query = token?.query ?? null;

  useEffect(() => {
    if (query === null) {
      return;
    }
    const controller = new AbortController();
    const timer = window.setTimeout(() => {
      apiService.assistantService
        .searchMentions(query, tab, { signal: controller.signal })
        .then((found) => {
          setResults(found);
          setHighlighted(0);
          setLoading(false);
        })
        .catch(() => {
          if (!controller.signal.aborted) {
            setResults([]);
            setLoading(false);
          }
        });
    }, SEARCH_DEBOUNCE_MS);
    return () => {
      controller.abort();
      window.clearTimeout(timer);
    };
  }, [query, tab]);

  const detect = (draft: string, caret: number) => {
    if (!enabled) {
      return;
    }
    const next = tokenAt(draft, caret);
    if (next && (!token || token.query !== next.query || token.start !== next.start)) {
      setLoading(true);
    }
    setToken(next);
  };

  const close = useCallback(() => {
    setToken(null);
    setTab("all");
  }, []);

  const changeTab = (next: MentionSearchType) => {
    setLoading(true);
    setTab(next);
  };

  const pick = (record: MentionCandidateRecord | undefined) => {
    if (!token || !record) {
      return;
    }
    const textarea = textareaRef.current;
    const caret = textarea ? textarea.selectionStart : value.length;
    const inserted = `@${record.label} `;
    onChange(value.slice(0, token.start) + inserted + value.slice(caret));
    if (!mentions.some((mention) => mention.type === record.type && mention.id === record.id)) {
      onMentionsChange([...mentions, { type: record.type, id: record.id, label: record.label }]);
    }
    const position = token.start + inserted.length;
    close();
    requestAnimationFrame(() => {
      if (textarea) {
        textarea.focus();
        textarea.setSelectionRange(position, position);
      }
    });
  };

  /** Writes an @ at the caret and opens the search, for the button that offers it. */
  const open = () => {
    const textarea = textareaRef.current;
    const caret = textarea ? textarea.selectionStart : value.length;
    const before = value.slice(0, caret);
    const added = (before && !/\s$/.test(before) ? " " : "") + "@";
    const next = before + added + value.slice(caret);
    onChange(next);
    requestAnimationFrame(() => {
      if (textarea) {
        textarea.focus();
        const position = caret + added.length;
        textarea.setSelectionRange(position, position);
        detect(next, position);
      }
    });
  };

  /** True when the key was the picker's to handle. */
  const onKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>): boolean => {
    if (!token) {
      return false;
    }
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setHighlighted((index) => Math.min(results.length - 1, index + 1));
      return true;
    }
    if (event.key === "ArrowUp") {
      event.preventDefault();
      setHighlighted((index) => Math.max(0, index - 1));
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
    setTab: changeTab,
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
    const current = element.querySelector<HTMLElement>(".dk-on");
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
        <button
          type="button"
          className="dk-mn-ar dk-l"
          tabIndex={-1}
          aria-label={t("Earlier kinds")}
          onClick={() => nudge(-1)}
        >
          <DeskIcon name="chevL" size={11} stroke={2.4} />
        </button>
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
          <button
            key={kind}
            type="button"
            className={tab === kind ? "dk-on" : undefined}
            onClick={() => onChange(kind)}
          >
            {t(TAB_LABELS[kind])}
          </button>
        ))}
      </div>
      {edges.right && (
        <button
          type="button"
          className="dk-mn-ar dk-r"
          tabIndex={-1}
          aria-label={t("More kinds")}
          onClick={() => nudge(1)}
        >
          <DeskIcon name="chevR" size={11} stroke={2.4} />
        </button>
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

/** The list over the box while an @ is being typed. */
export function DeskMentionPicker({ mentions }: { mentions: DeskMentions }) {
  const t = useT();
  if (!mentions.open) {
    return null;
  }
  return (
    <div className="dk-mn" onMouseDown={(event) => event.preventDefault()}>
      <MentionTabs tab={mentions.tab} onChange={mentions.setTab} />
      <div className="dk-mn-l" role="listbox">
        {mentions.loading && (
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
        )}
        {!mentions.loading &&
          mentions.results.map((record, index) => {
            const previous = mentions.results[index - 1];
            const heading =
              mentions.tab === "all" && (!previous || tabOf(previous.type) !== tabOf(record.type));
            return (
              <Fragment key={record.type + record.id}>
                {heading && <div className="dk-mn-h">{t(TAB_LABELS[tabOf(record.type)])}</div>}
                <button
                  type="button"
                  role="option"
                  aria-selected={mentions.highlighted === index}
                  className={cn("dk-mn-r", mentions.highlighted === index && "dk-hi")}
                  onMouseMove={() =>
                    mentions.highlighted !== index && mentions.setHighlighted(index)
                  }
                  onClick={() => mentions.pick(record)}
                >
                  <span className="dk-mn-ic">
                    <MentionGlyph type={record.type} />
                  </span>
                  <span className="dk-mn-t">
                    <b>{record.label}</b>
                    <em>{record.subtitle}</em>
                  </span>
                  {mentions.highlighted === index && <span className="dk-kbd">↵</span>}
                </button>
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

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
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

/** The draft drawn under the textarea with each named record marked, so the @labels read as chips. */
export function DeskMentionMirror({
  value,
  mentions,
}: {
  value: string;
  mentions: readonly AssistantEntityRef[];
}) {
  return (
    <div className="dk-mn-mirror" aria-hidden>
      {splitMentions(value, mentions).map((part, index) =>
        typeof part === "string" ? (
          <Fragment key={index}>{part}</Fragment>
        ) : (
          <mark key={part.at}>{"@" + part.mention.label}</mark>
        ),
      )}
      {"​"}
    </div>
  );
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
