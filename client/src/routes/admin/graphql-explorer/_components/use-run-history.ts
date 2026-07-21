import { useLocalStorage } from "@/hooks/use-local-storage";
import { useCallback } from "react";

export interface RunHistoryEntry {
  id: string;
  variables: string;
  at: number;
  status: "success" | "error";
  elapsedMs: number;
  bytes?: number;
  httpStatus?: number;
  message?: string;
}

type HistoryMap = Record<string, RunHistoryEntry[]>;

const STORAGE_KEY = "graphql-explorer:run-history";
const MAX_ENTRIES_PER_OPERATION = 20;

const EMPTY_ENTRIES: RunHistoryEntry[] = [];

export function useRunHistory(operationName: string) {
  const [history, setHistory] = useLocalStorage<HistoryMap>(STORAGE_KEY, {});
  const entries = history[operationName] ?? EMPTY_ENTRIES;

  const record = useCallback(
    (entry: Omit<RunHistoryEntry, "id" | "at">) => {
      setHistory((prev) => ({
        ...prev,
        [operationName]: [
          { ...entry, id: crypto.randomUUID(), at: Date.now() },
          ...(prev[operationName] ?? []),
        ].slice(0, MAX_ENTRIES_PER_OPERATION),
      }));
    },
    [operationName, setHistory],
  );

  const clear = useCallback(() => {
    setHistory((prev) => {
      const { [operationName]: _removed, ...rest } = prev;
      return rest;
    });
  }, [operationName, setHistory]);

  return { entries, record, clear };
}
