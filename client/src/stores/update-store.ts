import { updateService } from "@/services/update";
import type { UpdateStatus } from "@/types/update";
import { create } from "zustand";
import { persist } from "zustand/middleware";

interface UpdateState {
  status: UpdateStatus | null;
  isLoading: boolean;
  error: string | null;
  dismissedVersion: string | null;
  hasFetched: boolean;

  fetchStatus: () => Promise<void>;
  dismissUpdate: (version: string) => void;
}

export const useUpdateStore = create<UpdateState>()(
  persist(
    (set, get) => ({
      status: null,
      isLoading: false,
      error: null,
      dismissedVersion: null,
      hasFetched: false,

      fetchStatus: async () => {
        if (get().hasFetched) return;

        set({ isLoading: true, error: null });
        try {
          const status = await updateService.getUpdateStatus();
          set({ status, isLoading: false, hasFetched: true });
        } catch (error) {
          set({
            isLoading: false,
            hasFetched: true,
            error: error instanceof Error ? error.message : "Failed to fetch update status",
          });
        }
      },

      dismissUpdate: (version: string) => {
        set({ dismissedVersion: version });
      },
    }),
    {
      name: "update-storage",
      partialize: (state) => ({
        dismissedVersion: state.dismissedVersion,
      }),
    },
  ),
);
