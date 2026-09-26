import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { ProposalPreview, ProposalPreviewRequest } from "@/lib/graphql/agent-preview";
import type { ProposalField } from "@/types/assistant";
import { ProposalEditor, type ProposalEditorRequest } from "../proposal-editor";
import { field, preview, record } from "./preview-fixtures";

const fetchProposalPreview =
  vi.fn<(request: ProposalPreviewRequest, options?: unknown) => Promise<ProposalPreview>>();

vi.mock("@/lib/graphql/agent-preview", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-preview")>()),
  fetchProposalPreview: (request: ProposalPreviewRequest, options?: unknown) =>
    fetchProposalPreview(request, options),
}));

afterEach(() => {
  fetchProposalPreview.mockReset();
});

const FIELDS: ProposalField[] = [
  {
    name: "shipmentId",
    label: "Shipment",
    description: "",
    kind: "Text",
    required: true,
    options: [],
    minimum: null,
    maximum: null,
    maxLength: null,
    readOnly: true,
  },
  {
    name: "status",
    label: "Status",
    description: "",
    kind: "Text",
    required: true,
    options: [],
    minimum: null,
    maximum: null,
    maxLength: null,
  },
];

/** A preview whose digest and value follow the draft it was asked for. */
function previewFor({ modifications }: ProposalPreviewRequest): ProposalPreview {
  const status = typeof modifications?.status === "string" ? modifications.status : "Delayed";

  return preview({
    digest: `sha256:${status}`,
    changes: [record({ fields: [field({ before: "In transit", after: status })] })],
  });
}

function renderEditor(overrides: Partial<ProposalEditorRequest> = {}) {
  const onConfirm = vi.fn<ProposalEditorRequest["onConfirm"]>(async () => {});
  const onClose = vi.fn();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const request: ProposalEditorRequest = {
    summary: "Mark S-1001 delayed.",
    fields: FIELDS,
    arguments: { shipmentId: "shp_1", status: "Delayed" },
    preview: { scope: "approver", proposalId: "aprop_1" },
    onConfirm,
    ...overrides,
  };

  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ProposalEditor request={request} onClose={onClose} />
      </MemoryRouter>
    </QueryClientProvider>,
  );

  return { onConfirm, onClose };
}

const statusInput = () => screen.getByLabelText("Status");
const confirmButton = () => screen.getByRole("button", { name: "Approve with changes" });

function callsWithModifications() {
  return fetchProposalPreview.mock.calls.filter(([request]) => request.modifications !== null);
}

const SHIPMENT_FIELDS: ProposalField[] = [
  {
    name: "shipment",
    label: "Shipment",
    description: "",
    kind: "JSON",
    required: true,
    options: [],
    minimum: null,
    maximum: null,
    maxLength: null,
  },
];

describe("ProposalEditor focused on a value inside a field", () => {
  it("edits the one value and sends the field back whole with it changed", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockImplementation(async (request) => previewFor(request));
    const { onConfirm } = renderEditor({
      fields: SHIPMENT_FIELDS,
      arguments: { shipment: { customerId: "cus_1", bol: "SEED-BOL-009" } },
      focus: { param: "shipment.bol", label: "BOL" },
    });

    const bol = screen.getByLabelText("BOL");
    expect(bol).toHaveFocus();
    expect(bol).toHaveValue("SEED-BOL-009");
    await user.clear(bol);
    await user.type(bol, "BOL-2026-1");

    await waitFor(() => expect(confirmButton()).toBeEnabled());
    await user.click(confirmButton());

    await waitFor(() =>
      expect(onConfirm).toHaveBeenCalledWith(
        { shipment: { customerId: "cus_1", bol: "BOL-2026-1" } },
        "",
        expect.any(String),
      ),
    );
  });

  it("clears the value from the field when the input is emptied", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockImplementation(async (request) => previewFor(request));
    const { onConfirm } = renderEditor({
      fields: SHIPMENT_FIELDS,
      arguments: { shipment: { customerId: "cus_1", bol: "SEED-BOL-009" } },
      focus: { param: "shipment.bol", label: "BOL" },
    });

    await user.clear(screen.getByLabelText("BOL"));
    await waitFor(() => expect(confirmButton()).toBeEnabled());
    await user.click(confirmButton());

    await waitFor(() =>
      expect(onConfirm).toHaveBeenCalledWith(
        { shipment: { customerId: "cus_1" } },
        "",
        expect.any(String),
      ),
    );
  });

  it("focuses a top-level parameter's own control", () => {
    fetchProposalPreview.mockImplementation(async (request) => previewFor(request));
    renderEditor({ focus: { param: "status", label: "Status" } });

    expect(statusInput()).toHaveFocus();
  });
});

