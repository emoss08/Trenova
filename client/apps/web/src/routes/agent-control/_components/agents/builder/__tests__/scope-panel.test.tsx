import type { ToolCatalogEntry } from "@/types/assistant";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useEffect } from "react";
import { FormProvider, useForm, type UseFormReturn } from "react-hook-form";
import { describe, expect, it, vi } from "vitest";
import { agentFormDefaults, type AgentFormValues } from "../../agent-form-schema";
import { ScopePanel } from "../scope-panel";

function tool(name: string): ToolCatalogEntry {
  return {
    name,
    description: "",
    parameters: null,
    kind: "action",
    resource: name.split("_")[0]!,
    operation: "update",
    defaultAutonomyTier: "",
    reversible: false,
    core: false,
    prerequisites: [],
    extension: "",
    grantedToEveryAgent: false,
  } as ToolCatalogEntry;
}

const catalog = [tool("shipment_cancel"), tool("worker_update"), tool("invoice_send")];

function renderPanel(values: Partial<AgentFormValues>, onChoose = vi.fn()) {
  let form!: UseFormReturn<AgentFormValues>;
  function Host() {
    const hostForm = useForm<AgentFormValues>({
      defaultValues: { ...agentFormDefaults, ...values },
    });
    useEffect(() => {
      form = hostForm;
    });
    return (
      <FormProvider {...hostForm}>
        <ScopePanel selectable={catalog} catalog={catalog} onChoose={onChoose} />
      </FormProvider>
    );
  }
  render(<Host />);
  return () => form;
}

const everything = { areas: [], capabilities: [], denyAreas: [], denyCapabilities: [] };

describe("ScopePanel", () => {
  it("offers areas only once the agent is held to less than everything", async () => {
    const form = renderPanel({ scope: everything, limited: false });
    expect(screen.queryByRole("group", { name: "Areas" })).toBeNull();

    await userEvent.click(
      screen.getByRole("switch", { name: "Everything the person it works for may do" }),
    );

    expect(form().getValues("limited")).toBe(true);
    expect(screen.getByRole("group", { name: "Areas" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Shipments" }).getAttribute("aria-pressed")).toBe(
      "false",
    );
  });

  it("allows an area by its name in words and lifts a denial of it", async () => {
    const form = renderPanel({
      scope: { ...everything, capabilities: ["invoice_send"], denyAreas: ["shipment"] },
      limited: true,
    });

    await userEvent.click(screen.getByRole("button", { name: "Shipments" }));

    expect(form().getValues("scope")).toEqual({
      areas: ["shipment"],
      capabilities: ["invoice_send"],
      denyAreas: [],
      denyCapabilities: [],
    });
  });

  it("clears the areas and capabilities when let do everything again, keeping denials", async () => {
    const form = renderPanel({
      scope: { ...everything, areas: ["worker"], denyCapabilities: ["shipment_cancel"] },
      limited: true,
    });

    await userEvent.click(
      screen.getByRole("switch", { name: "Everything the person it works for may do" }),
    );

    expect(form().getValues("limited")).toBe(false);
    expect(form().getValues("scope")).toEqual({
      ...everything,
      denyCapabilities: ["shipment_cancel"],
    });
  });

  it("turns a switched-off tool back on and unpins a capability", async () => {
    const form = renderPanel({
      scope: { ...everything, denyCapabilities: ["shipment_cancel"] },
      pinnedTools: ["invoice_send"],
    });

    await userEvent.click(screen.getByRole("button", { name: /^Turn .+ back on$/ }));
    await userEvent.click(screen.getByRole("button", { name: /^Unpin / }));

    expect(form().getValues("scope")).toEqual(everything);
    expect(form().getValues("pinnedTools")).toEqual([]);
  });

  it("says when every area is switched off and switches the agent back on", async () => {
    const form = renderPanel({ scope: { ...everything, denyAreas: ["*"] } });

    expect(
      screen.getByText("Every area is switched off, so this agent can do nothing."),
    ).toBeTruthy();
    await userEvent.click(screen.getByRole("button", { name: "Switch it back on" }));

    expect(form().getValues("scope")).toEqual(everything);
  });
});
