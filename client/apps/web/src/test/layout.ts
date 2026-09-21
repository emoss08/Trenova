/**
 * jsdom lays nothing out, so every box measures zero and a windowed list
 * renders no rows at all. This gives every element one size, which is
 * enough for a virtualizer to open a window: the container and the rows
 * are the same height, so a viewport holds one row plus its overscan.
 *
 * The virtualizer reads `offsetWidth` and `offsetHeight`, which jsdom
 * defines as zero on the prototype, so those are what change.
 */
export function stubLayout(height = 320, width = 480): () => void {
  const proto = HTMLElement.prototype;
  const previous = {
    height: Object.getOwnPropertyDescriptor(proto, "offsetHeight"),
    width: Object.getOwnPropertyDescriptor(proto, "offsetWidth"),
  };
  Object.defineProperty(proto, "offsetHeight", { configurable: true, get: () => height });
  Object.defineProperty(proto, "offsetWidth", { configurable: true, get: () => width });

  return () => {
    if (previous.height) {
      Object.defineProperty(proto, "offsetHeight", previous.height);
    } else {
      delete (proto as { offsetHeight?: number }).offsetHeight;
    }
    if (previous.width) {
      Object.defineProperty(proto, "offsetWidth", previous.width);
    } else {
      delete (proto as { offsetWidth?: number }).offsetWidth;
    }
  };
}