describe("ProposalEditor preview", () => {
  // The record the change is about stays as proposed: a change may alter
  // what is done to it, never which record it is.
  it("shows the target field without letting it be edited", () => {
    fetchProposalPreview.mockImplementation(async (request) => previewFor(request));
    renderEditor();

    expect(screen.getByLabelText("Shipment")).toHaveAttribute("readonly");
    expect(statusInput()).not.toHaveAttribute("readonly");
    expect(
      screen.getByText("Stays as proposed: a change can't point this at a different record."),
    ).toBeInTheDocument();
  });

  it("previews the proposal as proposed until a value changes", async () => {
    fetchProposalPreview.mockImplementation(async (request) => previewFor(request));
    renderEditor();

    await screen.findByText("What it would do as proposed");
    await waitFor(() =>
      expect(fetchProposalPreview).toHaveBeenCalledWith(
        { scope: "approver", id: "aprop_1", modifications: null },
        expect.anything(),
      ),
    );
  });

  it("previews the draft once typing settles, not on every keystroke", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockImplementation(async (request) => previewFor(request));
    renderEditor();

    await user.clear(statusInput());
    await user.type(statusInput(), "Late");

    expect(callsWithModifications()).toHaveLength(0);
    await waitFor(() => expect(callsWithModifications()).toHaveLength(1));
    expect(callsWithModifications()[0]?.[0].modifications).toEqual({ status: "Late" });
    expect(await screen.findByText("Late")).toBeInTheDocument();
  });

  it("approves only against the preview of the values in the form, and sends its digest", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockImplementation(async (request) => previewFor(request));
    const { onConfirm } = renderEditor();

    await user.clear(statusInput());
    await user.type(statusInput(), "Late");
    expect(confirmButton()).toBeDisabled();

    await waitFor(() => expect(confirmButton()).toBeEnabled());
    await user.click(confirmButton());

    await waitFor(() =>
      expect(onConfirm).toHaveBeenCalledWith({ status: "Late" }, "", "sha256:Late"),
    );
  });

  it("reads the preview again and says so when the approval's digest no longer matches", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockImplementation(async (request) => previewFor(request));
    const type = "https://trenova.app/problems/resource-conflict";
    const onConfirm = vi.fn<ProposalEditorRequest["onConfirm"]>(async () => {
      throw new GraphQLRequestError({
        kind: "graphql",
        message: "conflict",
        status: 200,
        graphQLErrors: [{ message: "conflict", extensions: { type }, type }],
      });
    });
    const { onClose } = renderEditor({ onConfirm });

    await user.clear(statusInput());
    await user.type(statusInput(), "Late");
    await waitFor(() => expect(confirmButton()).toBeEnabled());
    const before = callsWithModifications().length;
    await user.click(confirmButton());

    expect(
      await screen.findByText("This change looks different now — review it again."),
    ).toBeInTheDocument();
    await waitFor(() => expect(callsWithModifications().length).toBeGreaterThan(before));
    expect(onClose).not.toHaveBeenCalled();
  });

  it("says why values the tool refuses would not go through, and will not approve them", async () => {
    const user = userEvent.setup();
    const type = "https://trenova.app/problems/validation-error";
    fetchProposalPreview.mockImplementation(async (request) => {
      if (request.modifications !== null) {
        throw new GraphQLRequestError({
          kind: "graphql",
          message: "Status is not a shipment status",
          status: 200,
          graphQLErrors: [
            { message: "Status is not a shipment status", extensions: { type }, type },
          ],
        });
      }
      return previewFor(request);
    });
    renderEditor();

    await user.clear(statusInput());
    await user.type(statusInput(), "Nonsense");

    expect(
      await screen.findByText(
        /These values would not go through: Status is not a shipment status/u,
      ),
    ).toBeInTheDocument();
    expect(confirmButton()).toBeDisabled();
  });
});
