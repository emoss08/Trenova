import { chime } from "@yusufceng1z/ding";
import { create } from "zustand";
import { persist } from "zustand/middleware";

export const TOAST_SOUND_VOLUME = 0.35;

interface ToastSoundState {
  on: boolean;
  setOn: (on: boolean) => void;
}

/**
 * Whether toasts play their cue: on unless this person turns it off, kept with the device's
 * other display choices. Turning it on plays one, so the choice is heard; it follows a
 * press, which browsers require before audio.
 */
export const useToastSoundStore = create<ToastSoundState>()(
  persist(
    (set) => ({
      on: true,
      setOn: (on) => {
        set({ on });
        if (on) chime("success", TOAST_SOUND_VOLUME);
      },
    }),
    { name: "trenova-toast-sound" },
  ),
);
