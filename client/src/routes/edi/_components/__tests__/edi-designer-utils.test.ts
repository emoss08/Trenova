import { describe, expect, it } from "vitest";
import {
  buildEDIDocumentResolutionRequest,
  buildEDIDocumentSourceRequest,
  getEDIDocumentSourceInputs,
  hasEDIDocumentSourceValue,
  parseEDIDocumentPayload,
  pruneEDIDocumentSourceValues,
  resolveEDIDocumentSourceContext,
} from "@/lib/edi/document-source";
import {
  partnerSettingFieldDisplayText,
  partnerSettingFieldSearchText,
  sourceContextFieldDisplayText,
  sourceContextFieldSearchText,
  toPartnerPathReference,
  toSourcePathReference,
} from "../designer/components/designer-fields";
import {
  buildConditionString,
  getReadOnlyReason,
  getTransformOperationDefinition,
  insertPathReference,
  isTemplateVersionEditable,
  parseConditionString,
} from "../designer/utils/edi-designer-utils";
import { ediScriptPresets, insertScriptPresetCode } from "../edi-script-presets";

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

  it("builds starlark function conditions", () => {
    expect(
      buildConditionString({
        mode: "starlarkFunction",
        functionName: "include_load_stop",
      }),
    ).toBe("starlark:include_load_stop");
  });

  it("parses starlark function conditions", () => {
    expect(parseConditionString("starlark:include_load_stop")).toEqual({
      mode: "starlarkFunction",
      functionName: "include_load_stop",
    });
  });

  it("parses legacy starlark function call conditions", () => {
    expect(parseConditionString("starlark:include_load_stop()")).toEqual({
      mode: "starlarkFunction",
      functionName: "include_load_stop",
    });
  });
});

describe("transform operation metadata", () => {
  it("describes backend-supported transform arguments", () => {
    const definition = getTransformOperationDefinition("replace");
    expect(definition?.arguments.map((argument) => argument.key)).toEqual(["old", "new", "count"]);
  });
});

describe("EDI document source inputs", () => {
  it("maps transaction sets to supported source fields", () => {
    expect(getEDIDocumentSourceInputs("204").map((input) => input.field)).toEqual([
      "shipmentId",
      "transferId",
      "payload",
    ]);
    expect(getEDIDocumentSourceInputs("210").map((input) => input.field)).toEqual([
      "invoiceId",
      "payload",
    ]);
    expect(getEDIDocumentSourceInputs("214").map((input) => input.field)).toEqual([
      "shipmentId",
      "shipmentEventId",
      "payload",
    ]);
    expect(getEDIDocumentSourceInputs("997").map((input) => input.field)).toEqual([
      "sourceMessageId",
      "payload",
    ]);
    expect(getEDIDocumentSourceInputs("999").map((input) => input.field)).toEqual([
      "sourceMessageId",
      "payload",
    ]);
    expect(getEDIDocumentSourceInputs(undefined).map((input) => input.field)).toEqual(["payload"]);
  });

  it("builds source requests with only fields valid for the transaction set", () => {
    expect(
      buildEDIDocumentSourceRequest(
        { shipmentId: " sp_123 ", invoiceId: " inv_123 " },
        "210",
        "Outbound",
      ),
    ).toMatchObject({
      invoiceId: "inv_123",
      shipmentId: undefined,
      transactionSet: "210",
      direction: "Outbound",
    });

    expect(
      buildEDIDocumentSourceRequest({ sourceMessageId: " edimsg_123 " }, "999", "Inbound"),
    ).toMatchObject({
      sourceMessageId: "edimsg_123",
      transactionSet: "999",
      direction: "Inbound",
    });
  });

  it("builds preview and generate resolution requests for source-backed transaction sets", () => {
    expect(
      buildEDIDocumentResolutionRequest({
        partnerDocumentProfileId: " profile_210 ",
        ediPartnerId: " partner_1 ",
        sourceValues: {
          shipmentId: "sp_ignored",
          invoiceId: " inv_123 ",
          payload: '{"ignored":true}',
        },
        transactionSet: "210",
        direction: "Outbound",
        payload: { invoice: { id: "inline" } } as never,
      }),
    ).toMatchObject({
      partnerDocumentProfileId: "profile_210",
      ediPartnerId: "partner_1",
      invoiceId: "inv_123",
      shipmentId: undefined,
      transactionSet: "210",
      direction: "Outbound",
      payload: { invoice: { id: "inline" } },
    });

    expect(
      buildEDIDocumentResolutionRequest({
        partnerDocumentProfileId: "profile_214",
        sourceValues: {
          shipmentId: " sp_123 ",
          shipmentEventId: " spe_123 ",
          invoiceId: "inv_ignored",
        },
        transactionSet: "214",
        direction: "Outbound",
      }),
    ).toMatchObject({
      partnerDocumentProfileId: "profile_214",
      shipmentId: "sp_123",
      shipmentEventId: "spe_123",
      invoiceId: undefined,
      transactionSet: "214",
      direction: "Outbound",
    });

    expect(
      buildEDIDocumentResolutionRequest({
        ediPartnerId: " partner_997 ",
        sourceValues: { sourceMessageId: " msg_997 ", transferId: "editr_ignored" },
        transactionSet: "997",
        direction: "Inbound",
      }),
    ).toMatchObject({
      ediPartnerId: "partner_997",
      sourceMessageId: "msg_997",
      transferId: undefined,
      transactionSet: "997",
      direction: "Inbound",
    });

    expect(
      buildEDIDocumentResolutionRequest({
        ediPartnerId: "partner_999",
        sourceValues: { sourceMessageId: " msg_999 " },
        transactionSet: "999",
        direction: "Inbound",
      }),
    ).toMatchObject({
      ediPartnerId: "partner_999",
      sourceMessageId: "msg_999",
      transactionSet: "999",
      direction: "Inbound",
    });
  });

  it("detects source values from the active transaction source set", () => {
    expect(hasEDIDocumentSourceValue({ shipmentId: "sp_123" }, "204")).toBe(true);
    expect(hasEDIDocumentSourceValue({ shipmentId: "sp_123" }, "210")).toBe(false);
    expect(hasEDIDocumentSourceValue({ invoiceId: "inv_123" }, "210")).toBe(true);
  });

  it("prunes source values that are inactive for the selected transaction set", () => {
    expect(
      pruneEDIDocumentSourceValues(
        {
          shipmentId: "sp_ignored",
          invoiceId: "inv_123",
          shipmentEventId: "spe_ignored",
          payload: "{}",
        },
        "210",
      ),
    ).toEqual({ invoiceId: "inv_123", payload: "{}" });
  });

  it("uses unsaved template metadata before selected profile metadata for source context", () => {
    expect(
      resolveEDIDocumentSourceContext({
        profile: { transactionSet: "204", direction: "Outbound" } as never,
        template: { transactionSet: "210", direction: "Inbound" } as never,
        fallbackTransactionSet: "214",
        fallbackDirection: "Outbound",
      }),
    ).toEqual({ transactionSet: "210", direction: "Inbound" });
  });

  it("reports invalid payload JSON without producing a payload", () => {
    expect(parseEDIDocumentPayload("{")).toEqual({ ok: false });
    expect(parseEDIDocumentPayload("")).toEqual({ ok: true });
    expect(parseEDIDocumentPayload('{"invoice":{"invoiceNumber":"INV-1"}}')).toEqual({
      ok: true,
      payload: { invoice: { invoiceNumber: "INV-1" } },
    });
  });
});

