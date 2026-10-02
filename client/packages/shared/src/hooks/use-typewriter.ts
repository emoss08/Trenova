import { useEffect, useReducer } from "react";

export type TypewriterPhase = "typing" | "holding" | "erasing";

export type TypewriterOptions = {
  /** Milliseconds between one character arriving and the next. */
  typeMs?: number;
  /** How long a finished line stands before it is taken back. */
  holdMs?: number;
  /** Milliseconds between one character leaving and the next. */
  eraseMs?: number;
  /** The pause between one line's last character leaving and the next line's first arriving. */
  gapMs?: number;
  /** When set, every line is shown whole and only the hold and the swap remain. */
  instant?: boolean;
  /** Stops the clock and holds whatever is showing. */
  paused?: boolean;
};

export type TypewriterState = {
  /** The characters on screen right now. */
  text: string;
  /** Which of the lines is being written, held or erased. */
  index: number;
  phase: TypewriterPhase;
  /** True once the whole current line is on screen, so it can be accepted as it stands. */
  complete: boolean;
};

type Action = { type: "tick" } | { type: "reset" };

type Internal = { index: number; length: number; phase: TypewriterPhase };

function step(state: Internal, lines: readonly string[], instant: boolean): Internal {
  const line = lines[state.index] ?? "";
  switch (state.phase) {
    case "typing": {
      const length = instant ? line.length : state.length + 1;
      return length >= line.length
        ? { ...state, length: line.length, phase: "holding" }
        : { ...state, length };
    }
    case "holding":
      return lines.length <= 1 ? state : { ...state, phase: "erasing" };
    default: {
      const length = instant ? 0 : state.length - 1;
      return length <= 0
        ? { index: (state.index + 1) % lines.length, length: 0, phase: "typing" }
        : { ...state, length };
    }
  }
}

function delayFor(state: Internal, options: Required<Omit<TypewriterOptions, "paused">>): number {
  switch (state.phase) {
    case "typing":
      if (options.instant) {
        return state.length === 0 ? options.gapMs : options.holdMs;
      }
      return state.length === 0 ? options.gapMs : options.typeMs;
    case "holding":
      return options.holdMs;
    default:
      return options.instant ? options.gapMs : options.eraseMs;
  }
}

/**
 * Writes a set of lines one character at a time, holds each, takes it back
 * and writes the next, round and round; a line is complete once every
 * character stands. The clock is a chain of single timeouts so each phase
 * sets its own pace, and a change of lines starts over from the first.
 */
export function useTypewriter(
  lines: readonly string[],
  options: TypewriterOptions = {},
): TypewriterState {
  const {
    typeMs = 26,
    holdMs = 2800,
    eraseMs = 11,
    gapMs = 360,
    instant = false,
    paused = false,
  } = options;
  const key = lines.join("\u0000");

  const [state, dispatch] = useReducer(
    (current: Internal & { key: string }, action: Action) => {
      if (action.type === "reset") {
        return { key, index: 0, length: 0, phase: "typing" as const };
      }
      return { ...step(current, lines, instant), key: current.key };
    },
    { key, index: 0, length: 0, phase: "typing" as const },
  );

  useEffect(() => {
    dispatch({ type: "reset" });
    // The lines are read through their joined key so a new array with the same words is not a restart.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key]);

  const line = lines[state.index] ?? "";
  const settled = lines.length <= 1 && state.phase === "holding";

  useEffect(() => {
    if (paused || lines.length === 0 || settled || state.key !== key) {
      return;
    }
    const delay = delayFor(state, { typeMs, holdMs, eraseMs, gapMs, instant });
    const timer = setTimeout(() => dispatch({ type: "tick" }), delay);
    return () => clearTimeout(timer);
  }, [paused, lines, settled, state, key, typeMs, holdMs, eraseMs, gapMs, instant]);

  if (state.key !== key || lines.length === 0) {
    return { text: "", index: 0, phase: "typing", complete: false };
  }

  return {
    text: line.slice(0, state.length),
    index: state.index,
    phase: state.phase,
    complete: state.length >= line.length && line.length > 0,
  };
}
