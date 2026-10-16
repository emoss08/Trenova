import { useSyncExternalStore } from "react";

/**
 * Whether this browser has been found to have no speech service behind its
 * recogniser. Helium, Brave and Arc expose the speech API and identify as
 * Chrome, then fail every attempt, so the only way to know is an attempt
 * that fails. The first one is remembered here, and the microphone stays
 * hidden in this browser until the person switches dictation off and on in
 * the Desk's settings.
 *
 * It is kept in this browser's storage rather than in the person's settings,
 * which follow them to other devices: failing in one browser must not hide
 * the microphone in another. It says nothing about the person or their work.
 */
const STORAGE_KEY = "trenova:speech-service-missing";

const listeners = new Set<() => void>();

function readStored(): boolean {
  try {
    return typeof window !== "undefined" && window.localStorage.getItem(STORAGE_KEY) === "1";
  } catch {
    return false;
  }
}

let missing = readStored();

function write(next: boolean) {
  missing = next;
  try {
    if (next) {
      window.localStorage.setItem(STORAGE_KEY, "1");
    } else {
      window.localStorage.removeItem(STORAGE_KEY);
    }
  } catch {
    // Storage blocked or full: remembered for this page only.
  }
  for (const listener of listeners) {
    listener();
  }
}

/** Whether a dictation attempt here found no speech service behind the recogniser. */
export function speechServiceMissing(): boolean {
  return missing;
}

/** Remembers that this browser has no speech service, and tells every microphone. */
export function markSpeechServiceMissing(): void {
  if (!missing) {
    write(true);
  }
}

/** Forgets it, so the next attempt finds out again. */
export function clearSpeechServiceMissing(): void {
  if (missing) {
    write(false);
  }
}

export function subscribeSpeechService(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

/** Whether this browser was found to have no speech service, kept current as that changes. */
export function useSpeechServiceMissing(): boolean {
  return useSyncExternalStore(subscribeSpeechService, speechServiceMissing, () => false);
}
