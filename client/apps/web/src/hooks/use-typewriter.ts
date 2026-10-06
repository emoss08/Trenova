import { useReducedMotion } from "motion/react";
import { useEffect, useRef, useState } from "react";

export const TYPEWRITER_THINK_MS = 650;
export const TYPEWRITER_SETTLE_MS = 180;

/**
 * How long to wait after revealing `char` before the next one: a short random beat,
 * longer after the end of a sentence, a clause or a line, so a typed line reads at the
 * pace a person would type it.
 */
export function typewriterDelay(char: string, random: number): number {
  let delay = 12 + random * 18;
  if (char === "." || char === "?" || char === "!") {
    delay += 260;
  } else if (char === "," || char === "—") {
    delay += 110;
  } else if (char === "\n") {
    delay += 200;
  }
  return delay;
}

export type TypewriterState = {
  revealed: number;
  thinking: boolean;
  typing: boolean;
};

/**
 * Reveals `text` one character at a time after a short "Typing" pause, then calls
 * `onDone`. With `animate` off, or when the person prefers reduced motion, the whole
 * text shows at once and `onDone` still fires, so a line that was already typed never
 * types again. The text is read once, when the line starts: a line types what it was
 * asked to type.
 */
export function useTypewriter(
  text: string,
  { animate, onDone }: { animate: boolean; onDone?: () => void },
): TypewriterState {
  const reduceMotion = useReducedMotion() ?? false;
  const shouldAnimate = animate && !reduceMotion;
  const [startedAnimated] = useState(shouldAnimate);
  const [revealed, setRevealed] = useState(startedAnimated ? 0 : text.length);
  const [thinking, setThinking] = useState(startedAnimated);
  const doneRef = useRef(onDone);
  const textRef = useRef(text);

  useEffect(() => {
    doneRef.current = onDone;
  }, [onDone]);

  useEffect(() => {
    if (!startedAnimated) {
      doneRef.current?.();
      return undefined;
    }

    const full = textRef.current;
    let alive = true;
    let index = 0;
    let timer: ReturnType<typeof setTimeout>;

    const step = () => {
      if (!alive) {
        return;
      }
      index += 1;
      setRevealed(index);
      if (index >= full.length) {
        timer = setTimeout(() => {
          if (alive) {
            doneRef.current?.();
          }
        }, TYPEWRITER_SETTLE_MS);
        return;
      }
      timer = setTimeout(step, typewriterDelay(full[index - 1], Math.random()));
    };

    timer = setTimeout(() => {
      if (!alive) {
        return;
      }
      setThinking(false);
      step();
    }, TYPEWRITER_THINK_MS);

    return () => {
      alive = false;
      clearTimeout(timer);
    };
  }, [startedAnimated]);

  if (!startedAnimated) {
    return { revealed: text.length, thinking: false, typing: false };
  }

  return {
    revealed: Math.min(revealed, text.length),
    thinking,
    typing: !thinking && revealed < text.length,
  };
}
