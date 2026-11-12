import type { AssistantEntityRef } from "@/types/assistant";
import { useSyncExternalStore } from "react";

/**
 * What a person has typed and not sent, kept per place they typed it: one
 * draft a conversation and one for each space that starts conversations (the
 * Desk's front page, a record's case, a page's assistant). A draft is the
 * words and the records named in them with @, so a restored draft still
 * carries its records.
 *
 * Typing never waits on the disk: a draft is changed in memory at once and
 * written to localStorage a moment after typing pauses, and when the page is
 * hidden or closed. Another tab's writes are read as they land, so a draft
 * started in one tab is there in the next. `clear` forgets every draft, in
 * memory and on disk, and is called wherever a session ends.
 */
export type ComposerDraft = {
  text: string;
  mentions: readonly AssistantEntityRef[];
  /** When it was last changed, in milliseconds, so the oldest give way first. */
  at: number;
};

export const DRAFT_STORAGE_KEY = "trenova-composer-drafts";
/** The most drafts kept; past it the one left alone longest is dropped. */
export const MAX_DRAFTS = 30;
/** The longest draft kept on disk. A longer one stays in memory for the visit. */
export const MAX_DRAFT_LENGTH = 20_000;
/** How long typing has to pause before the drafts are written to disk. */
export const DRAFT_WRITE_MS = 250;

const NO_MENTIONS: readonly AssistantEntityRef[] = [];
const STORAGE_VERSION = 1;

/** The key of a draft: the conversation it is in, or the space it would start one from. */
export const draftKey = {
  thread: (threadId: string) => `thread:${threadId}`,
  space: (space: string) => `space:${space}`,
};

/**
 * The space of a record: its case and the margin beside its page are the
 * same place to ask about it, so a question begun in one is there in the
 * other.
 */
export function recordSpace(kind: string, id: string): string {
  return `record:${kind}:${id}`;
}

type Listener = () => void;

function isMention(value: unknown): value is AssistantEntityRef {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const ref = value as Record<string, unknown>;
  return (
    typeof ref.type === "string" && typeof ref.id === "string" && typeof ref.label === "string"
  );
}

/** What a stored blob holds, with anything malformed left out rather than trusted. */
export function parseDrafts(raw: string | null): Map<string, ComposerDraft> {
  const drafts = new Map<string, ComposerDraft>();
  if (!raw) {
    return drafts;
  }
  let stored: unknown;
  try {
    stored = JSON.parse(raw);
  } catch {
    return drafts;
  }
  if (typeof stored !== "object" || stored === null) {
    return drafts;
  }
  const blob = stored as { v?: unknown; drafts?: unknown };
  if (blob.v !== STORAGE_VERSION || typeof blob.drafts !== "object" || blob.drafts === null) {
    return drafts;
  }
  for (const [key, value] of Object.entries(blob.drafts as Record<string, unknown>)) {
    if (typeof value !== "object" || value === null) {
      continue;
    }
    const draft = value as { text?: unknown; mentions?: unknown; at?: unknown };
    if (typeof draft.text !== "string" || draft.text === "") {
      continue;
    }
    drafts.set(key, {
      text: draft.text,
      mentions: Array.isArray(draft.mentions) ? draft.mentions.filter(isMention) : NO_MENTIONS,
      at: typeof draft.at === "number" ? draft.at : 0,
    });
  }
  return drafts;
}

/** The drafts as they are written: the newest `MAX_DRAFTS`, none over the length kept. */
export function serializeDrafts(drafts: ReadonlyMap<string, ComposerDraft>): string {
  const kept = [...drafts.entries()]
    .filter(([, draft]) => draft.text.length <= MAX_DRAFT_LENGTH)
    .sort(([, a], [, b]) => b.at - a.at)
    .slice(0, MAX_DRAFTS);
  return JSON.stringify({ v: STORAGE_VERSION, drafts: Object.fromEntries(kept) });
}

function storage(): Storage | null {
  try {
    return typeof window === "undefined" ? null : window.localStorage;
  } catch {
    return null;
  }
}

