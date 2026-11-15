import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { useEffect, useState } from "react";

/**
 * A search box whose query lives in the address. The box holds what is being
 * typed; once typing pauses the query is written to the address, which is
 * the source of truth. When the address changes on its own — Back, a link, a
 * "clear the search" elsewhere on the page — the box follows it and the
 * pending text it replaced is dropped rather than written back.
 *
 * Starting or clearing a search is a step Back can undo; refining one that is
 * already running replaces it, so Back does not walk through every pause.
 */
export function useUrlSearch({
  query,
  onCommit,
  delay,
}: {
  /** The query the address holds now. */
  query: string;
  /** Writes a query to the address; `replace` refines the current entry. */
  onCommit: (query: string, replace: boolean) => void;
  delay: number;
}): [string, (value: string) => void] {
  const [draft, setDraft] = useState({ text: query, source: query });
  if (draft.source !== query) {
    setDraft({
      text: draft.text.trim() === query.trim() ? draft.text : query,
      source: query,
    });
  }
  const text = draft.source === query ? draft.text : query;
  const debounced = useDebounce(text, delay);

  useEffect(() => {
    if (debounced !== text || debounced.trim() === query.trim()) {
      return;
    }
    onCommit(debounced, query.trim() !== "" && debounced.trim() !== "");
  }, [debounced, text, query, onCommit]);

  return [text, (value: string) => setDraft({ text: value, source: query })];
}
