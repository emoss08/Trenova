import { describe, expect, it } from "vitest";
import { packetAfterPlace, packetAfterToggle, packetDocumentIds } from "../billing-queue-packet";

const requirements = [
  { documentIds: ["doc_pod", "doc_pod_2"] },
  { documentIds: ["doc_bol", "doc_pod"] },
  { documentIds: [] },
];

describe("invoice packet", () => {
  it("sends each required document, requirement by requirement, while the biller has picked none", () => {
    expect(packetDocumentIds([], false, requirements)).toEqual(["doc_pod", "doc_pod_2", "doc_bol"]);
  });

  it("sends exactly the biller's pick, in their order, once there is one", () => {
    expect(packetDocumentIds(["doc_lumper", "doc_pod"], true, requirements)).toEqual([
      "doc_lumper",
      "doc_pod",
    ]);
  });

  it("sends nothing when nothing is picked and nothing is required", () => {
    expect(packetDocumentIds([], false, [])).toEqual([]);
  });

  it("sends nothing once the biller has taken every document out", () => {
    expect(packetDocumentIds([], true, requirements)).toEqual([]);
  });

  it("adds a document last, keeping the default the biller was looking at", () => {
    expect(packetAfterToggle("doc_lumper", true, ["doc_pod", "doc_bol"])).toEqual([
      "doc_pod",
      "doc_bol",
      "doc_lumper",
    ]);
  });

  it("takes out only the document left out", () => {
    expect(packetAfterToggle("doc_bol", false, ["doc_pod", "doc_bol"])).toEqual(["doc_pod"]);
  });

  it("never lists a document twice", () => {
    expect(packetAfterToggle("doc_pod", true, ["doc_pod"])).toEqual(["doc_pod"]);
  });

  it("moves a document already in the packet to the place it is dropped", () => {
    expect(packetAfterPlace("doc_lumper", 0, ["doc_pod", "doc_bol", "doc_lumper"])).toEqual([
      "doc_lumper",
      "doc_pod",
      "doc_bol",
    ]);
    expect(packetAfterPlace("doc_pod", 1, ["doc_pod", "doc_bol", "doc_lumper"])).toEqual([
      "doc_bol",
      "doc_pod",
      "doc_lumper",
    ]);
  });

  it("puts a document from outside the packet at the place it is dropped", () => {
    expect(packetAfterPlace("doc_lumper", 1, ["doc_pod", "doc_bol"])).toEqual([
      "doc_pod",
      "doc_lumper",
      "doc_bol",
    ]);
  });

  it("puts a document dropped past the end last", () => {
    expect(packetAfterPlace("doc_lumper", 9, ["doc_pod", "doc_bol"])).toEqual([
      "doc_pod",
      "doc_bol",
      "doc_lumper",
    ]);
  });
});
