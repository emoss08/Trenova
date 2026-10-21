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

export function stubLayout(height = 320, width = 480): () => void {
  const proto = HTMLElement.prototype;
  const size = { height, width };
  const properties = Object.keys(STUBBED) as StubbedProperty[];
  const previous = new Map(
    properties.map((property) => [property, Object.getOwnPropertyDescriptor(proto, property)]),
  );
  for (const property of properties) {
    Object.defineProperty(proto, property, {
      configurable: true,
      get: () => size[STUBBED[property]],
    });
  }

  return () => {
    for (const property of properties) {
      const descriptor = previous.get(property);
      if (descriptor) {
        Object.defineProperty(proto, property, descriptor);
      } else {
        Reflect.deleteProperty(proto, property);
      }
    }
  };
}
