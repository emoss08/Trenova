/**
 * A press held still on a touch screen, read as the right-click a touch
 * screen does not have.
 *
 * The press must stay within a few pixels for half a second: a finger that
 * moves is scrolling or swiping, and one lifted early is a tap. Once it
 * fires, the click the lifted finger would send is swallowed, so opening a
 * row's menu does not also open the row. A mouse never long-presses; it has a
 * right button.
 */
export const LONG_PRESS_MS = 500;
export const LONG_PRESS_TOLERANCE = 10;
/** How long after a long press its stray click (or the browser's own contextmenu) is ignored. */
export const LONG_PRESS_SWALLOW_MS = 800;

export type PressEvent = {
  pointerType: string;
  clientX: number;
  clientY: number;
  button: number;
};

export type LongPressOptions<C = undefined> = {
  /** `context` is whatever the press that fired was started with, such as the element pressed. */
  onLongPress: (point: { x: number; y: number }, context: C | undefined) => void;
  delay?: number;
  tolerance?: number;
  schedule?: (callback: () => void, ms: number) => unknown;
  cancel?: (handle: unknown) => void;
  now?: () => number;
};

export type LongPressTracker<C = undefined> = {
  down: (event: PressEvent, context?: C) => void;
  move: (event: PressEvent) => void;
  up: () => void;
  cancel: () => void;
  /** True once for the click a long press leaves behind; that click should be ignored. */
  consumeClick: () => boolean;
  /** Whether a long press fired recently enough that a contextmenu event is its echo. */
  justFired: () => boolean;
};

export function createLongPress<C = undefined>({
  onLongPress,
  delay = LONG_PRESS_MS,
  tolerance = LONG_PRESS_TOLERANCE,
  schedule = (callback, ms) => setTimeout(callback, ms),
  cancel: clear = (handle) => clearTimeout(handle as ReturnType<typeof setTimeout>),
  now = () => Date.now(),
}: LongPressOptions<C>): LongPressTracker<C> {
  let timer: unknown = null;
  let origin: { x: number; y: number } | null = null;
  let firedAt = Number.NEGATIVE_INFINITY;
  let swallowClick = false;

  const stop = () => {
    if (timer !== null) {
      clear(timer);
      timer = null;
    }
    origin = null;
  };

  return {
    down: (event, context) => {
      stop();
      if (event.pointerType === "mouse" || event.button !== 0) {
        return;
      }
      const start = { x: event.clientX, y: event.clientY };
      origin = start;
      timer = schedule(() => {
        timer = null;
        origin = null;
        firedAt = now();
        swallowClick = true;
        onLongPress(start, context);
      }, delay);
    },
    move: (event) => {
      if (origin === null) {
        return;
      }
      if (Math.hypot(event.clientX - origin.x, event.clientY - origin.y) > tolerance) {
        stop();
      }
    },
    up: stop,
    cancel: stop,
    consumeClick: () => {
      if (!swallowClick) {
        return false;
      }
      swallowClick = false;
      return now() - firedAt <= LONG_PRESS_SWALLOW_MS;
    },
    justFired: () => now() - firedAt <= LONG_PRESS_SWALLOW_MS,
  };
}