function createDraftStore() {
  let drafts: Map<string, ComposerDraft> | null = null;
  let timer: ReturnType<typeof setTimeout> | null = null;
  const listeners = new Map<string, Set<Listener>>();

  const notify = (key: string) => {
    for (const listener of listeners.get(key) ?? []) {
      listener();
    }
  };

  const notifyAll = () => {
    for (const set of listeners.values()) {
      for (const listener of set) {
        listener();
      }
    }
  };

  const flush = () => {
    if (timer !== null) {
      clearTimeout(timer);
      timer = null;
    }
    if (drafts === null) {
      return;
    }
    try {
      if (drafts.size === 0) {
        storage()?.removeItem(DRAFT_STORAGE_KEY);
      } else {
        storage()?.setItem(DRAFT_STORAGE_KEY, serializeDrafts(drafts));
      }
    } catch {
      // A full or blocked disk loses the draft on reload; it never loses the typing.
    }
  };

  const load = (): Map<string, ComposerDraft> => {
    if (drafts !== null) {
      return drafts;
    }
    let raw: string | null = null;
    try {
      raw = storage()?.getItem(DRAFT_STORAGE_KEY) ?? null;
    } catch {
      raw = null;
    }
    drafts = parseDrafts(raw);
    if (typeof window !== "undefined") {
      window.addEventListener("pagehide", flush);
      document.addEventListener("visibilitychange", () => {
        if (document.visibilityState === "hidden") {
          flush();
        }
      });
      window.addEventListener("storage", (event) => {
        if (event.key !== DRAFT_STORAGE_KEY || timer !== null) {
          return;
        }
        drafts = parseDrafts(event.newValue);
        notifyAll();
      });
    }
    return drafts;
  };

  const schedule = () => {
    if (timer === null) {
      timer = setTimeout(flush, DRAFT_WRITE_MS);
    }
  };

  const write = (key: string, next: Pick<ComposerDraft, "text" | "mentions">) => {
    const all = load();
    if (next.text === "") {
      if (!all.delete(key)) {
        return;
      }
    } else {
      all.set(key, { text: next.text, mentions: next.mentions, at: Date.now() });
      if (all.size > MAX_DRAFTS) {
        let oldest: string | null = null;
        let oldestAt = Infinity;
        for (const [candidate, draft] of all) {
          if (candidate !== key && draft.at < oldestAt) {
            oldest = candidate;
            oldestAt = draft.at;
          }
        }
        if (oldest !== null) {
          all.delete(oldest);
          notify(oldest);
        }
      }
    }
    schedule();
    notify(key);
  };

  return {
    text: (key: string): string => load().get(key)?.text ?? "",
    mentions: (key: string): readonly AssistantEntityRef[] =>
      load().get(key)?.mentions ?? NO_MENTIONS,
    /** Sets the words of a draft; an empty draft is forgotten, with its records. */
    setText: (key: string, text: string) => {
      const current = load().get(key);
      if ((current?.text ?? "") === text) {
        return;
      }
      write(key, { text, mentions: current?.mentions ?? NO_MENTIONS });
    },
    /** Sets the records a draft names. Records without words are not kept. */
    setMentions: (key: string, mentions: readonly AssistantEntityRef[]) => {
      const current = load().get(key);
      if (!current) {
        return;
      }
      write(key, { text: current.text, mentions });
    },
    subscribe: (key: string, listener: Listener) => {
      let set = listeners.get(key);
      if (!set) {
        set = new Set();
        listeners.set(key, set);
      }
      set.add(listener);
      return () => {
        set.delete(listener);
        if (set.size === 0) {
          listeners.delete(key);
        }
      };
    },
    /** Writes what is pending to disk now. */
    flush,
    /** Forgets every draft, in memory and on disk. */
    clear: () => {
      if (timer !== null) {
        clearTimeout(timer);
        timer = null;
      }
      drafts = new Map();
      try {
        storage()?.removeItem(DRAFT_STORAGE_KEY);
      } catch {
        // Nothing to remove from a disk that cannot be reached.
      }
      notifyAll();
    },
    /** Drops what is in memory so the next read comes from disk. For tests. */
    reload: () => {
      if (timer !== null) {
        clearTimeout(timer);
        timer = null;
      }
      drafts = drafts === null ? null : parseDrafts(storage()?.getItem(DRAFT_STORAGE_KEY) ?? null);
      notifyAll();
    },
  };
}

export const composerDrafts = createDraftStore();

/** The words of the draft kept under `key`, following every change to it. */
export function useDraftText(key: string): string {
  return useSyncExternalStore(
    (listener) => composerDrafts.subscribe(key, listener),
    () => composerDrafts.text(key),
    () => "",
  );
}

/** The records the draft kept under `key` names. */
export function useDraftMentions(key: string): readonly AssistantEntityRef[] {
  return useSyncExternalStore(
    (listener) => composerDrafts.subscribe(key, listener),
    () => composerDrafts.mentions(key),
    () => NO_MENTIONS,
  );
}
