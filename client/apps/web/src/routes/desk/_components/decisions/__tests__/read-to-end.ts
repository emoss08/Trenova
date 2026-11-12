import { vi } from "vitest";

/**
 * The test environment's IntersectionObserver never reports, so a batch
 * sheet would never see its list read to the end. This one reports every
 * element it is given as in view at once, as a list short enough to fit does.
 */
export function stubListReadToEnd() {
  vi.stubGlobal(
    "IntersectionObserver",
    class {
      callback: IntersectionObserverCallback;
      constructor(callback: IntersectionObserverCallback) {
        this.callback = callback;
      }
      observe(target: Element) {
        this.callback(
          [{ isIntersecting: true, target } as unknown as IntersectionObserverEntry],
          this as unknown as IntersectionObserver,
        );
      }
      disconnect() {}
    },
  );
}
