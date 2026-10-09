import { apiService } from "@/services/api";
import type { ChecklistTemplate, TemplateItem } from "@/types/case-checklist";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ChecklistTemplateEditor } from "../checklist-template-editor";

const LOCKED = ["delivered", "pod", "paperwork", "rateConfirmation", "billingHolds"];

const ITEMS: TemplateItem[] = [
  "delivered",
  "pod",
  "paperwork",
  "rateConfirmation",
  "carrierRateConfirmed",
  "accessorials",
  "customerNotified",
  "billingHolds",
].map((key) => ({ key, mode: "Required" }));

function template(overrides: Partial<ChecklistTemplate> = {}): ChecklistTemplate {
  return {
    id: "cct_1",
    kind: "ReadyToBill",
    customerId: "",
    customerName: "",
    items: ITEMS,
    version: 3,
    updatedAt: 0,
    ...overrides,
  };
}

function renderEditor(value = template()) {
  const onSaved = vi.fn();
  render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter>
        <ChecklistTemplateEditor
          kind="ReadyToBill"
          template={value}
          locked={LOCKED}
          onSaved={onSaved}
          onRemoved={vi.fn()}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );

  return { onSaved };
}

function rowOf(name: string): HTMLElement {
  const row = screen.getByText(name).closest("li");
  if (!row) {
    throw new Error(`no row for ${name}`);
  }
  return row;
}

/**
 * The editor lays a checklist out: each step required, optional or off
 * unless it follows a rule kept elsewhere, steps of the organization's own,
 * and a save that sends the steps in order at the version read.
 */
describe("ChecklistTemplateEditor", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("keeps a step that follows a rule kept elsewhere required", () => {
    renderEditor();

    expect(within(rowOf("Proof of delivery received")).getByText("Always required")).toBeVisible();
    expect(within(rowOf("Proof of delivery received")).queryByRole("radiogroup")).toBeNull();
    expect(within(rowOf("Customer notified")).getByRole("radiogroup")).toBeVisible();
  });

  it("saves a step turned off, in order, at the version it was read at", async () => {
    const save = vi
      .spyOn(apiService.caseChecklistService, "save")
      .mockImplementation(async (body) => ({ ...template(), ...body, id: "cct_1", version: 4 }));
    renderEditor();

    fireEvent.click(within(rowOf("Customer notified")).getByRole("radio", { name: "Off" }));
    fireEvent.click(screen.getByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(save).toHaveBeenCalledTimes(1));
    const body = save.mock.calls[0][0];
    expect(body).toMatchObject({ id: "cct_1", version: 3, kind: "ReadyToBill" });
    expect(body.items.map((item) => item.key)).toEqual(ITEMS.map((item) => item.key));
    expect(body.items.find((item) => item.key === "customerNotified")?.mode).toBe("Off");
  });

  it("refuses to save a step of its own without a name", async () => {
    const save = vi.spyOn(apiService.caseChecklistService, "save");
    renderEditor();

    fireEvent.click(screen.getByRole("button", { name: "Add a step" }));
    expect(screen.getByLabelText("Name")).toBeVisible();
    fireEvent.click(screen.getByRole("button", { name: "Save changes" }));

    expect(await screen.findByText("Name the step")).toBeVisible();
    expect(save).not.toHaveBeenCalled();
  });

  it("sends a new step with a key the server accepts", async () => {
    const save = vi
      .spyOn(apiService.caseChecklistService, "save")
      .mockImplementation(async (body) => ({ ...template(), ...body, id: "cct_1", version: 4 }));
    renderEditor();

    fireEvent.click(screen.getByRole("button", { name: "Add a step" }));
    fireEvent.change(screen.getByLabelText("Name"), { target: { value: "Shipper called" } });
    fireEvent.click(screen.getByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(save).toHaveBeenCalledTimes(1));
    const added = save.mock.calls[0][0].items.at(-1);
    expect(added?.key).toMatch(/^custom:[a-z0-9]{6,32}$/u);
    expect(added?.custom).toMatchObject({ label: "Shipper called", check: "Manual" });
  });
});
