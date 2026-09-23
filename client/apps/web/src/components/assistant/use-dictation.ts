import { useCallback, useEffect, useMemo, useRef, useState } from "react";

/**
 * Whether dictation can run here at all, and if not, which of the reasons a
 * person could be told about. Each has a different remedy, so they are not
 * folded into one "unsupported".
 */
export type DictationAvailability = "available" | "unsupported" | "insecure" | "blocked";

/**
 * Why an attempt ended without the person ending it. Each maps to one
 * sentence the composer shows, because "nothing happens" is what a person
 * sees when these are swallowed.
 */
export type DictationIssue =
  | "denied"
  | "blocked"
  | "service"
  | "no-microphone"
  | "microphone-busy"
  | "network"
  | "no-speech"
  | "language"
  | "interrupted"
  | "failed";

/**
 * `starting` covers the permission prompt and the recogniser warming up, so
 * the control can say it heard the click before the browser answers.
 */
export type DictationPhase = "idle" | "starting" | "listening";

export type DictationState = {
  availability: DictationAvailability;
  /** Shorthand for `availability === "available"`. */
  supported: boolean;
  phase: DictationPhase;
  /** True from the click until the recogniser lets go: starting or listening. */
  listening: boolean;
  /** Why the last attempt stopped, when it was not the person stopping it. */
  issue: DictationIssue | null;
  /** The microphone while it is open, for a level meter. */
  stream: MediaStream | null;
  start: () => void;
  /** Stops listening and keeps the phrase in progress. */
  stop: () => void;
  /** Stops listening and drops the phrase in progress, without calling back. */
  abort: () => void;
  toggle: () => void;
  dismissIssue: () => void;
};

type RecognitionConstructor = new () => SpeechRecognition;

/** What the page offers dictation, read once per question so a test can hand it in. */
export type DictationScope = {
  recognition: RecognitionConstructor | null;
  secure: boolean;
  getUserMedia: ((constraints: MediaStreamConstraints) => Promise<MediaStream>) | null;
  /** Null when the browser does not say what its Permissions-Policy allows. */
  policyAllowsMicrophone: boolean | null;
};

/** How long a stopped recogniser may take to hand back its last phrase before it is let go. */
const STOP_GRACE_MS = 1500;

export function readDictationScope(): DictationScope {
  if (typeof window === "undefined") {
    return { recognition: null, secure: false, getUserMedia: null, policyAllowsMicrophone: null };
  }
  const devices = typeof navigator === "undefined" ? undefined : navigator.mediaDevices;
  const policy =
    typeof document === "undefined"
      ? undefined
      : (document.permissionsPolicy ?? document.featurePolicy);

  return {
    recognition: window.SpeechRecognition ?? window.webkitSpeechRecognition ?? null,
    secure: window.isSecureContext !== false,
    getUserMedia:
      devices && typeof devices.getUserMedia === "function"
        ? (constraints) => devices.getUserMedia(constraints)
        : null,
    policyAllowsMicrophone:
      policy && typeof policy.allowsFeature === "function"
        ? policy.allowsFeature("microphone")
        : null,
  };
}

export function dictationAvailability(scope: DictationScope): DictationAvailability {
  if (scope.recognition === null) {
    return "unsupported";
  }
  if (!scope.secure) {
    return "insecure";
  }
  if (scope.policyAllowsMicrophone === false) {
    return "blocked";
  }

  return "available";
}

/**
 * A SpeechRecognition error code as an issue. "aborted" is the person
 * stopping when they asked for it, and something else taking the microphone
 * when they did not. Chromium builds without Google's speech service (Brave,
 * Arc, Electron) fail every attempt with "network".
 */
export function issueFromRecognitionError(
  code: string,
  stoppedByPerson: boolean,
): DictationIssue | null {
  switch (code) {
    case "not-allowed":
      return "denied";
    case "service-not-allowed":
      return "service";
    case "audio-capture":
      return "no-microphone";
    case "network":
      return "network";
    case "no-speech":
      return "no-speech";
    case "language-not-supported":
      return "language";
    case "aborted":
      return stoppedByPerson ? null : "interrupted";
    default:
      return "failed";
  }
}

