import { create } from "zustand";
import { devtools } from "zustand/middleware";

export type RealtimeConnectionState = "connecting" | "connected" | "disconnected";

interface RealtimeState {
  connectionState: RealtimeConnectionState;
  setConnectionState: (connectionState: RealtimeConnectionState) => void;
  lastEventAt: number | null;
  setLastEventAt: (lastEventAt: number | null) => void;
}

export const useRealtimeStore = create<RealtimeState>()(
  devtools((set) => ({
    connectionState: "disconnected",
    setConnectionState: (connectionState) => set({ connectionState }),
    lastEventAt: null,
    setLastEventAt: (lastEventAt) => set({ lastEventAt }),
  })),
);
