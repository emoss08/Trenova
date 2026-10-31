import { describe, expect, it } from "vitest";
import { isInPacket, packetAfterToggle } from "../billing-queue-packet";

const pod = { id: "doc_pod", documentTypeId: "dt_pod" };
const bol = { id: "doc_bol", documentTypeId: "dt_bol" };
const lumper = { id: "doc_lumper", documentTypeId: "dt_receipt" };
const untyped = { id: "doc_photo", documentTypeId: null };
const documents = [pod, bol, lumper, untyped];
const required = new Set(["dt_pod", "dt_bol"]);

describe("invoice packet", () => {
  it("sends the customer's required documents while the biller has picked none", () => {
    expect(documents.filter((doc) => isInPacket(doc, [], required))).toEqual([pod, bol]);
  });

  it("sends exactly the biller's pick once there is one", () => {
    expect(documents.filter((doc) => isInPacket(doc, ["doc_lumper"], required))).toEqual([lumper]);
  });

  it("starts the first change from the default, so adding a receipt keeps the POD and BOL", () => {
    expect(packetAfterToggle(lumper, true, documents, [], required)).toEqual([
      "doc_pod",
      "doc_bol",
      "doc_lumper",
    ]);
  });

  it("drops only the unticked document from the default", () => {
    expect(packetAfterToggle(bol, false, documents, [], required)).toEqual(["doc_pod"]);
  });

  it("never lists a document twice", () => {
    expect(packetAfterToggle(pod, true, documents, ["doc_pod"], required)).toEqual(["doc_pod"]);
  });
});
