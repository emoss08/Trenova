import { useElementSize } from "./use-element-size";

/**
 * The content width of an element, kept current by a ResizeObserver so a chart drawn in
 * pixels redraws when its column narrows. Zero until the element has been measured.
 */
export function useElementWidth<T extends HTMLElement>(): [(node: T | null) => void, number] {
  const [ref, size] = useElementSize<T>();
  return [ref, size.width];
}
