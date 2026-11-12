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
  it("opens on what the page in front of the person is about", () => {
    const store = useDeskMarginStore.getState();
    store.setPageSubject(SHIPMENT);
    store.toggleMargin();

    expect(useDeskMarginStore.getState().margin).toEqual({ threadId: null, subject: SHIPMENT });
  });

  it("folds on the second toggle", () => {
    const store = useDeskMarginStore.getState();
    store.toggleMargin();
    store.toggleMargin();

    expect(useDeskMarginStore.getState().margin).toBeNull();
  });

  it("keeps the record a conversation is about when the conversation starts", () => {
    const store = useDeskMarginStore.getState();
    store.openMargin(SHIPMENT);
    store.showThread("ath_1");

    expect(useDeskMarginStore.getState().margin).toEqual({ threadId: "ath_1", subject: SHIPMENT });
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

    store.closeMargin();
    store.openMargin(null);

    expect(useDeskMarginStore.getState().margin).toEqual({ threadId: null, subject: SHIPMENT });
  });

  it("gives a margin opened on something else over to the case that opens", () => {
    const store = useDeskMarginStore.getState();
    store.showThread("ath_other", null);
    store.setCasePage(SHIPMENT);

    expect(useDeskMarginStore.getState().margin).toEqual({ threadId: null, subject: SHIPMENT });
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
    store.openMargin(SHIPMENT);
    store.clearSession();

    expect(useDeskMarginStore.getState()).toMatchObject({
      margin: null,
      pageSubject: null,
      openWork: [],
    });
  });
});
