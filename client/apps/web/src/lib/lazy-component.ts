import { lazy, type FunctionComponent, type LazyExoticComponent } from "react";

/**
 * A module read once however often it is asked for: every call shares the
 * one import, and a failed one is forgotten so the next call asks again
 * rather than replaying the failure for the rest of the session.
 */
export function preloadable<M>(load: () => Promise<M>): () => Promise<M> {
  let loading: Promise<M> | null = null;
  return () => {
    loading ??= load().catch((error: unknown) => {
      loading = null;
      throw error;
    });
    return loading;
  };
}

/** A component whose code is read when it is first drawn, or earlier when asked. */
export type LazyComponent<P extends object> = {
  Component: LazyExoticComponent<FunctionComponent<P>>;
  /** Reads the code ahead of the first drawing; the drawing then waits for nothing. */
  preload: () => Promise<unknown>;
};

/**
 * One component of a module that is read on demand. The drawing and an
 * earlier preload share one import, so the module is never read twice.
 */
export function lazyComponent<M, P extends object>(
  load: () => Promise<M>,
  pick: (module: M) => FunctionComponent<P>,
): LazyComponent<P> {
  const preload = preloadable(load);
  return {
    Component: lazy(() => preload().then((module) => ({ default: pick(module) }))),
    preload,
  };
}
