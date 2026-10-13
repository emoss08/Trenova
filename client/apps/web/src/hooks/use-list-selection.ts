import { useCallback, useMemo, useState } from "react";

export type ListSelection<T> = {
  /** The selected items that are still in the list. */
  selected: T[];
  selectedIds: ReadonlySet<string>;
  isSelected: (id: string) => boolean;
  setSelected: (id: string, selected: boolean) => void;
  /** "all" when every listed item is selected, "some" when only part is. */
  allState: "none" | "some" | "all";
  setAll: (selected: boolean) => void;
  clear: () => void;
};

/**
 * Bulk selection over a list that refetches under it. Selection is kept by id
 * and read back against the current items, so a row that leaves the list (it
 * was approved elsewhere, or filtered out) drops out of the selection instead
 * of being acted on unseen.
 */
export function useListSelection<T>(
  items: readonly T[],
  getId: (item: T) => string,
): ListSelection<T> {
  const [ids, setIds] = useState<ReadonlySet<string>>(() => new Set());

  const selected = useMemo(() => items.filter((item) => ids.has(getId(item))), [items, ids, getId]);
  const selectedIds = useMemo(() => new Set(selected.map(getId)), [selected, getId]);

  const setSelected = useCallback((id: string, value: boolean) => {
    setIds((prev) => {
      const next = new Set(prev);
      if (value) {
        next.add(id);
      } else {
        next.delete(id);
      }
      return next;
    });
  }, []);

  const setAll = useCallback(
    (value: boolean) => setIds(value ? new Set(items.map(getId)) : new Set()),
    [items, getId],
  );
  const clear = useCallback(() => setIds(new Set()), []);
  const isSelected = useCallback((id: string) => selectedIds.has(id), [selectedIds]);

  const allState =
    selected.length === 0 ? "none" : selected.length === items.length ? "all" : "some";

  return { selected, selectedIds, isSelected, setSelected, allState, setAll, clear };
}
