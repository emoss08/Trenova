/**
 * jsdom lays nothing out, so every box measures zero and a windowed list
 * renders no rows at all. This gives every element one size, which is
 * enough for a virtualizer to open a window: the container and the rows
 * are the same height, so a viewport holds one row plus its overscan.
 *
 * The virtualizer reads the scroll element's `clientWidth` and
 * `clientHeight` and a row's `offsetWidth` and `offsetHeight`, which the
 * test DOM defines as zero on the prototype, so those are what change.
 */
const STUBBED = {
  offsetHeight: "height",
  offsetWidth: "width",
  clientHeight: "height",
  clientWidth: "width",
} as const;

type StubbedProperty = keyof typeof STUBBED;

/**
 * How far an element scrolls: as far as the tallest (or widest) box sized inside it, which
 * is a virtualizer's body, and never less than the element itself. A virtualizer clamps a
 * scroll to `scrollHeight - clientHeight`, so without this nothing can be scrolled to.
 */
const SCROLL_EXTENT = { scrollHeight: "height", scrollWidth: "width" } as const;

type ScrollExtentProperty = keyof typeof SCROLL_EXTENT;

function contentExtent(element: HTMLElement, axis: "height" | "width", own: number): number {
  let extent = own;
  for (const child of element.querySelectorAll<HTMLElement>("[style]")) {
    const value = Number.parseFloat(child.style[axis]);
    if (Number.isFinite(value)) {
      extent = Math.max(extent, value);
    }
  }
  return extent;
}

export function stubLayout(height = 320, width = 480): () => void {
  const proto = HTMLElement.prototype;
  const size = { height, width };
  const properties = Object.keys(STUBBED) as StubbedProperty[];
  const extents = Object.keys(SCROLL_EXTENT) as ScrollExtentProperty[];
  const previous = new Map(
    [...properties, ...extents].map((property) => [
      property,
      Object.getOwnPropertyDescriptor(proto, property),
    ]),
  );
  for (const property of properties) {
    Object.defineProperty(proto, property, {
      configurable: true,
      get: () => size[STUBBED[property]],
    });
  }
  for (const property of extents) {
    const axis = SCROLL_EXTENT[property];
    Object.defineProperty(proto, property, {
      configurable: true,
      get(this: HTMLElement) {
        return contentExtent(this, axis, size[axis]);
      },
    });
  }

  return () => {
    for (const property of [...properties, ...extents]) {
      const descriptor = previous.get(property);
      if (descriptor) {
        Object.defineProperty(proto, property, descriptor);
      } else {
        Reflect.deleteProperty(proto, property);
      }
    }
  };
}

/**
 * jsdom has no `Element.scrollTo`, so a virtualizer asked to scroll a row into view does
 * nothing. This moves `scrollTop` and fires the scroll event a browser would, so the
 * window follows exactly as it does on screen.
 */
export function stubScrollTo(): () => void {
  const proto = HTMLElement.prototype;
  const previous = Object.getOwnPropertyDescriptor(proto, "scrollTo");
  Object.defineProperty(proto, "scrollTo", {
    configurable: true,
    writable: true,
    value(this: HTMLElement, options?: ScrollToOptions) {
      if (options?.top === undefined) return;
      this.scrollTop = options.top;
      this.dispatchEvent(new Event("scroll"));
    },
  });

  return () => {
    if (previous) {
      Object.defineProperty(proto, "scrollTo", previous);
    } else {
      Reflect.deleteProperty(proto, "scrollTo");
    }
  };
}
