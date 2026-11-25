import { afterEach, describe, expect, it } from "vitest";
import { useDeskMarginStore, withOpenWork } from "../desk-margin-store";

const SHIPMENT = { kind: "shipment", id: "shp_1", label: "S-240412" };
const INVOICE = { kind: "invoice", id: "inv_1", label: "INV-10482" };

afterEach(() => useDeskMarginStore.getState().clearSession());

describe("withOpenWork", () => {
  it("puts the case just opened first and keeps each case once", () => {
    const work = withOpenWork([INVOICE, SHIPMENT], SHIPMENT);

    expect(work).toEqual([SHIPMENT, INVOICE]);
  });

  it("keeps the four newest cases", () => {
    const cases = Array.from({ length: 5 }, (_, index) => ({
      kind: "shipment",
      id: `shp_${index}`,
      label: `S-${index}`,
    }));
    const work = cases.reduce(withOpenWork, [] as typeof cases);

    expect(work.map((entry) => entry.id)).toEqual(["shp_4", "shp_3", "shp_2", "shp_1"]);
  });
});

describe("Nova's margin", () => {
  it("holds a conversation that already exists with the record it is about", () => {
    const store = useDeskMarginStore.getState();
    store.showThread("ath_1", SHIPMENT);

    expect(useDeskMarginStore.getState().margin).toEqual({ threadId: "ath_1", subject: SHIPMENT });
  });

  it("keeps the record it was about when another conversation on it is shown", () => {
    const store = useDeskMarginStore.getState();
    store.showThread("ath_1", SHIPMENT);
    store.showThread("ath_2");

    expect(useDeskMarginStore.getState().margin).toEqual({ threadId: "ath_2", subject: SHIPMENT });
  });

  it("holds only the case's conversation while a case is in front", () => {
    const store = useDeskMarginStore.getState();
    store.setCasePage(SHIPMENT);
    store.showThread("ath_case", SHIPMENT);
    store.showThread("ath_other", null);
    store.showThread("ath_invoice", INVOICE);

    expect(useDeskMarginStore.getState().margin).toEqual({
      threadId: "ath_case",
      subject: SHIPMENT,
    });
  });

  it("folds a conversation about something else when a case opens", () => {
    const store = useDeskMarginStore.getState();
    store.showThread("ath_other", null);
    store.setCasePage(SHIPMENT);

    expect(useDeskMarginStore.getState().margin).toBeNull();
  });

  it("keeps the case's own conversation when its case opens", () => {
    const store = useDeskMarginStore.getState();
    store.showThread("ath_case", SHIPMENT);
    store.setCasePage(SHIPMENT);

    expect(useDeskMarginStore.getState().margin).toEqual({
      threadId: "ath_case",
      subject: SHIPMENT,
    });
  });

  it("takes any conversation again once the case is left", () => {
    const store = useDeskMarginStore.getState();
    store.setCasePage(SHIPMENT);
    store.setCasePage(null);
    store.showThread("ath_other", null);

    expect(useDeskMarginStore.getState().margin).toEqual({ threadId: "ath_other", subject: null });
  });

  it("forgets the open cases and the margin when the session ends", () => {
    const store = useDeskMarginStore.getState();
    store.noteOpenWork(SHIPMENT);
    store.showThread("ath_1", SHIPMENT);
    store.clearSession();

    expect(useDeskMarginStore.getState()).toMatchObject({
      margin: null,
      pageSubject: null,
      openWork: [],
    });
  });
});
