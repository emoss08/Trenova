import { useCallback, useSyncExternalStore } from "react";

/** What a TanStack Table v9 state atom offers a reader: its value and a way to hear it change. */
export type TableAtomSource<T> = {
  get: () => T;
  subscribe: (listener: (value: T) => void) => { unsubscribe: () => void };
};

const identity = <T>(value: T) => value;

/**
 * Reads one table state atom, or a value derived from it, and redraws the caller
 * only when that value changes. A selector that returns a primitive (a count, one
 * row's checkbox) lets a component follow a slice of state that changes often
 * while redrawing only when its own answer does.
 */
export function useTableAtom<T, S = T>(
  atom: TableAtomSource<T>,
  selector: (value: T) => S = identity as (value: T) => S,
): S {
  const subscribe = useCallback(
    (onChange: () => void) => {
      const subscription = atom.subscribe(onChange);
      return () => subscription.unsubscribe();
    },
    [atom],
  );
  return useSyncExternalStore(subscribe, () => selector(atom.get()));
}
