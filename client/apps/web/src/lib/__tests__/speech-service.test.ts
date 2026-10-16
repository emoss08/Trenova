import { afterEach, describe, expect, it, vi } from "vitest";
import {
  clearSpeechServiceMissing,
  markSpeechServiceMissing,
  speechServiceMissing,
  subscribeSpeechService,
} from "../speech-service";

const STORAGE_KEY = "trenova:speech-service-missing";

afterEach(() => {
  clearSpeechServiceMissing();
  window.localStorage.removeItem(STORAGE_KEY);
});

/**
 * A browser found to have no speech service is remembered in that browser,
 * so the microphone stays hidden there on the next visit, and forgotten when
 * the person asks to try again.
 */
describe("speech service memory", () => {
  it("remembers a missing service in this browser and tells every listener", () => {
    const listener = vi.fn();
    const unsubscribe = subscribeSpeechService(listener);

    markSpeechServiceMissing();

    expect(speechServiceMissing()).toBe(true);
    expect(window.localStorage.getItem(STORAGE_KEY)).toBe("1");
    expect(listener).toHaveBeenCalledTimes(1);
    unsubscribe();
  });

  it("forgets it, so the next attempt finds out again", () => {
    markSpeechServiceMissing();
    const listener = vi.fn();
    const unsubscribe = subscribeSpeechService(listener);

    clearSpeechServiceMissing();

    expect(speechServiceMissing()).toBe(false);
    expect(window.localStorage.getItem(STORAGE_KEY)).toBeNull();
    expect(listener).toHaveBeenCalledTimes(1);
    unsubscribe();
  });

  it("tells nobody when nothing changed", () => {
    const listener = vi.fn();
    const unsubscribe = subscribeSpeechService(listener);

    clearSpeechServiceMissing();
    markSpeechServiceMissing();
    markSpeechServiceMissing();

    expect(listener).toHaveBeenCalledTimes(1);
    unsubscribe();
  });

  it("still remembers for this page when the browser's storage refuses", () => {
    const setItem = vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
      throw new DOMException("full", "QuotaExceededError");
    });

    markSpeechServiceMissing();

    expect(speechServiceMissing()).toBe(true);
    setItem.mockRestore();
  });

  it("stops telling a listener that unsubscribed", () => {
    const listener = vi.fn();
    subscribeSpeechService(listener)();

    markSpeechServiceMissing();

    expect(listener).not.toHaveBeenCalled();
  });
});
