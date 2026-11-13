/**
 * What a developer types in the browser console to work on a page tour. Development builds
 * only: the store installs it behind `import.meta.env.DEV`, so a release has none of it.
 *
 *   tour.show()            open the tour of the page on screen, seen or not
 *   tour.show("key")       open one tour by key, when a page has more than one
 *   tour.always()          open every tour on every visit, as a first-time visitor sees it
 *   tour.always(false)     back to once per person
 *   tour.list()            the tours the page on screen can run
 */
export type TourConsole = {
  show: (tour?: string) => string;
  always: (on?: boolean) => string;
  list: () => readonly string[];
};

/** The part of the tour store the console drives. */
type TourControls = {
  getState: () => {
    available: readonly string[];
    request: (tour: string) => void;
    setAlways: (on: boolean) => void;
  };
};

declare global {
  interface Window {
    tour?: TourConsole;
  }
}

/**
 * Whether tours open on every visit. It is a switch on a developer's own browser, not
 * anything about a person, and it is only ever read in a development build.
 */
const ALWAYS_KEY = "trenova-dev-tour-always";

export function readTourAlways(): boolean {
  try {
    return window.localStorage.getItem(ALWAYS_KEY) === "true";
  } catch {
    return false;
  }
}

function keepTourAlways(on: boolean): void {
  try {
    if (on) window.localStorage.setItem(ALWAYS_KEY, "true");
    else window.localStorage.removeItem(ALWAYS_KEY);
  } catch {
    // Storage is blocked: the switch still holds until the page is reloaded.
  }
}

export function installTourConsole(store: TourControls): void {
  window.tour = {
    show: (tour) => {
      const { available, request } = store.getState();
      const key = tour ?? available[0];
      if (key === undefined) return "No tour on this page.";
      if (!available.includes(key)) {
        // i18n-ignore: a reply in the browser console of a development build, read by a developer
        return `No tour "${key}" on this page. Tours here: ${available.join(", ") || "none"}.`;
      }
      request(key);
      // i18n-ignore: a reply in the browser console of a development build, read by a developer
      return `Showing the "${key}" tour.`;
    },
    always: (on = true) => {
      keepTourAlways(on);
      store.getState().setAlways(on);
      return on
        ? "Tours open on every visit, seen or not. tour.always(false) turns this off."
        : "Tours open once per person again.";
    },
    list: () => store.getState().available,
  };
}
