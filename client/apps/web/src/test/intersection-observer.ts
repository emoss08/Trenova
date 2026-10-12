import { vi } from "vitest";

/**
 * An IntersectionObserver a test drives by hand: nothing is in view until
 * the test says so with `show`. Install it with `installIntersectionObserver`
 * in a `beforeEach`; `vi.unstubAllGlobals` restores the runner's own.
 */
export class FakeIntersectionObserver {
  static instances: FakeIntersectionObserver[] = [];

  readonly targets = new Set<Element>();
  readonly options: IntersectionObserverInit;
  disconnected = false;
  private readonly callback: IntersectionObserverCallback;

  constructor(callback: IntersectionObserverCallback, options: IntersectionObserverInit = {}) {
    this.callback = callback;
    this.options = options;
    FakeIntersectionObserver.instances.push(this);
  }

  observe(target: Element) {
    this.targets.add(target);
  }

  unobserve(target: Element) {
    this.targets.delete(target);
  }

  disconnect() {
    this.targets.clear();
    this.disconnected = true;
  }

  takeRecords(): IntersectionObserverEntry[] {
    return [];
  }

  /** Brings the element into view (or takes it out) for every observer watching it. */
  static show(target: Element, visible = true) {
    for (const observer of FakeIntersectionObserver.instances) {
      if (!observer.targets.has(target)) continue;
      observer.callback(
        [{ target, isIntersecting: visible } as IntersectionObserverEntry],
        observer as unknown as IntersectionObserver,
      );
    }
  }
}

export function installIntersectionObserver() {
  FakeIntersectionObserver.instances = [];
  vi.stubGlobal("IntersectionObserver", FakeIntersectionObserver);
}
