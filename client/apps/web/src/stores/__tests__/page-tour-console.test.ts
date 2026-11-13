import { beforeEach, describe, expect, it } from "vitest";
import { readTourAlways } from "../page-tour-console";
import { usePageTourStore } from "../page-tour-store";

describe("the tour console", () => {
  beforeEach(() => {
    window.localStorage.clear();
    usePageTourStore.setState({ available: [], requested: null, showing: null, always: false });
  });

  it("is on the window in a development build", () => {
    expect(window.tour).toBeDefined();
  });

  it("opens the tour of the page on screen", () => {
    usePageTourStore.setState({ available: ["freight-outlook"] });

    expect(window.tour!.show()).toBe('Showing the "freight-outlook" tour.');
    expect(usePageTourStore.getState().requested).toBe("freight-outlook");
  });

  it("opens one tour by key on a page that has several", () => {
    usePageTourStore.setState({ available: ["console", "console-lanes"] });

    expect(window.tour!.show("console-lanes")).toBe('Showing the "console-lanes" tour.');
    expect(usePageTourStore.getState().requested).toBe("console-lanes");
  });

  it("says so instead of asking for a tour the page cannot run", () => {
    expect(window.tour!.show()).toBe("No tour on this page.");
    expect(window.tour!.show("freight-outlook")).toBe(
      'No tour "freight-outlook" on this page. Tours here: none.',
    );

    usePageTourStore.setState({ available: ["console"] });
    expect(window.tour!.show("freight-outlook")).toBe(
      'No tour "freight-outlook" on this page. Tours here: console.',
    );
    expect(usePageTourStore.getState().requested).toBeNull();
  });

  it("lists the tours the page on screen can run", () => {
    usePageTourStore.setState({ available: ["freight-outlook"] });

    expect(window.tour!.list()).toEqual(["freight-outlook"]);
  });

  it("turns tours on for every visit, and keeps the switch across reloads", () => {
    expect(window.tour!.always()).toBe(
      "Tours open on every visit, seen or not. tour.always(false) turns this off.",
    );
    expect(usePageTourStore.getState().always).toBe(true);
    expect(readTourAlways()).toBe(true);

    expect(window.tour!.always(false)).toBe("Tours open once per person again.");
    expect(usePageTourStore.getState().always).toBe(false);
    expect(readTourAlways()).toBe(false);
  });
});
