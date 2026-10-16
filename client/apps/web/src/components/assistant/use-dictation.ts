import { markSpeechServiceMissing, useSpeechServiceMissing } from "@/lib/speech-service";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

/**
 * Whether dictation can run here at all, and if not, which of the reasons a
 * person could be told about. Each has a different remedy, so they are not
 * folded into one "unsupported". "no-service" is a browser with the speech
 * API and nothing behind it, found by an attempt that failed
 * (lib/speech-service.ts).
 */
export type DictationAvailability =
  | "available"
  | "unsupported"
  | "insecure"
  | "blocked"
  | "no-service";

/**
 * Why an attempt ended without the person ending it. Each maps to one
 * sentence the composer shows, because "nothing happens" is what a person
 * sees when these are swallowed.
 */
export type DictationIssue =
  | "denied"
  | "blocked"
  | "service"
  | "no-service"
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

/**
 * A recogniser that ends this soon after it started did not end on a pause:
 * it could not listen at all, as on a browser without a speech service.
 */
const QUICK_END_MS = 1000;

/** How many quick ends in a row mean the recogniser cannot listen here, rather than a blip. */
const MAX_QUICK_ENDS = 3;

/** How long the microphone stays open with nothing heard before dictation lets it go. */
const MAX_SILENCE_MS = 30_000;

/**
 * What a recogniser that ended without the person stopping it can be
 * started again after. Browsers end a continuous session on their own after
 * a pause, at a time limit, or on a dropped connection; the person is still
 * dictating. Anything else, such as a refused or missing microphone, is the
 * end of the session.
 */
const RESUMABLE: ReadonlySet<DictationIssue | null> = new Set([null, "no-speech", "network"]);

/** How a recogniser that ended on its own was doing, for deciding whether to listen again. */
export type RecognitionEnding = {
  issue: DictationIssue | null;
  /** How long it listened, in milliseconds. */
  ranFor: number;
  /** How many recognisers before it in a row ended quickly. */
  quickEnds: number;
  /** How long since anything was heard, in milliseconds. */
  silentFor: number;
  /** Whether the browser says it is online: a missing service, not a lost connection. */
  online: boolean;
};

/**
 * Whether dictation listens again after a recogniser ended without the
 * person stopping it, or why it stops: what it ended on, unless that is one
 * a fresh recogniser gets past; a recogniser that keeps ending as soon as it
 * starts; or a microphone left open with nothing said. One that keeps failing
 * on the network while the browser is online has no speech service behind
 * it, as in Helium, Brave and Arc.
 */
export function afterRecognitionEnds(
  ending: RecognitionEnding,
): { resume: true; quickEnds: number } | { resume: false; issue: DictationIssue | null } {
  if (!RESUMABLE.has(ending.issue)) {
    return { resume: false, issue: ending.issue };
  }
  const quickEnds = ending.ranFor < QUICK_END_MS ? ending.quickEnds + 1 : 0;
  if (quickEnds >= MAX_QUICK_ENDS) {
    if (ending.issue === "network" && ending.online) {
      return { resume: false, issue: "no-service" };
    }
    return { resume: false, issue: ending.issue ?? "failed" };
  }
  if (ending.silentFor >= MAX_SILENCE_MS) {
    return { resume: false, issue: "no-speech" };
  }

  return { resume: true, quickEnds };
}

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

export function dictationAvailability(
  scope: DictationScope,
  serviceMissing = false,
): DictationAvailability {
  if (scope.recognition === null) {
    return "unsupported";
  }
  if (!scope.secure) {
    return "insecure";
  }
  if (scope.policyAllowsMicrophone === false) {
    return "blocked";
  }

  return serviceMissing ? "no-service" : "available";
}

/**
 * The issues that mean this browser cannot recognise speech at all, rather
 * than that one attempt failed: the microphone is hidden here from then on.
 */
export function meansNoSpeechService(issue: DictationIssue | null): boolean {
  return issue === "no-service" || issue === "service";
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
  /** How many recognisers in a row ended as soon as they started. */
  quickEnds: number;
  /** When something was last heard, or the session began; epoch milliseconds. */
  heardAt: number;
  /** The guess in progress, kept if its recogniser ends before it is final. */
  interim: string;
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
 *
 * A session lasts until the person stops it. Browsers end a continuous
 * recogniser on their own, after a pause or a minute or a dropped
 * connection; a fresh one takes over on the same microphone, so dictation
 * does not stop mid-thought (afterRecognitionEnds).
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
  const serviceMissing = useSpeechServiceMissing();
  const sessionRef = useRef<Session | null>(null);
  const callbacksRef = useRef({ onTranscript, onInterim, onEnd });
  useEffect(() => {
    callbacksRef.current = { onTranscript, onInterim, onEnd };
  });

  // What the page supports does not change while it is open, so it is read
  // once rather than on every render.
  const scope = useMemo(() => scopeOverride ?? readDictationScope(), [scopeOverride]);
  const availability = dictationAvailability(scope, serviceMissing);

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
      if (meansNoSpeechService(ending.issue)) {
        markSpeechServiceMissing();
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
      const listen = () => {
        if (sessionRef.current !== session) {
          return;
        }
        const recognition = new Recognition();
        recognition.lang =
          lang ?? (typeof navigator === "undefined" ? "en-US" : navigator.language);
        recognition.continuous = true;
        recognition.interimResults = true;
        recognition.maxAlternatives = 1;
        const startedAt = Date.now();
        recognition.onstart = () => {
          if (sessionRef.current === session && !session.stoppedByPerson) {
            setPhase("listening");
          }
        };
        recognition.onresult = (event) => {
          if (sessionRef.current !== session || session.recognition !== recognition) {
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
          if (finished !== "" || interim.trim() !== "") {
            session.heardAt = Date.now();
          }
          session.interim = interim.trim();
          if (finished !== "") {
            callbacksRef.current.onTranscript(finished);
          }
          callbacksRef.current.onInterim?.(session.interim);
        };
        recognition.onerror = (event) => {
          if (sessionRef.current !== session || session.recognition !== recognition) {
            return;
          }
          session.issue = issueFromRecognitionError(event.error, session.stoppedByPerson);
        };
        recognition.onend = () => {
          if (sessionRef.current !== session || session.recognition !== recognition) {
            return;
          }
          if (session.stoppedByPerson) {
            finish(session, { issue: session.issue, silent: false });
            return;
          }
          const now = Date.now();
          const next = afterRecognitionEnds({
            issue: session.issue,
            ranFor: now - startedAt,
            quickEnds: session.quickEnds,
            silentFor: now - session.heardAt,
            online: typeof navigator === "undefined" || navigator.onLine !== false,
          });
          if (!next.resume) {
            finish(session, { issue: next.issue, silent: false });
            return;
          }
          // The guess it had not finished is kept as said: the recogniser
          // that heard it is gone and will not finish it.
          if (session.interim !== "") {
            callbacksRef.current.onTranscript(session.interim);
            callbacksRef.current.onInterim?.("");
            session.interim = "";
          }
          session.quickEnds = next.quickEnds;
          session.issue = null;
          listen();
        };
        session.recognition = recognition;
        try {
          recognition.start();
        } catch {
          session.recognition = null;
          finish(session, { issue: "failed", silent: false });
        }
      };
      listen();
    },
    [finish, lang],
  );

  const start = useCallback(() => {
    const current = scope;
    const Recognition = current.recognition;
    if (dictationAvailability(current, serviceMissing) !== "available" || Recognition === null) {
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
      quickEnds: 0,
      heardAt: Date.now(),
      interim: "",
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
  }, [begin, finish, scope, serviceMissing]);

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