/** A getUserMedia rejection as an issue, read from the DOMException's name. */
export function issueFromMediaError(error: unknown): DictationIssue {
  const name =
    typeof error === "object" && error !== null && "name" in error && typeof error.name === "string"
      ? error.name
      : "";
  switch (name) {
    case "NotAllowedError":
    case "PermissionDeniedError":
      return "denied";
    case "SecurityError":
      return "blocked";
    case "NotFoundError":
    case "DevicesNotFoundError":
    case "OverconstrainedError":
      return "no-microphone";
    case "NotReadableError":
    case "TrackStartError":
    case "AbortError":
      return "microphone-busy";
    default:
      return "failed";
  }
}

/**
 * Joins what was typed with what was said: each part trimmed, empty parts
 * dropped, one space between. The typed part keeps its own inner spacing.
 */
export function joinSpoken(base: string, ...parts: string[]): string {
  const spoken = parts
    .map((part) => part.trim())
    .filter((part) => part !== "")
    .join(" ");
  const typed = base.trimEnd();
  if (spoken === "") {
    return typed;
  }

  return typed === "" ? spoken : typed + " " + spoken;
}

function releaseStream(stream: MediaStream | null) {
  stream?.getTracks().forEach((track) => track.stop());
}

type Session = {
  recognition: SpeechRecognition | null;
  stream: MediaStream | null;
  stoppedByPerson: boolean;
  issue: DictationIssue | null;
  graceTimer: number | null;
};

/**
 * Speaking into the composer.
 *
 * A click first asks for the microphone with getUserMedia, so the browser's
 * permission prompt appears and a refusal is reported as a refusal rather
 * than as silence; the stream is then held for the level meter and released
 * the moment listening ends. `onTranscript` receives each finished phrase,
 * `onInterim` the guess in progress, and `onEnd` fires once when a session
 * the person did not abort lets go, after its last phrase has landed.
 */