describe("insertPathReference", () => {
  it("inserts backend $path references into argument lists", () => {
    expect(insertPathReference("", "shipment.bol")).toBe("$shipment.bol");
    expect(insertPathReference("ABC", "partner.receiver")).toBe("ABC, $partner.receiver");
  });

  it("builds source and partner field insert paths", () => {
    expect(toSourcePathReference({ path: "shipment.bol" } as never)).toBe("shipment.bol");
    expect(toPartnerPathReference({ path: "envelope.receiverId" } as never)).toBe(
      "partner.envelope.receiverId",
    );
    expect(insertPathReference("", toSourcePathReference({ path: "shipment.bol" } as never))).toBe(
      "$shipment.bol",
    );
    expect(
      insertPathReference("ABC", toPartnerPathReference({ path: "envelope.receiverId" } as never)),
    ).toBe("ABC, $partner.envelope.receiverId");
  });
});

describe("field picker display text", () => {
  it("formats searchable source context field text", () => {
    const field = {
      path: "shipment.stops[0].city",
      displayName: "Stop city",
      description: "Pickup or delivery city",
      dataType: "string",
    } as never;

    expect(sourceContextFieldDisplayText(field)).toBe("shipment.stops[0].city (string)");
    expect(sourceContextFieldSearchText(field)).toContain("Stop city");
    expect(sourceContextFieldSearchText(field)).toContain("Pickup or delivery city");
  });

  it("formats searchable partner setting field text", () => {
    const field = {
      path: "envelope.receiverId",
      label: "Receiver ID",
      description: "ISA receiver identifier",
      dataType: "string",
      groupKey: "envelope",
    } as never;

    expect(partnerSettingFieldDisplayText(field)).toBe("envelope.receiverId (string)");
    expect(partnerSettingFieldSearchText(field)).toContain("Receiver ID");
    expect(partnerSettingFieldSearchText(field)).toContain("envelope");
  });
});

describe("ediScriptPresets", () => {
  it("has unique registry IDs", () => {
    const ids = ediScriptPresets.map((preset) => preset.id);
    expect(new Set(ids).size).toBe(ids.length);
  });

  it("has non-empty code for every preset", () => {
    expect(ediScriptPresets.every((preset) => preset.code.trim().length > 0)).toBe(true);
  });

  it("inserts preset code into an empty editor", () => {
    expect(insertScriptPresetCode("", { code: "\nvalue\n" })).toBe("value");
  });

  it("appends preset code to a non-empty editor with one blank line", () => {
    expect(insertScriptPresetCode("existing\n", { code: "\nvalue\n" })).toBe("existing\n\nvalue");
  });

  it("parses condition reference presets as Starlark function references", () => {
    const referencePresets = ediScriptPresets.filter(
      (preset) =>
        preset.category === "condition" &&
        preset.id.endsWith(".reference") &&
        preset.code.startsWith("starlark:"),
    );

    expect(referencePresets.length).toBeGreaterThan(0);
    for (const preset of referencePresets) {
      expect(preset.code).not.toContain("()");
      expect(parseConditionString(preset.code)).toEqual({
        mode: "starlarkFunction",
        functionName: preset.code.replace("starlark:", ""),
      });
    }
  });

  it("keeps inline condition presets as inline Starlark", () => {
    const inlinePresets = ediScriptPresets.filter(
      (preset) => preset.category === "condition" && preset.id.endsWith(".inline"),
    );

    expect(inlinePresets.length).toBeGreaterThan(0);
    for (const preset of inlinePresets) {
      const parsed = parseConditionString(preset.code);
      expect(parsed.mode).toBe("inlineStarlark");
      expect(buildConditionString(parsed)).toBe(preset.code.trim());
    }
  });
});
