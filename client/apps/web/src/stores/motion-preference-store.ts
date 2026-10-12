import { create } from "zustand";
import { persist } from "zustand/middleware";

interface MotionPreferenceState {
  reduceMotion: boolean;
  setReduceMotion: (reduce: boolean) => void;
}

/**
 * Whether this person asked Trenova for less movement. It is kept with the device's
 * other display choices, beside the theme, and applied to the whole app.
 */
export const useMotionPreferenceStore = create<MotionPreferenceState>()(
  persist(
    (set) => ({
      reduceMotion: false,
      setReduceMotion: (reduce) => set({ reduceMotion: reduce }),
    }),
    { name: "trenova-motion-preference" },
  ),
);
