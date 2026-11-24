import {
  allowTool,
  areaLabel,
  denyTool,
  formScope,
  isSwitchedOff,
  scopeAllows,
  toAgentScope,
  toAgentScopeInput,
  toolArea,
  withAreas,
  type AgentScope,
} from "@/lib/agent-scope";
import { describe, expect, it } from "vitest";

function scope(over: Partial<AgentScope> = {}): AgentScope {
  return { areas: [], capabilities: [], denyAreas: [], denyCapabilities: [], ...over };
}

/**
 * The rules come from the server's `agentdefinition.Scope` (domain/agentdefinition/scope.go)
 * and `agentcapability.AreaOf`, so the builder shows what the agent will hold at run time.
 */
describe("toolArea", () => {
  it("reads the word before the first underscore", () => {
    expect(toolArea("dispatch_assign_move")).toBe("dispatch");
    expect(toolArea("invoice_list")).toBe("invoice");
  });

  it("puts the agent's own verbs under agent and the generic reads under records", () => {
    expect(toolArea("search_tools")).toBe("agent");
    expect(toolArea("delegate_task")).toBe("agent");
    expect(toolArea("records_query")).toBe("records");
    expect(toolArea("record_get")).toBe("records");
    expect(toolArea("describe_entity")).toBe("records");
    expect(toolArea("remember")).toBe("agent");
  });
});

describe("scopeAllows", () => {
  it("allows everything when no area or capability is named", () => {
    expect(scopeAllows({}, "shipment_cancel")).toBe(true);
    expect(scopeAllows(scope(), "invoice_send")).toBe(true);
  });

  it("allows an area's tools and named capabilities only, once something is named", () => {
    const held = scope({ areas: ["shipment"], capabilities: ["invoice_send"] });
    expect(scopeAllows(held, "shipment_cancel")).toBe(true);
    expect(scopeAllows(held, "invoice_send")).toBe(true);
    expect(scopeAllows(held, "invoice_void")).toBe(false);
  });

  it("lets a denial win over an area, a capability and everything", () => {
    expect(
      scopeAllows(
        scope({ areas: ["shipment"], denyCapabilities: ["shipment_cancel"] }),
        "shipment_cancel",
      ),
    ).toBe(false);
    expect(
      scopeAllows(
        scope({ capabilities: ["invoice_send"], denyAreas: ["invoice"] }),
        "invoice_send",
      ),
    ).toBe(false);
    expect(scopeAllows(scope({ denyAreas: ["*"] }), "shipment_get")).toBe(false);
    expect(scopeAllows(scope(), "")).toBe(false);
  });
});

describe("allowTool and denyTool", () => {
  it("names a tool only when nothing else allows it", () => {
    expect(allowTool(scope({ areas: ["shipment"] }), "shipment_get")).toEqual(
      scope({ areas: ["shipment"] }),
    );
    expect(allowTool(scope({ areas: ["shipment"] }), "invoice_send")).toEqual(
      scope({ areas: ["shipment"], capabilities: ["invoice_send"] }),
    );
  });

  it("lifts a switch-off instead of naming the tool", () => {
    expect(allowTool(scope({ denyCapabilities: ["shipment_cancel"] }), "shipment_cancel")).toEqual(
      scope(),
    );
  });

  it("drops a named tool, and switches off one an area or everything still allows", () => {
    expect(
      denyTool(scope({ capabilities: ["invoice_send", "shipment_get"] }), "invoice_send"),
    ).toEqual(scope({ capabilities: ["shipment_get"] }));
    expect(denyTool(scope({ areas: ["shipment"] }), "shipment_cancel")).toEqual(
      scope({ areas: ["shipment"], denyCapabilities: ["shipment_cancel"] }),
    );
    expect(denyTool(scope(), "shipment_cancel")).toEqual(
      scope({ denyCapabilities: ["shipment_cancel"] }),
    );
  });

  it("reads a scope being narrowed that names nothing as holding nothing", () => {
    expect(allowTool(scope(), "invoice_send", true)).toEqual(
      scope({ capabilities: ["invoice_send"] }),
    );
    expect(denyTool(scope({ capabilities: ["invoice_send"] }), "invoice_send", true)).toEqual(
      scope(),
    );
  });
});

describe("formScope", () => {
  it("holds nothing while narrowed with nothing named, and is the scope otherwise", () => {
    expect(scopeAllows(formScope(scope(), true), "shipment_get")).toBe(false);
    expect(formScope(scope(), false)).toEqual(scope());
    expect(formScope(scope({ areas: ["worker"] }), true)).toEqual(scope({ areas: ["worker"] }));
  });
});

describe("withAreas", () => {
  it("sets the areas, each once, and lifts a denial of any area now allowed", () => {
    expect(
      withAreas(scope({ denyAreas: ["shipment", "billing"] }), ["shipment", "worker", "shipment"]),
    ).toEqual(scope({ areas: ["shipment", "worker"], denyAreas: ["billing"] }));
  });
});

describe("toAgentScope and toAgentScopeInput", () => {
  it("reads null and absent lists as empty", () => {
    expect(toAgentScope(null)).toEqual(scope());
    expect(toAgentScope({ areas: ["shipment"], capabilities: null })).toEqual(
      scope({ areas: ["shipment"] }),
    );
  });

  it("sends every list, once each and sorted, so the same scope is the same request", () => {
    expect(
      toAgentScopeInput(
        scope({ areas: ["worker", "shipment", "worker"], denyCapabilities: ["b", "a"] }),
      ),
    ).toEqual({
      areas: ["shipment", "worker"],
      capabilities: [],
      denyAreas: [],
      denyCapabilities: ["a", "b"],
    });
  });
});

describe("isSwitchedOff", () => {
  it("is true only when every area is denied", () => {
    expect(isSwitchedOff({ denyAreas: ["*"] })).toBe(true);
    expect(isSwitchedOff({ denyAreas: ["shipment"] })).toBe(false);
    expect(isSwitchedOff({})).toBe(false);
  });
});

describe("areaLabel", () => {
  it("names an area in words, never by its code", () => {
    expect(areaLabel("masterdata")).toBe("Master data");
    expect(areaLabel("records")).toBe("Any record");
    expect(areaLabel("worker_pto")).toBe("Worker time off");
  });
});
