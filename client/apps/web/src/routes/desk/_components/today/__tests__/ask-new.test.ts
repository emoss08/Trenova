import { describe, expect, it } from "vitest";
import { startAbout } from "../ask-new";

const SHIPMENT = { kind: "shipment", id: "shp_1", label: "S-240412" };
const FILE = new File(["x"], "pod.pdf", { type: "application/pdf" });

describe("startAbout", () => {
  it("starts a question about nothing with what was asked and no binding", () => {
    expect(
      startAbout(null, {
        files: [FILE],
        providerId: "aip_1",
        mentions: [{ type: "invoice", id: "inv_1", label: "INV-10482" }],
      }),
    ).toEqual({
      extras: {
        files: [FILE],
        providerId: "aip_1",
        mentions: [{ type: "invoice", id: "inv_1", label: "INV-10482" }],
      },
    });
  });

  it("names the record first and binds a record with a case to it", () => {
    expect(
      startAbout(SHIPMENT, {
        files: [],
        providerId: "",
        mentions: [{ type: "invoice", id: "inv_1", label: "INV-10482" }],
      }),
    ).toEqual({
      extras: {
        files: [],
        providerId: "",
        mentions: [
          { type: "shipment", id: "shp_1", label: "S-240412" },
          { type: "invoice", id: "inv_1", label: "INV-10482" },
        ],
      },
      binding: {
        subjectType: "Shipment",
        subjectId: "shp_1",
        page: {
          path: "/desk/r/shipment/shp_1",
          entityType: "shipment",
          entityId: "shp_1",
          title: "S-240412",
          view: null,
          draft: null,
        },
      },
    });
  });

  it("names the record once when the question already mentions it", () => {
    const { extras } = startAbout(SHIPMENT, {
      files: [],
      providerId: "",
      mentions: [{ type: "shipment", id: "shp_1", label: "S-240412" }],
    });

    expect(extras.mentions).toEqual([{ type: "shipment", id: "shp_1", label: "S-240412" }]);
  });

  it("binds a record with no case to its page only", () => {
    expect(startAbout({ kind: "customer", id: "cus_1", label: "Acme" }).binding).toEqual({
      subjectType: undefined,
      subjectId: undefined,
      page: {
        path: "/desk/r/customer/cus_1",
        entityType: "customer",
        entityId: "cus_1",
        title: "Acme",
        view: null,
        draft: null,
      },
    });
  });

  it("starts with no files, no provider and no mentions when nothing was attached", () => {
    expect(startAbout(null)).toEqual({ extras: { files: [], providerId: "", mentions: [] } });
  });
});