export function useDictation({
  lang,
  onTranscript,
  onInterim,
  onEnd,
  scope: scopeOverride,
}: {
  lang?: string;
  onTranscript: (text: string) => void;
  onInterim?: (text: string) => void;
  onEnd?: () => void;
  /** For tests: the page's dictation support, instead of reading it from window. */
  scope?: DictationScope;
}): DictationState {
  const [phase, setPhase] = useState<DictationPhase>("idle");
  const [issue, setIssue] = useState<DictationIssue | null>(null);
  const [stream, setStream] = useState<MediaStream | null>(null);
  const sessionRef = useRef<Session | null>(null);
  const callbacksRef = useRef({ onTranscript, onInterim, onEnd });
  useEffect(() => {
    callbacksRef.current = { onTranscript, onInterim, onEnd };
  });

  // What the page supports does not change while it is open, so it is read
  // once rather than on every render.
  const scope = useMemo(() => scopeOverride ?? readDictationScope(), [scopeOverride]);
  const availability = dictationAvailability(scope);

  const finish = useCallback(
    (session: Session, ending: { issue: DictationIssue | null; silent: boolean }) => {
      if (sessionRef.current !== session) {
        return;
      }
      sessionRef.current = null;
      if (session.graceTimer !== null) {
        window.clearTimeout(session.graceTimer);
      }
      releaseStream(session.stream);
      session.stream = null;
      setStream(null);
      setPhase("idle");
      if (ending.issue !== null) {
        setIssue(ending.issue);
      }
      if (!ending.silent) {
        callbacksRef.current.onEnd?.();
      }
    },
    [],
  );

  const abort = useCallback(() => {
    const session = sessionRef.current;
    if (!session) {
      return;
    }
    session.stoppedByPerson = true;
    session.recognition?.abort();
    finish(session, { issue: null, silent: true });
  }, [finish]);

  const stop = useCallback(() => {
    const session = sessionRef.current;
    if (!session) {
      return;
    }
    session.stoppedByPerson = true;
    if (session.recognition === null) {
      finish(session, { issue: null, silent: false });
      return;
    }
    // The recogniser hands back the phrase in progress after stop(), then
    // ends. The control answers the click now; the session stays open until
    // that last phrase lands, and a recogniser that never ends is let go.
    releaseStream(session.stream);
    session.stream = null;
    setStream(null);
    setPhase("idle");
    session.graceTimer = window.setTimeout(
      () => finish(session, { issue: null, silent: false }),
      STOP_GRACE_MS,
    );
    session.recognition.stop();
  }, [finish]);

  const begin = useCallback(
    (session: Session, Recognition: RecognitionConstructor) => {
      if (sessionRef.current !== session) {
        return;
      }
      const recognition = new Recognition();
      recognition.lang = lang ?? (typeof navigator === "undefined" ? "en-US" : navigator.language);
      recognition.continuous = true;
      recognition.interimResults = true;
      recognition.maxAlternatives = 1;
      recognition.onstart = () => {
        if (sessionRef.current === session && !session.stoppedByPerson) {
          setPhase("listening");
        }
      };
      recognition.onresult = (event) => {
        if (sessionRef.current !== session) {
          return;
        }
        // Every final in one event is committed together: calling back once
        // per final let a second phrase overwrite the first.
        let finished = "";
        let interim = "";
        for (let index = event.resultIndex; index < event.results.length; index += 1) {
          const result = event.results[index];
          const transcript = result[0]?.transcript ?? "";
          if (result.isFinal) {
            finished = joinSpoken(finished, transcript);
          } else {
            interim += transcript;
          }
        }
        if (finished !== "") {
          callbacksRef.current.onTranscript(finished);
        }
        callbacksRef.current.onInterim?.(interim.trim());
      };
      recognition.onerror = (event) => {
        if (sessionRef.current !== session) {
          return;
        }
        session.issue = issueFromRecognitionError(event.error, session.stoppedByPerson);
      };
      recognition.onend = () => {
        finish(session, { issue: session.issue, silent: false });
      };
      session.recognition = recognition;
      try {
        recognition.start();
      } catch {
        session.recognition = null;
        finish(session, { issue: "failed", silent: false });
      }
    },
    [finish, lang],
  );

  const start = useCallback(() => {
    const current = scope;
    const Recognition = current.recognition;
    if (dictationAvailability(current) !== "available" || Recognition === null) {
      return;
    }
    // A session still handing back its last phrase gives way to a new one.
    const previous = sessionRef.current;
    if (previous) {
      previous.stoppedByPerson = true;
      previous.recognition?.abort();
      finish(previous, { issue: null, silent: true });
    }
    const session: Session = {
      recognition: null,
      stream: null,
      stoppedByPerson: false,
      issue: null,
      graceTimer: null,
    };
    sessionRef.current = session;
    setIssue(null);
    setPhase("starting");

    if (current.getUserMedia === null) {
      begin(session, Recognition);
      return;
    }
    current.getUserMedia({ audio: true }).then(
      (opened) => {
        if (sessionRef.current !== session || session.stoppedByPerson) {
          releaseStream(opened);
          return;
        }
        session.stream = opened;
        setStream(opened);
        begin(session, Recognition);
      },
      (error: unknown) => {
        finish(session, { issue: issueFromMediaError(error), silent: false });
      },
    );
  }, [begin, finish, scope]);

  const toggle = useCallback(() => {
    if (sessionRef.current && phase !== "idle") {
      stop();
    } else {
      start();
    }
  }, [phase, start, stop]);

  const dismissIssue = useCallback(() => setIssue(null), []);

  useEffect(() => abort, [abort]);

  return {
    availability,
    supported: availability === "available",
    phase,
    listening: phase !== "idle",
    issue,
    stream,
    start,
    stop,
    abort,
    toggle,
    dismissIssue,
  };
}
