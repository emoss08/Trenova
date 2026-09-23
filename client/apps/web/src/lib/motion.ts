/**
 * The house curves as motion's cubic-bezier tuples, for the places a
 * transition is driven from script rather than from a class. They are the
 * same curves as `--ease-swift`, `--ease-settle` and `--ease-spring` in tokens.css, so a
 * layout animation and a CSS one arrive the same way.
 */
export const EASE_SWIFT = [0.2, 0.8, 0.2, 1] as const;
export const EASE_SETTLE = [0.16, 1, 0.3, 1] as const;
export const EASE_SPRING = [0.34, 1.4, 0.64, 1] as const;
