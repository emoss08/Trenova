import { apiService } from "@/services/api";
import type { ChecklistTemplate, ChecklistTemplates, TemplateItem } from "@/types/case-checklist";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CaseChecklistsSection } from "../case-checklists-section";

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
    id: "cct_org",
    kind: "ReadyToBill",
    customerId: "",
    customerName: "",
    items: ITEMS,
    version: 3,
    updatedAt: 0,
    ...overrides,
  };
}

function listing(): ChecklistTemplates {
  return {
    kind: "ReadyToBill",
    organization: template(),
    customers: [
      template({
        id: "cct_acme",
        customerId: "cus_acme",
        customerName: "Acme Foods",
        items: ITEMS.map((item) =>
          item.key === "accessorials" ? { ...item, mode: "Optional" } : item,
        ),
      }),
    ],
    locked: LOCKED,
  };
}

function renderSection({ readOnly = false } = {}) {
  vi.spyOn(apiService.caseChecklistService, "list").mockResolvedValue(listing());
  const save = vi
    .spyOn(apiService.caseChecklistService, "save")
    .mockImplementation(async (body) => ({
      ...template(),
      ...body,
      id: body.id ?? "cct_new",
      customerId: body.customerId ?? "",
      customerName: "",
      version: body.version + 1,
    }));
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <CaseChecklistsSection readOnly={readOnly} />
      </MemoryRouter>
    </QueryClientProvider>,
  );

  return { save };
}

function rowOf(name: string): HTMLElement {
  const row = screen.getByText(name).closest("li");
  if (!row) {
    throw new Error(`no row for ${name}`);
  }
  return row;
}

/**
 * Case checklists in the Desk's settings: the steps a person sets apart from
 * the ones that follow rules kept elsewhere, a Reorder mode for the case's
 * order, and a save that sends the steps in order at the version read.
 */
describe("CaseChecklistsSection", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("never offers a mode for a step that follows a rule kept elsewhere", async () => {
    renderSection();

    await screen.findByText("Proof of delivery received");
    const pod = rowOf("Proof of delivery received");
    expect(within(pod).queryByRole("radiogroup")).toBeNull();
    expect(within(pod).getByRole("link", { name: /Set in Billing profiles/u })).toBeVisible();
    expect(within(rowOf("Customer notified")).getByRole("radiogroup")).toBeVisible();
  });

  it("saves a step turned off, in order, at the version it was read at", async () => {
    const { save } = renderSection();
    await screen.findByText("Customer notified");

    fireEvent.click(within(rowOf("Customer notified")).getByRole("radio", { name: "Off" }));
    expect(screen.getByText("1 unsaved change")).toBeVisible();
    fireEvent.click(screen.getByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(save).toHaveBeenCalledTimes(1));
    const body = save.mock.calls[0][0];
    expect(body).toMatchObject({ id: "cct_org", version: 3, kind: "ReadyToBill" });
    expect(body.items.map((item) => item.key)).toEqual(ITEMS.map((item) => item.key));
    expect(body.items.find((item) => item.key === "customerNotified")?.mode).toBe("Off");
  });

  it("moves a step with the arrow keys on its handle and keeps focus on it", async () => {
    const { save } = renderSection();
    await screen.findByText("Customer notified");

    fireEvent.click(screen.getByRole("button", { name: "Reorder" }));
    const handle = screen.getByRole("button", { name: "Move Customer notified. Use the arrow keys." });
    act(() => handle.focus());
    fireEvent.keyDown(handle, { key: "ArrowUp" });

    await waitFor(() =>
      expect(
        screen.getByRole("button", { name: "Move Customer notified. Use the arrow keys." }),
      ).toHaveFocus(),
    );
    fireEvent.click(screen.getByRole("button", { name: "Save changes" }));
    await waitFor(() => expect(save).toHaveBeenCalledTimes(1));
    const keys = save.mock.calls[0][0].items.map((item) => item.key);
    expect(keys.indexOf("customerNotified")).toBe(keys.indexOf("accessorials") - 1);
  });

  it("refuses to switch to another checklist while this one has unsaved changes", async () => {
    renderSection();
    await screen.findByText("Customer notified");

    fireEvent.click(within(rowOf("Customer notified")).getByRole("radio", { name: "Off" }));
    fireEvent.click(screen.getByRole("button", { name: /Acme Foods/u }));

    expect(await screen.findByText("Save or discard before switching")).toBeVisible();
    expect(screen.getByRole("heading", { name: "Your organization's checklist" })).toBeVisible();

    fireEvent.click(screen.getByRole("button", { name: "Discard" }));
    fireEvent.click(screen.getByRole("button", { name: /Acme Foods/u }));
    expect(await screen.findByRole("heading", { name: "Acme Foods's checklist" })).toBeVisible();
  });

  it("refuses to save a document step without its document type, and opens it", async () => {
    const { save } = renderSection();
    await screen.findByText("Customer notified");

    fireEvent.click(screen.getByRole("button", { name: "Add a step" }));
    fireEvent.change(screen.getByLabelText(/^Name/u), { target: { value: "Lumper receipt" } });
    fireEvent.click(screen.getByRole("radio", { name: /A document on file/u }));
    fireEvent.click(screen.getByRole("button", { name: "Done" }));
    expect(screen.queryByLabelText(/^Name/u)).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Save changes" }));

    expect(await screen.findByText("Needs a fix")).toBeVisible();
    expect(screen.getByLabelText(/^Name/u)).toHaveValue("Lumper receipt");
    expect(save).not.toHaveBeenCalled();
  });

  it("sends a new step with a key the server accepts", async () => {
    const { save } = renderSection();
    await screen.findByText("Customer notified");

    fireEvent.click(screen.getByRole("button", { name: "Add a step" }));
    fireEvent.change(screen.getByLabelText(/^Name/u), { target: { value: "Shipper called" } });
    fireEvent.click(screen.getByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(save).toHaveBeenCalledTimes(1));
    const added = save.mock.calls[0][0].items.at(-1);
    expect(added?.key).toMatch(/^custom:[a-z0-9]{6,32}$/u);
    expect(added?.custom).toMatchObject({ label: "Shipper called", check: "Manual" });
  });

  it("shows a read-only checklist without a way to change it", async () => {
    renderSection({ readOnly: true });
    await screen.findByText("Customer notified");

    expect(screen.queryByRole("button", { name: "Reorder" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Add a step" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Add a customer" })).toBeNull();
    expect(within(rowOf("Customer notified")).getByRole("radio", { name: "Off" })).toBeDisabled();
  });
});
