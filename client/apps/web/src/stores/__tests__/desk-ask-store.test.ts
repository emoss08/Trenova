import { afterEach, describe, expect, it } from "vitest";
import { useDeskAskStore } from "../desk-ask-store";

const SHIPMENT = { kind: "shipment", id: "shp_1", label: "S-240412" };

afterEach(() => useDeskAskStore.getState().reset());

describe("a new conversation asked for at the Desk", () => {
  it("carries the record the page was about and asks for the caret", () => {
    useDeskAskStore.getState().askNew(SHIPMENT);

    expect(useDeskAskStore.getState()).toMatchObject({ subject: SHIPMENT, focusPending: true });
  });

  it("hands the caret request over once", () => {
    const store = useDeskAskStore.getState();
    store.askNew(null);

    expect(store.takeFocus()).toBe(true);
    expect(useDeskAskStore.getState().takeFocus()).toBe(false);
  });

  it("keeps the record after the caret is taken", () => {
    const store = useDeskAskStore.getState();
    store.askNew(SHIPMENT);
    store.takeFocus();

    expect(useDeskAskStore.getState().subject).toEqual(SHIPMENT);
  });

  it("asking with no record drops the one carried before", () => {
    const store = useDeskAskStore.getState();
    store.askNew(SHIPMENT);
    store.askNew(null);

    expect(useDeskAskStore.getState().subject).toBeNull();
  });

  it("forgets the record and the caret request on reset", () => {
    const store = useDeskAskStore.getState();
    store.askNew(SHIPMENT);
    store.reset();

    expect(useDeskAskStore.getState()).toMatchObject({ subject: null, focusPending: false });
  });
});
