import { queries } from "@/lib/queries";
import type {
  PreferenceDataSchema,
  UpdatePreferenceDataSchema,
} from "@/lib/schemas/user-preference-schema";
import { api } from "@/services/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useRef, useState } from "react";

const STORAGE_KEY = "trenova-user-preferences";
const SYNC_DEBOUNCE_MS = 1000;

function getLocalPreferences(): PreferenceDataSchema {
  if (typeof window !== "undefined") {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    if (stored) {
      try {
        return JSON.parse(stored);
      } catch (e) {
        console.error("Failed to parse stored preferences:", e);
      }
    }
  }
  return {
    dismissedNotices: [],
    dismissedDialogs: [],
    uiSettings: {},
  };
}

function mergePreferences(
  local: PreferenceDataSchema,
  backend: PreferenceDataSchema,
): PreferenceDataSchema {
  return {
    dismissedNotices: Array.from(
      new Set([...local.dismissedNotices, ...backend.dismissedNotices]),
    ),
    dismissedDialogs: Array.from(
      new Set([...local.dismissedDialogs, ...backend.dismissedDialogs]),
    ),
    uiSettings: {
      ...local.uiSettings,
      ...backend.uiSettings,
    },
  };
}

export function useUserPreference() {
  const queryClient = useQueryClient();
  const mergedBackendIdRef = useRef<string | null>(null);

  const { data: backendPreferences } = useQuery({
    ...queries.organization.getUserPreference(),
  });

  const [localPreferences, setLocalPreferences] =
    useState<PreferenceDataSchema>(getLocalPreferences);

  const backendId = backendPreferences?.id;
  const backendPrefs = backendPreferences?.preferences;

  // eslint-disable-next-line react-hooks/refs
  if (backendId && backendPrefs && mergedBackendIdRef.current !== backendId) {
    // eslint-disable-next-line react-hooks/refs
    mergedBackendIdRef.current = backendId;

    const local = getLocalPreferences();
    const merged = mergePreferences(local, backendPrefs);

    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(merged));

    setLocalPreferences(merged);
  }

  const { mutate: updateBackend } = useMutation({
    mutationFn: async (updates: UpdatePreferenceDataSchema) => {
      return await api.userPreference.merge(updates);
    },
    onSuccess: (data) => {
      queryClient.setQueryData(
        queries.organization.getUserPreference().queryKey,
        data,
      );
    },
    onError: (error) => {
      console.error("Failed to sync preferences to backend:", error);
    },
  });

  const [syncTimer, setSyncTimer] = useState<ReturnType<
    typeof setTimeout
  > | null>(null);

  const syncToBackend = useCallback(
    (updates: UpdatePreferenceDataSchema) => {
      if (syncTimer) {
        clearTimeout(syncTimer);
      }

      const timer = setTimeout(() => {
        updateBackend(updates);
      }, SYNC_DEBOUNCE_MS);

      setSyncTimer(timer);
    },
    [syncTimer, updateBackend],
  );

  const updatePreferences = useCallback(
    (updates: Partial<PreferenceDataSchema>) => {
      setLocalPreferences((prev) => {
        const updated = {
          dismissedNotices: updates.dismissedNotices ?? prev.dismissedNotices,
          dismissedDialogs: updates.dismissedDialogs ?? prev.dismissedDialogs,
          uiSettings: updates.uiSettings
            ? { ...prev.uiSettings, ...updates.uiSettings }
            : prev.uiSettings,
        };

        window.localStorage.setItem(STORAGE_KEY, JSON.stringify(updated));

        syncToBackend(updates);

        return updated;
      });
    },
    [syncToBackend],
  );

  const isDismissed = useCallback(
    (key: string, type: "notice" | "dialog" = "notice"): boolean => {
      const list =
        type === "notice"
          ? localPreferences.dismissedNotices
          : localPreferences.dismissedDialogs;
      return list.includes(key);
    },
    [localPreferences],
  );

  const dismissNotice = useCallback(
    (key: string) => {
      if (!isDismissed(key, "notice")) {
        updatePreferences({
          dismissedNotices: [...localPreferences.dismissedNotices, key],
        });
      }
    },
    [isDismissed, localPreferences.dismissedNotices, updatePreferences],
  );

  const dismissDialog = useCallback(
    (key: string) => {
      if (!isDismissed(key, "dialog")) {
        updatePreferences({
          dismissedDialogs: [...localPreferences.dismissedDialogs, key],
        });
      }
    },
    [isDismissed, localPreferences.dismissedDialogs, updatePreferences],
  );

  const getSetting = useCallback(
    <T = any>(key: string, defaultValue?: T): T | undefined => {
      return (localPreferences.uiSettings[key] as T) ?? defaultValue;
    },
    [localPreferences.uiSettings],
  );

  const setSetting = useCallback(
    (key: string, value: any) => {
      updatePreferences({
        uiSettings: {
          [key]: value,
        },
      });
    },
    [updatePreferences],
  );

  const clearPreferences = useCallback(() => {
    const empty: PreferenceDataSchema = {
      dismissedNotices: [],
      dismissedDialogs: [],
      uiSettings: {},
    };
    setLocalPreferences(empty);
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(empty));
  }, []);

  return {
    preferences: localPreferences,
    isDismissed,
    dismissNotice,
    dismissDialog,
    getSetting,
    setSetting,
    clearPreferences,
  };
}
