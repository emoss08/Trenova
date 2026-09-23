import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useComposerDictation } from "../use-composer-dictation";
import {
  dictationAvailability,
  issueFromMediaError,
  issueFromRecognitionError,
  joinSpoken,
  useDictation,
} from "../use-dictation";
import {
  dictationScope,
  FakeRecognition,
  openMicrophone,
  refusedMicrophone,
} from "./dictation-fakes";

beforeEach(() => {
  FakeRecognition.instances = [];
});

afterEach(cleanup);

/**
 * The page used to decide only whether the constructor existed. A browser
 * with the API and a page whose Permissions-Policy turns the microphone off
 * looked the same, and the second one failed every click in silence.
 */
describe("dictationAvailability", () => {
  it("is unsupported without a recognition constructor, whatever else holds", () => {
    expect(dictationAvailability(dictationScope({ recognition: null }))).toBe("unsupported");
  });

  it("needs a secure context", () => {
    expect(dictationAvailability(dictationScope({ secure: false }))).toBe("insecure");
  });

  it("is blocked when the page's policy turns the microphone off", () => {
    expect(dictationAvailability(dictationScope({ policyAllowsMicrophone: false }))).toBe(
      "blocked",
    );
  });

  it("is available when the policy says nothing either way", () => {
    expect(dictationAvailability(dictationScope({ policyAllowsMicrophone: null }))).toBe(
      "available",
    );
  });
});

describe("issueFromRecognitionError", () => {
  it.each([
    ["not-allowed", "denied"],
    ["service-not-allowed", "service"],
    ["audio-capture", "no-microphone"],
    ["network", "network"],
    ["no-speech", "no-speech"],
    ["language-not-supported", "language"],
    ["bad-grammar", "failed"],
  ] as const)("reads %s as %s", (code, issue) => {
    expect(issueFromRecognitionError(code, false)).toBe(issue);
  });

  it("is silent about an abort the person asked for, and not about one they did not", () => {
    expect(issueFromRecognitionError("aborted", true)).toBeNull();
    expect(issueFromRecognitionError("aborted", false)).toBe("interrupted");
  });
});

describe("issueFromMediaError", () => {
  it.each([
    ["NotAllowedError", "denied"],
    ["SecurityError", "blocked"],
    ["NotFoundError", "no-microphone"],
    ["NotReadableError", "microphone-busy"],
    ["TypeError", "failed"],
  ] as const)("reads a %s as %s", (name, issue) => {
    expect(issueFromMediaError(new DOMException("x", name))).toBe(issue);
  });

  it("reads something that is not an error at all as a failure", () => {
    expect(issueFromMediaError("nope")).toBe("failed");
    expect(issueFromMediaError(null)).toBe("failed");
  });
});

describe("joinSpoken", () => {
  it("puts one space between what was typed and what was said", () => {
    expect(joinSpoken("Check on ", " load 42 ")).toBe("Check on load 42");
  });

  it("drops empty parts and keeps an empty draft empty", () => {
    expect(joinSpoken("", "", "  ")).toBe("");
    expect(joinSpoken("", "hello", "", "there")).toBe("hello there");
  });

  it("keeps the typed text's own lines", () => {
    expect(joinSpoken("first line\nsecond", "spoken")).toBe("first line\nsecond spoken");
  });
});

describe("useDictation", () => {
  it("asks for the microphone before it starts listening, and lets it go when listening ends", async () => {
    const microphone = openMicrophone();
    const scope = dictationScope({ getUserMedia: microphone.getUserMedia });
    const { result } = renderHook(() => useDictation({ scope, onTranscript: () => {} }));

    act(() => result.current.start());
    expect(result.current.phase).toBe("starting");
    expect(microphone.getUserMedia).toHaveBeenCalledWith({ audio: true });
    expect(FakeRecognition.instances).toHaveLength(0);

    await act(async () => {});
    expect(FakeRecognition.latest().start).toHaveBeenCalled();
    expect(result.current.phase).toBe("listening");
    expect(result.current.stream).not.toBeNull();

    act(() => FakeRecognition.latest().emitEnd());
    expect(result.current.phase).toBe("idle");
    expect(result.current.stream).toBeNull();
    expect(microphone.trackStop).toHaveBeenCalled();
  });

  /**
   * The bug: a refused microphone set an error nobody read, so clicking the
   * mic did nothing at all. The refusal is now an issue the composer shows.
   */
  it("reports a refused microphone and never starts recognition", async () => {
    const scope = dictationScope({
      getUserMedia: refusedMicrophone("NotAllowedError").getUserMedia,
    });
    const { result } = renderHook(() => useDictation({ scope, onTranscript: () => {} }));

    act(() => result.current.start());
    await act(async () => {});

    expect(result.current.issue).toBe("denied");
    expect(result.current.phase).toBe("idle");
    expect(FakeRecognition.instances).toHaveLength(0);
  });

  it("surfaces a recognition error once the recogniser ends", () => {
    const scope = dictationScope();
    const { result } = renderHook(() => useDictation({ scope, onTranscript: () => {} }));

    act(() => result.current.start());
    act(() => {
      FakeRecognition.latest().emitError("network");
      FakeRecognition.latest().emitEnd();
    });

    expect(result.current.issue).toBe("network");
    expect(result.current.phase).toBe("idle");
  });

  it("does not start at all where the policy blocks the microphone", () => {
    const microphone = openMicrophone();
    const scope = dictationScope({
      getUserMedia: microphone.getUserMedia,
      policyAllowsMicrophone: false,
    });
    const { result } = renderHook(() => useDictation({ scope, onTranscript: () => {} }));

    act(() => result.current.start());

    expect(result.current.supported).toBe(false);
    expect(microphone.getUserMedia).not.toHaveBeenCalled();
    expect(FakeRecognition.instances).toHaveLength(0);
  });

  it("stops a start that is still waiting on the permission prompt", async () => {
    const microphone = openMicrophone();
    const scope = dictationScope({ getUserMedia: microphone.getUserMedia });
    const onEnd = vi.fn();
    const { result } = renderHook(() => useDictation({ scope, onTranscript: () => {}, onEnd }));

    act(() => result.current.start());
    act(() => result.current.stop());
    await act(async () => {});

    expect(result.current.phase).toBe("idle");
    expect(onEnd).toHaveBeenCalledTimes(1);
    expect(FakeRecognition.instances).toHaveLength(0);
    expect(microphone.trackStop).toHaveBeenCalled();
  });
});

