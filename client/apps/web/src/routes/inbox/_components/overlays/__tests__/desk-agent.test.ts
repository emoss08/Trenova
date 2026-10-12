import { describe, expect, it } from "vitest";
import { agentRoleFor, pickAgentFor, type AgentLike } from "../desk/desk-agent";

const DISPATCH: AgentLike = {
  id: "a_dispatch",
  name: "Dispatch",
  description: "Loads, tenders, carriers",
  template: "DispatchAssistant",
  toolNames: ["create_shipment", "tender_shipment"],
};
const BILLING: AgentLike = {
  id: "a_billing",
  name: "Billing",
  description: "Invoices, detention, pay",
  template: "BillingAssistant",
  toolNames: ["file_document", "waive_detention"],
};
const SERVICE: AgentLike = {
  id: "a_cs",
  name: "Customer service",
  description: "Status, ETAs, replies",
  template: "CustomerAssistant",
  toolNames: ["shipment_eta", "draft_reply"],
};
const AGENTS = [SERVICE, BILLING, DISPATCH];

describe("which agent a message is handed to", () => {
  it("sends tenders and rate cons to dispatch, money and proof to billing, the rest to service", () => {
    expect(agentRoleFor("Tender")).toBe("dispatch");
    expect(agentRoleFor("RateConfirmation")).toBe("dispatch");
    expect(agentRoleFor("Invoice")).toBe("billing");
    expect(agentRoleFor("DetentionDispute")).toBe("billing");
    expect(agentRoleFor("ProofOfDelivery")).toBe("billing");
    expect(agentRoleFor("StatusRequest")).toBe("service");
    expect(agentRoleFor("Other")).toBe("service");
    expect(agentRoleFor(null)).toBe("service");
  });

  it("finds the agent by what it is called and does, whatever order they are listed in", () => {
    expect(pickAgentFor(AGENTS, "Tender")?.id).toBe("a_dispatch");
    expect(pickAgentFor(AGENTS, "Invoice")?.id).toBe("a_billing");
    expect(pickAgentFor(AGENTS, "StatusRequest")?.id).toBe("a_cs");
  });

  it("reads an agent's name ahead of the tools it happens to hold", () => {
    const carrierDesk: AgentLike = {
      id: "a_carrier_desk",
      name: "Carrier desk",
      description: "Books carriers on open loads",
      template: "",
      toolNames: ["send_invoice"],
    };
    expect(pickAgentFor([BILLING, carrierDesk], "Tender")?.id).toBe("a_carrier_desk");
  });

  it("falls back to the agent last asked when none fits, then to the first", () => {
    const general: AgentLike = { id: "a_g", name: "Helper", description: "", template: "", toolNames: [] };
    const other: AgentLike = { id: "a_o", name: "Ops", description: "", template: "", toolNames: [] };
    expect(pickAgentFor([general, other], "Invoice", "a_o")?.id).toBe("a_o");
    expect(pickAgentFor([general, other], "Invoice", "a_missing")?.id).toBe("a_g");
  });

  it("prefers the agent last asked between two that fit as well", () => {
    const billingTwo: AgentLike = { ...BILLING, id: "a_billing_2" };
    expect(pickAgentFor([BILLING, billingTwo], "Invoice", "a_billing_2")?.id).toBe("a_billing_2");
  });

  it("has nobody to pick from an empty list", () => {
    expect(pickAgentFor([], "Tender")).toBeNull();
  });
});
