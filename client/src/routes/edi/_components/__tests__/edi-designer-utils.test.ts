import { describe, expect, it } from "vitest";
import {
  buildConditionString,
  getReadOnlyReason,
  getTransformOperationDefinition,
  insertPathReference,
  isTemplateVersionEditable,
  parseConditionString,
} from "../edi-designer-utils";

describe("isTemplateVersionEditable", () => {
  it("allows only draft versions", () => {
    expect(isTemplateVersionEditable({ status: "Draft" })).toBe(true);
    expect(isTemplateVersionEditable({ status: "Certified" })).toBe(false);
    expect(isTemplateVersionEditable({ status: "Active" })).toBe(false);
  });

  it("returns a lifecycle read-only reason", () => {
    expect(getReadOnlyReason({ status: "Certified", isActive: false })).toContain("Certified");
    expect(getReadOnlyReason({ status: "Active", isActive: true })).toContain("Active");
  });
});

describe("conditions", () => {
  it("builds declarative path conditions", () => {
    expect(buildConditionString({ mode: "truthy", path: "shipment.bol" })).toBe("shipment.bol");
    expect(buildConditionString({ mode: "falsey", path: "shipment.bol" })).toBe("!shipment.bol");
  });

  it("builds quoted comparisons", () => {
    expect(
      buildConditionString({
        mode: "comparison",
        path: "partner.mode",
        operator: "==",
        value: "test",
      }),
    ).toBe('partner.mode == "test"');
  });

  it("parses starlark function conditions", () => {
    expect(parseConditionString("starlark:include_stop()")).toEqual({
      mode: "starlarkFunction",
      functionName: "include_stop",
    });
  });
});

describe("transform operation metadata", () => {
  it("describes backend-supported transform arguments", () => {
    const definition = getTransformOperationDefinition("replace");
    expect(definition?.arguments.map((argument) => argument.key)).toEqual(["old", "new", "count"]);
  });
});

describe("insertPathReference", () => {
  it("inserts backend $path references into argument lists", () => {
    expect(insertPathReference("", "shipment.bol")).toBe("$shipment.bol");
    expect(insertPathReference("ABC", "partner.receiver")).toBe("ABC, $partner.receiver");
  });
});
