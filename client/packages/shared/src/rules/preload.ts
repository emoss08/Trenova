// Starts loading rules.wasm once the page has loaded and the browser is idle, so a screen
// that needs it later (the formula editor, a rate screen) finds it ready. It never competes
// with first paint: it waits for the load event and then for idle time, and the loader itself
// is imported only then, so nothing of it is in the startup bundle. A failure is logged and
// forgotten; useRules() and loadRules() try again when something needs the module.

const IDLE_TIMEOUT_MS = 10_000;
const NO_IDLE_CALLBACK_DELAY_MS = 2_000;

type PreloadEnvironment = {
  load: () => Promise<unknown>;
  whenIdle: (task: () => void) => void;
  whenLoaded: (task: () => void) => void;
  report: (error: unknown) => void;
};

function browserEnvironment(): PreloadEnvironment {
  return {
    load: () => import("./load").then(({ loadRules }) => loadRules()),
    whenIdle: (task) => {
      if (typeof window.requestIdleCallback === "function") {
        window.requestIdleCallback(task, { timeout: IDLE_TIMEOUT_MS });
      } else {
        window.setTimeout(task, NO_IDLE_CALLBACK_DELAY_MS);
      }
    },
    whenLoaded: (task) => {
      if (document.readyState === "complete") task();
      else window.addEventListener("load", task, { once: true });
    },
    report: (error) =>
      console.error(
        "Could not preload the rules module; it loads again when a screen needs it:",
        error,
      ),
  };
}

/** Loads the rules module after the page has loaded, when the browser is idle. */
export function preloadRulesWhenIdle(environment: PreloadEnvironment = browserEnvironment()): void {
  environment.whenLoaded(() =>
    environment.whenIdle(() => {
      environment.load().catch(environment.report);
    }),
  );
}
