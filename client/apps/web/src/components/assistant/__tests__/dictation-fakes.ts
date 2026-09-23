import { vi } from "vitest";
import type { DictationScope } from "../use-dictation";

/**
 * A SpeechRecognition that does what the Web Speech API does, driven by the
 * test: `start` fires `onstart`, and results, errors and the end are pushed
 * in by hand in the order a browser sends them.
 */
export class FakeRecognition extends EventTarget {
  static instances: FakeRecognition[] = [];

  lang = "";
  continuous = false;
  interimResults = false;
  maxAlternatives = 1;
  onresult: ((event: SpeechRecognitionEvent) => void) | null = null;
  onerror: ((event: SpeechRecognitionErrorEvent) => void) | null = null;
  onstart: (() => void) | null = null;
  onend: (() => void) | null = null;
  start = vi.fn(() => {
    this.onstart?.();
  });
  stop = vi.fn();
  abort = vi.fn();

  constructor() {
    super();
    FakeRecognition.instances.push(this);
  }

  static latest(): FakeRecognition {
    const latest = FakeRecognition.instances.at(-1);
    if (!latest) {
      throw new Error("no recognition was started");
    }

    return latest;
  }

  /** One `result` event; each entry is one result in the list from `resultIndex` on. */
  emitResults(entries: { transcript: string; isFinal: boolean }[], resultIndex = 0) {
    const results = entries.map((entry) =>
      Object.assign([{ transcript: entry.transcript, confidence: 1 }], { isFinal: entry.isFinal }),
    );
    this.onresult?.({ resultIndex, results } as unknown as SpeechRecognitionEvent);
  }

  emitError(error: string) {
    this.onerror?.({ error, message: "" } as unknown as SpeechRecognitionErrorEvent);
  }

  emitEnd() {
    this.onend?.();
  }
}

export type FakeMicrophone = {
  getUserMedia: ReturnType<
    typeof vi.fn<(constraints: MediaStreamConstraints) => Promise<MediaStream>>
  >;
  trackStop: ReturnType<typeof vi.fn>;
};

/** A microphone that opens, with a track whose `stop` the test can watch. */
export function openMicrophone(): FakeMicrophone {
  const trackStop = vi.fn();
  const stream = { getTracks: () => [{ stop: trackStop }] } as unknown as MediaStream;

  return {
    getUserMedia: vi.fn(async (_constraints: MediaStreamConstraints) => stream),
    trackStop,
  };
}

/** A microphone the browser refuses, with the DOMException name it refuses with. */
export function refusedMicrophone(name: string): FakeMicrophone {
  return {
    getUserMedia: vi.fn(async (_constraints: MediaStreamConstraints) => {
      throw new DOMException("refused", name);
    }),
    trackStop: vi.fn(),
  };
}

export function dictationScope(overrides: Partial<DictationScope> = {}): DictationScope {
  return {
    recognition: FakeRecognition as unknown as DictationScope["recognition"],
    secure: true,
    getUserMedia: null,
    policyAllowsMicrophone: true,
    ...overrides,
  };
}
