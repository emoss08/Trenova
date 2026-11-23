import { createElement, lazy, useState, type FunctionComponent } from "react";

const PRELOADABLE = new WeakSet<() => Promise<unknown>>();

/**
 * A module read once however often it is asked for: every call shares the
 * one import, and a failed one is forgotten so the next call asks again
 * rather than replaying the failure for the rest of the session. Wrapping a
 * loader that is already preloadable returns it unchanged, so two owners of
 * one module (a registry of loaders and the lazy piece built on one) share
 * one read.
 */
export function preloadable<M>(load: () => Promise<M>): () => Promise<M> {
  if (PRELOADABLE.has(load)) return load;
  let loading: Promise<M> | null = null;
  const read = () => {
    loading ??= load().catch((error: unknown) => {
      loading = null;
      throw error;
    });
    return loading;
  };
  PRELOADABLE.add(read);
  return read;
}

/** A component whose code is read when it is first drawn, or earlier when asked. */
export type LazyComponent<P extends object> = {
  Component: FunctionComponent<P>;
  /** Reads the code ahead of the first drawing; the drawing then waits for nothing. */
  preload: () => Promise<unknown>;
};

/**
 * One component of a module that is read on demand. The drawing and an
 * earlier preload share one import, so the module is never read twice. An
 * instance mounted after the module has arrived draws it straight away; one
 * mounted before suspends to the nearest `Suspense` until it arrives, and a
 * failed read throws to the nearest error boundary, which asks again on retry.
 */
export function lazyComponent<M, P extends object>(
  load: () => Promise<M>,
  pick: (module: M) => FunctionComponent<P>,
): LazyComponent<P> {
  let ready: FunctionComponent<P> | null = null;
  const preload = preloadable(load);
  const read = () =>
    preload().then((module) => {
      ready = pick(module);
      return module;
    });
  // React keeps a lazy component's failure for good, so a failed read gets a fresh one: the
  // instance an error boundary mounts on retry suspends on a new read instead of the old error.
  const suspending = () =>
    lazy(() =>
      read().then(
        (module) => ({ default: pick(module) }),
        (error: unknown) => {
          pending = suspending();
          throw error;
        },
      ),
    );
  let pending = suspending();
  function Lazy(props: P) {
    "use no memo";
    // Chosen once per instance: moving from the suspending path to the direct one would remount
    // what is drawn and lose its state.
    const [drawn] = useState<FunctionComponent<P>>(() => ready ?? pending);
    return createElement(drawn, props);
  }
  return {
    Component: Lazy,
    preload: () => read(),
  };
}
