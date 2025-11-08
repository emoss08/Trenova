import { queries } from "@/lib/queries";
import type {
  PreferenceDataSchema,
  UpdatePreferenceDataSchema,
} from "@/lib/schemas/user-preference-schema";
import { api } from "@/services/api";
import { useQuery } from "@tanstack/react-query";
import { useEffect } from "react";
import type { StoreApi, UseBoundStore } from "zustand";
import { create } from "zustand";
import { createJSONStorage, persist } from "zustand/middleware";

const STORAGE_KEY = "trenova-user-preferences";
const SYNC_DEBOUNCE_MS = 1000;

type WithSelectors<S> = S extends { getState: () => infer T }
  ? S & { use: { [K in keyof T]: () => T[K] } }
  : never;

const createSelectors = <S extends UseBoundStore<StoreApi<object>>>(
  _store: S,
) => {
  const store = _store as WithSelectors<typeof _store>;
  store.use = {} as any;
  for (const k of Object.keys(store.getState())) {
    (store.use as any)[k] = () => store((s) => s[k as keyof typeof s]);
  }

  return store;
};

interface UserPreferenceState {
  dismissedNotices: string[];
  dismissedDialogs: string[];
  uiSettings: Record<string, any>;
  isSyncing: boolean;
  lastSyncedVersion: number | null;
  isDismissed: (key: string, type: "notice" | "dialog") => boolean;
  dismissNotice: (key: string) => void;
  dismissDialog: (key: string) => void;
  getSetting: <T = any>(key: string, defaultValue?: T) => T | undefined;
  setSetting: (key: string, value: any) => void;
  clearPreferences: () => void;
  _syncToBackend: (updates: UpdatePreferenceDataSchema) => void;
  _syncFromBackend: (
    backendData: PreferenceDataSchema,
    backendVersion: number,
  ) => void;
  _setLastSyncedVersion: (version: number) => void;
}

let syncTimer: ReturnType<typeof setTimeout> | null = null;

const useUserPreferenceStoreBase = create<UserPreferenceState>()(
  persist(
    (set, get) => ({
      dismissedNotices: [],
      dismissedDialogs: [],
      uiSettings: {},
      isSyncing: false,
      lastSyncedVersion: null,

      isDismissed: (key: string, type: "notice" | "dialog" = "notice") => {
        const state = get();
        const list =
          type === "notice" ? state.dismissedNotices : state.dismissedDialogs;
        return list.includes(key);
      },

      dismissNotice: (key: string) => {
        const state = get();
        if (!state.isDismissed(key, "notice")) {
          set((prev) => ({
            dismissedNotices: [...prev.dismissedNotices, key],
          }));

          get()._syncToBackend({
            dismissedNotices: [...state.dismissedNotices, key],
          });
        }
      },

      dismissDialog: (key: string) => {
        const state = get();
        if (!state.isDismissed(key, "dialog")) {
          set((prev) => ({
            dismissedDialogs: [...prev.dismissedDialogs, key],
          }));

          get()._syncToBackend({
            dismissedDialogs: [...state.dismissedDialogs, key],
          });
        }
      },

      getSetting: <T = any>(key: string, defaultValue?: T): T | undefined => {
        return (get().uiSettings[key] as T) ?? defaultValue;
      },

      setSetting: (key: string, value: any) => {
        set((prev) => ({
          uiSettings: {
            ...prev.uiSettings,
            [key]: value,
          },
        }));

        get()._syncToBackend({
          uiSettings: {
            [key]: value,
          },
        });
      },

      clearPreferences: () => {
        set({
          dismissedNotices: [],
          dismissedDialogs: [],
          uiSettings: {},
        });
      },

      _syncToBackend: (updates: UpdatePreferenceDataSchema) => {
        if (syncTimer) {
          clearTimeout(syncTimer);
        }

        syncTimer = setTimeout(async () => {
          set({ isSyncing: true });
          try {
            const response = await api.userPreference.merge(updates);
            if (response.version !== undefined) {
              set({ lastSyncedVersion: response.version });
            }
          } catch (error) {
            console.error("Failed to sync preferences to backend:", error);
          } finally {
            set({ isSyncing: false });
          }
        }, SYNC_DEBOUNCE_MS);
      },

      _syncFromBackend: (
        backendData: PreferenceDataSchema,
        backendVersion: number,
      ) => {
        const currentVersion = get().lastSyncedVersion;

        if (currentVersion === null || backendVersion !== currentVersion) {
          set({
            dismissedNotices: backendData.dismissedNotices,
            dismissedDialogs: backendData.dismissedDialogs,
            uiSettings: backendData.uiSettings,
            lastSyncedVersion: backendVersion,
          });
        }
      },

      _setLastSyncedVersion: (version: number) => {
        set({ lastSyncedVersion: version });
      },
    }),
    {
      name: STORAGE_KEY,
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        dismissedNotices: state.dismissedNotices,
        dismissedDialogs: state.dismissedDialogs,
        uiSettings: state.uiSettings,
        lastSyncedVersion: state.lastSyncedVersion,
      }),
    },
  ),
);

export const useUserPreferenceStore = createSelectors(
  useUserPreferenceStoreBase,
);

export function useInitializeUserPreferences() {
  const { data: backendPreferences } = useQuery({
    ...queries.organization.getUserPreference(),
  });

  const syncFromBackend = useUserPreferenceStore(
    (state) => state._syncFromBackend,
  );

  useEffect(() => {
    if (backendPreferences?.version !== undefined) {
      console.log("[UserPreference] Backend data loaded, attempting sync:", {
        version: backendPreferences.version,
        preferences: backendPreferences.preferences,
      });
      syncFromBackend(
        backendPreferences.preferences,
        backendPreferences.version,
      );
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [backendPreferences?.version]);
}

export function useNotice(key: string) {
  const isDismissed = useUserPreferenceStore((state) =>
    state.dismissedNotices.includes(key),
  );
  const dismissNotice = useUserPreferenceStore((state) => state.dismissNotice);

  return {
    isDismissed,
    dismiss: () => dismissNotice(key),
  };
}

export function useDialog(key: string) {
  const isDismissed = useUserPreferenceStore((state) =>
    state.dismissedDialogs.includes(key),
  );
  const dismissDialog = useUserPreferenceStore((state) => state.dismissDialog);

  return {
    isDismissed,
    dismiss: () => dismissDialog(key),
  };
}
