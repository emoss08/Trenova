import { useCallback, useEffect, useRef } from "react";
import {
  joinSpoken,
  useDictation,
  type DictationScope,
  type DictationState,
} from "./use-dictation";

type SpokenSession = {
  /** The draft as it stood when the person started speaking. */
  base: string;
  /** Every finished phrase so far. */
  committed: string;
  /** Whether anything was heard, so a silent session leaves the draft alone. */
  heard: boolean;
};

export type ComposerDictation = DictationState & {
  /** Starts listening from the draft as it stands, or stops. */
  toggleFromDraft: () => void;
  /** The person took over (typed, sent): stop without writing anything more. */
  release: () => void;
};

/**
 * Dictation written straight into the draft.
 *
 * The words appear in the box as they are recognised: the guess in progress
 * is shown after what has been said and typed, and replaced as the guess
 * firms up, so what the person sees is what will be sent. Typing or sending
 * while listening hands the box back to the keyboard, keeping what is on
 * screen and dropping anything the recogniser had not yet said.
 */
export function useComposerDictation({
  draft,
  onDraftChange,
  scope,
}: {
  draft: string;
  onDraftChange: (draft: string) => void;
  scope?: DictationScope;
}): ComposerDictation {
  const draftRef = useRef(draft);
  const onDraftChangeRef = useRef(onDraftChange);
  useEffect(() => {
    draftRef.current = draft;
    onDraftChangeRef.current = onDraftChange;
  });
  const sessionRef = useRef<SpokenSession | null>(null);

  const dictation = useDictation({
    scope,
    onTranscript: (text) => {
      const session = sessionRef.current;
      if (!session) {
        return;
      }
      session.committed = joinSpoken(session.committed, text);
      session.heard = true;
      onDraftChangeRef.current(joinSpoken(session.base, session.committed));
    },
    onInterim: (text) => {
      const session = sessionRef.current;
      if (!session || (text === "" && !session.heard)) {
        return;
      }
      session.heard = true;
      onDraftChangeRef.current(joinSpoken(session.base, session.committed, text));
    },
    onEnd: () => {
      const session = sessionRef.current;
      sessionRef.current = null;
      if (session?.heard) {
        onDraftChangeRef.current(joinSpoken(session.base, session.committed));
      }
    },
  });

  const { abort, phase, start, stop, supported } = dictation;

  const toggleFromDraft = useCallback(() => {
    if (phase !== "idle") {
      stop();
      return;
    }
    if (!supported) {
      return;
    }
    sessionRef.current = { base: draftRef.current, committed: "", heard: false };
    start();
  }, [phase, start, stop, supported]);

  const release = useCallback(() => {
    sessionRef.current = null;
    abort();
  }, [abort]);

  return { ...dictation, toggleFromDraft, release };
}