function renderComposerDictation(initial: string) {
  const drafts: string[] = [];
  let draft = initial;
  const scope = dictationScope();
  const view = renderHook(() =>
    useComposerDictation({
      draft,
      onDraftChange: (next) => {
        draft = next;
        drafts.push(next);
      },
      scope,
    }),
  );

  return { ...view, drafts, current: () => draft };
}

/**
 * Words appear in the draft as they are heard, after what was typed, and the
 * guess in progress is replaced as it firms up.
 */
describe("useComposerDictation", () => {
  it("writes the guess and then the phrase into the draft after what was typed", () => {
    const view = renderComposerDictation("Check on");

    act(() => view.result.current.toggleFromDraft());
    act(() => FakeRecognition.latest().emitResults([{ transcript: "load", isFinal: false }]));
    expect(view.current()).toBe("Check on load");

    act(() => FakeRecognition.latest().emitResults([{ transcript: "load 42", isFinal: true }]));
    expect(view.current()).toBe("Check on load 42");
  });

  /**
   * Two phrases finished in one event used to be written one at a time from
   * the same stale draft, so the second overwrote the first.
   */
  it("keeps every phrase that finishes in one event", () => {
    const view = renderComposerDictation("");

    act(() => view.result.current.toggleFromDraft());
    act(() =>
      FakeRecognition.latest().emitResults([
        { transcript: "where is", isFinal: true },
        { transcript: "shipment 12", isFinal: true },
      ]),
    );

    expect(view.current()).toBe("where is shipment 12");
  });

  it("keeps the phrase that lands after stop, and drops a guess that never did", () => {
    const view = renderComposerDictation("");

    act(() => view.result.current.toggleFromDraft());
    act(() => FakeRecognition.latest().emitResults([{ transcript: "hello", isFinal: false }]));
    act(() => view.result.current.toggleFromDraft());
    expect(FakeRecognition.latest().stop).toHaveBeenCalled();
    expect(view.result.current.phase).toBe("idle");

    act(() => FakeRecognition.latest().emitResults([{ transcript: "hello there", isFinal: true }]));
    act(() => FakeRecognition.latest().emitResults([{ transcript: "and", isFinal: false }]));
    act(() => FakeRecognition.latest().emitEnd());

    expect(view.current()).toBe("hello there");
  });

  it("stops writing once the person takes the box back", () => {
    const view = renderComposerDictation("");

    act(() => view.result.current.toggleFromDraft());
    act(() => FakeRecognition.latest().emitResults([{ transcript: "hel", isFinal: false }]));
    act(() => view.result.current.release());
    const written = view.drafts.length;
    act(() => FakeRecognition.latest().emitResults([{ transcript: "hello", isFinal: true }]));
    act(() => FakeRecognition.latest().emitEnd());

    expect(FakeRecognition.latest().abort).toHaveBeenCalled();
    expect(view.drafts).toHaveLength(written);
  });

  it("leaves the draft alone when nothing was heard", () => {
    const view = renderComposerDictation("typed   ");

    act(() => view.result.current.toggleFromDraft());
    act(() => {
      FakeRecognition.latest().emitError("no-speech");
      FakeRecognition.latest().emitEnd();
    });

    expect(view.drafts).toHaveLength(0);
    expect(view.result.current.issue).toBe("no-speech");
  });
});
