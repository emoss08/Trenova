/** Where one toast sits in its stack, relative to the edge the stack grows from. */
export type ToastSlot = {
  y: number;
  scale: number;
  height: number;
  /** Peeking out from under the front toast, with its content hidden. */
  behind: boolean;
  /** Past what the stack shows; still counting down, but not drawn. */
  hidden: boolean;
};

export type ToastStackInput = {
  /** Natural heights, newest first; 0 for a toast not measured yet. */
  heights: readonly number[];
  expanded: boolean;
  /** 1 when the stack hangs from the top of the screen, -1 when it rises from the bottom. */
  direction: 1 | -1;
  gap: number;
  peek: number;
  /** How many toasts a collapsed stack draws. */
  visible: number;
  /** How many toasts an expanded stack draws. */
  expandedVisible: number;
};

export type ToastStack = {
  slots: ToastSlot[];
  /** The height the stack covers, so hovering anywhere over it keeps it open. */
  height: number;
};

const DEPTH_SCALE_STEP = 0.05;

export function layoutToastStack({
  heights,
  expanded,
  direction,
  gap,
  peek,
  visible,
  expandedVisible,
}: ToastStackInput): ToastStack {
  const slots: ToastSlot[] = new Array(heights.length);
  const front = heights[0] ?? 0;

  if (expanded) {
    let offset = 0;
    let covered = 0;
    for (let i = 0; i < heights.length; i += 1) {
      const hidden = i >= expandedVisible;
      slots[i] = {
        y: offset * direction || 0,
        scale: 1,
        height: heights[i],
        behind: false,
        hidden,
      };
      if (!hidden) {
        covered = offset + heights[i];
        offset += heights[i] + gap;
      }
    }
    return { slots, height: covered };
  }

  for (let i = 0; i < heights.length; i += 1) {
    const depth = Math.min(i, visible - 1);
    slots[i] = {
      y: depth * peek * direction || 0,
      scale: 1 - depth * DEPTH_SCALE_STEP,
      height: i === 0 ? heights[i] : front,
      behind: i > 0,
      hidden: i >= visible,
    };
  }
  const drawn = Math.min(heights.length, visible);
  return { slots, height: drawn === 0 ? 0 : front + (drawn - 1) * peek };
}
