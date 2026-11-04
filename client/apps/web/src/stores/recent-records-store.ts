import type { GlobalSearchEntityType } from "@/services/global-search";
import { create } from "zustand";

export const RECENT_RECORDS_LIMIT = 10;

const EMPTY_RECORDS: readonly RecentRecord[] = [];

/** Where an earlier build kept these on disk; it is deleted wherever it is still found. */
const RETIRED_STORAGE_KEY = "recent-records-storage";

export interface RecentRecord {
  entityType: GlobalSearchEntityType;
  id: string;
  title: string;
  subtitle?: string;
  href: string;
  metadata?: Record<string, string>;
  visitedAt: number;
}

export type RecentRecordInput = Omit<RecentRecord, "visitedAt">;

interface RecentRecordsState {
  recordsByOrganization: Record<string, RecentRecord[]>;
  recordOpen: (organizationId: string | undefined, record: RecentRecordInput) => void;
  touch: (
    organizationId: string | undefined,
    entityType: GlobalSearchEntityType,
    id: string,
  ) => void;
  forget: (
    organizationId: string | undefined,
    entityType: GlobalSearchEntityType,
    id: string,
  ) => void;
  /** Forgets every record, for the end of a session. */
  clear: () => void;
}

function sameRecord(
  entry: Pick<RecentRecord, "entityType" | "id">,
  entityType: GlobalSearchEntityType,
  id: string,
): boolean {
  return entry.entityType === entityType && entry.id === id;
}

/**
 * The records a person opened most recently, newest first and kept per
 * organization. A record is remembered with the name it was found under, so
 * the palette can offer it again before any search has run. Opening one of
 * them from anywhere else in the app moves it back to the front, but a page
 * alone cannot add one: it knows an id, not what the record is called.
 *
 * They are held in memory only and forgotten when the session ends
 * (`clearSessionCache`): a record's name and status written to the browser's
 * disk would stay on a shared dispatch-office PC for the next person to find.
 */
export const useRecentRecordsStore = create<RecentRecordsState>()((set) => ({
  recordsByOrganization: {},

  recordOpen: (organizationId, record) => {
    if (!organizationId || record.id === "" || record.title.trim() === "") {
      return;
    }
    set((state) => {
      const current = state.recordsByOrganization[organizationId] ?? EMPTY_RECORDS;
      const others = current.filter((entry) => !sameRecord(entry, record.entityType, record.id));
      const next = [
        { ...record, title: record.title.trim(), visitedAt: Date.now() },
        ...others,
      ].slice(0, RECENT_RECORDS_LIMIT);
      return {
        recordsByOrganization: { ...state.recordsByOrganization, [organizationId]: next },
      };
    });
  },

  touch: (organizationId, entityType, id) => {
    if (!organizationId || id === "") {
      return;
    }
    set((state) => {
      const current = state.recordsByOrganization[organizationId];
      const index = current?.findIndex((entry) => sameRecord(entry, entityType, id)) ?? -1;
      if (!current || index < 0) {
        return state;
      }
      const touched = { ...current[index], visitedAt: Date.now() };
      const next = [touched, ...current.slice(0, index), ...current.slice(index + 1)];
      return {
        recordsByOrganization: { ...state.recordsByOrganization, [organizationId]: next },
      };
    });
  },

  forget: (organizationId, entityType, id) => {
    if (!organizationId) {
      return;
    }
    set((state) => {
      const current = state.recordsByOrganization[organizationId];
      if (!current) {
        return state;
      }
      return {
        recordsByOrganization: {
          ...state.recordsByOrganization,
          [organizationId]: current.filter((entry) => !sameRecord(entry, entityType, id)),
        },
      };
    });
  },

  clear: () => set({ recordsByOrganization: {} }),
}));

if (typeof window !== "undefined") {
  try {
    window.localStorage.removeItem(RETIRED_STORAGE_KEY);
  } catch {
    // Storage the browser refuses to open holds nothing to delete.
  }
}

export function useRecentRecords(organizationId: string | undefined): readonly RecentRecord[] {
  return useRecentRecordsStore((state) =>
    organizationId ? (state.recordsByOrganization[organizationId] ?? EMPTY_RECORDS) : EMPTY_RECORDS,
  );
}
