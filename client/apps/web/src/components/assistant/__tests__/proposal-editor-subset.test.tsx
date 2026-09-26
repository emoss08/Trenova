import type { PendingProposalFieldsFragment } from "@trenova/graphql/generated/graphql";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ProposalPreview, ProposalPreviewRequest } from "@/lib/graphql/agent-preview";
import { asAssistantProposal } from "@/routes/desk/_components/decisions/decision-presenters";
import { stubLayout } from "@/test/layout";
import { assistantProposalSchema } from "@/types/assistant";
import { ProposalEditor, type ProposalEditorRequest } from "../proposal-editor";
import { field, preview, record } from "./preview-fixtures";

const fetchProposalPreview =
  vi.fn<(request: ProposalPreviewRequest, options?: unknown) => Promise<ProposalPreview>>();

vi.mock("@/lib/graphql/agent-preview", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-preview")>()),
  fetchProposalPreview: (request: ProposalPreviewRequest, options?: unknown) =>
    fetchProposalPreview(request, options),
}));

// The list is windowed, and a windowed list lays out nothing without a size.
let restoreLayout = () => {};

beforeEach(() => {
  restoreLayout = stubLayout(320);
});

afterEach(() => {
  restoreLayout();
  fetchProposalPreview.mockReset();
});

const SHP_1 = "shp_01K0000000000000000000001";
const SHP_2 = "shp_01K0000000000000000000002";
const SHP_3 = "shp_01K0000000000000000000003";

type ParameterField = PendingProposalFieldsFragment["parameterFields"][number];

/*
Written from the contract: AgentProposal.parameterFields as the
PendingProposalFields fragment selects it (services/tms/internal/api/graphql/
schema/agent.graphqls). A RecordSubset field names its resource and lists
every proposed record as { id, label }; the label is the id when the record
is gone (SHP_3 here). Every other kind has null for both.
*/
function subsetField(overrides: Partial<ParameterField> = {}): ParameterField {
  return {
    name: "shipmentIds",
    label: "Shipment ids",
    description: "The shipments to transfer.",
    kind: "RecordSubset",
    required: true,
    options: [],
    minimum: null,
    maximum: null,
    maxLength: null,
    readOnly: false,
    resource: "shipment",
    choices: [
      { id: SHP_1, label: "PRO-1001" },
      { id: SHP_2, label: "PRO-1002" },
      { id: SHP_3, label: SHP_3 },
    ],
    ...overrides,
  };
}

function billTypeField(): ParameterField {
  return {
    name: "billType",
    label: "Bill type",
    description: "",
    kind: "Choice",
    required: false,
    options: ["Invoice", "CreditMemo", "DebitMemo"],
    minimum: null,
    maximum: null,
    maxLength: null,
    readOnly: false,
    resource: null,
    choices: null,
  };
}

function pendingNode(
  parameterFields: ParameterField[],
  toolParams: Record<string, unknown>,
): PendingProposalFieldsFragment {
  return {
    id: "aprop_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    runId: "arun_1",
    toolName: "transfer_to_billing",
    toolParams,
    confidence: 0.9,
    rationale: "Delivered and paperwork in.",
    autonomyTier: "Propose",
    status: "Pending",
    planId: null,
    planStep: 0,
    simulatedAt: null,
    simulation: null,
    modifications: null,
    version: 1,
    createdAt: 1_790_000_000,
    updatedAt: 1_790_000_000,
    parameterFields,
    evidence: [],
    run: null,
  };
}

/** The preview names SHP_2's refusal; SHP_1 and SHP_3 are past what it shows. */
function transferPreview({ modifications }: ProposalPreviewRequest): ProposalPreview {
  const kept = (modifications?.shipmentIds as string[] | undefined) ?? [SHP_1, SHP_2, SHP_3];

  return preview({
    tool: "transfer_to_billing",
    digest: `sha256:${kept.join("+")}`,
    changes: kept.includes(SHP_2)
      ? [
          record({
            entityId: SHP_2,
            record: { entityType: "shipment", id: SHP_2 },
            label: "PRO-1002",
            fields: [
              field({
                path: "outcome",
                label: "What the transfer does",
                before: null,
                after: "Refused (RequirementsUnmet): Proof of delivery is missing",
              }),
            ],
          }),
        ]
      : [],
    omittedRecords: kept.length - (kept.includes(SHP_2) ? 1 : 0),
  });
}

function renderEditor(
  parameterFields: ParameterField[] = [subsetField(), billTypeField()],
  toolParams: Record<string, unknown> = { shipmentIds: [SHP_1, SHP_2, SHP_3] },
) {
  const proposal = asAssistantProposal(pendingNode(parameterFields, toolParams));
  const onConfirm = vi.fn<ProposalEditorRequest["onConfirm"]>(async () => {});
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ProposalEditor
          request={{
            summary: "Transfer 3 shipments to billing.",
            fields: proposal.fields,
            arguments: proposal.arguments,
            preview: { scope: "approver", proposalId: "aprop_1" },
            onConfirm,
          }}
          onClose={vi.fn()}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );

  return { onConfirm };
}

const confirmButton = () => screen.getByRole("button", { name: "Approve with changes" });
const shipmentBox = (label: string) => screen.getByRole("checkbox", { name: label });

describe("ProposalEditor record subset", () => {
  it("lists every proposed record by its label, all ticked", () => {
    fetchProposalPreview.mockImplementation(async (request) => transferPreview(request));
    renderEditor();

    for (const label of ["PRO-1001", "PRO-1002", SHP_3]) {
      expect(shipmentBox(label)).toBeChecked();
    }
    expect(screen.getByText("3 of 3 shipments")).toBeInTheDocument();
    expect(screen.queryByPlaceholderText("Comma-separated values")).not.toBeInTheDocument();
  });

  it("sends the ids it kept, in the order proposed, as the parameter's value", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockImplementation(async (request) => transferPreview(request));
    const { onConfirm } = renderEditor();

    await user.click(shipmentBox("PRO-1002"));

    expect(shipmentBox("PRO-1002")).not.toBeChecked();
    expect(screen.getByText("2 of 3 shipments")).toBeInTheDocument();
    await waitFor(() =>
      expect(fetchProposalPreview).toHaveBeenCalledWith(
        { scope: "approver", id: "aprop_1", modifications: { shipmentIds: [SHP_1, SHP_3] } },
        expect.anything(),
      ),
    );
    await waitFor(() => expect(confirmButton()).toBeEnabled());
    await user.click(confirmButton());

    await waitFor(() =>
      expect(onConfirm).toHaveBeenCalledWith(
        { shipmentIds: [SHP_1, SHP_3] },
        "",
        `sha256:${SHP_1}+${SHP_3}`,
      ),
    );
  });

  it("sends nothing for the subset once every record is ticked again", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockImplementation(async (request) => transferPreview(request));
    renderEditor();

    await user.click(shipmentBox("PRO-1001"));
    await user.click(screen.getByRole("button", { name: "Select all" }));

    expect(shipmentBox("PRO-1001")).toBeChecked();
    expect(screen.getByText("Nothing changed yet")).toBeInTheDocument();
    expect(confirmButton()).toBeDisabled();
  });

  it("will not approve with every record unticked, and says to keep one or reject", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockImplementation(async (request) => transferPreview(request));
    renderEditor();

    await user.click(screen.getByRole("button", { name: "Clear" }));

    for (const label of ["PRO-1001", "PRO-1002", SHP_3]) {
      expect(shipmentBox(label)).not.toBeChecked();
    }
    expect(screen.getByText("0 of 3 shipments")).toBeInTheDocument();
    expect(
      screen.getByText("Keep at least one, or reject the proposal instead."),
    ).toBeInTheDocument();
    expect(confirmButton()).toBeDisabled();
    await new Promise((resolve) => setTimeout(resolve, 500));
    expect(
      fetchProposalPreview.mock.calls.some(
        ([request]) => (request.modifications?.shipmentIds as unknown[] | undefined)?.length === 0,
      ),
    ).toBe(false);
  });

  it("shows a record's outcome from the preview on its row", async () => {
    fetchProposalPreview.mockImplementation(async (request) => transferPreview(request));
    renderEditor();

    const row = shipmentBox("PRO-1002").closest("[data-slot='subset-row']");
    expect(row).not.toBeNull();
    expect(
      await within(row as HTMLElement).findByText(
        "Refused (RequirementsUnmet): Proof of delivery is missing",
      ),
    ).toBeInTheDocument();
  });

  it("keeps the preview's outcome on a row after it is unticked", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockImplementation(async (request) => transferPreview(request));
    renderEditor();

    const row = shipmentBox("PRO-1002").closest("[data-slot='subset-row']") as HTMLElement;
    await within(row).findByText("Refused (RequirementsUnmet): Proof of delivery is missing");
    await user.click(shipmentBox("PRO-1002"));

    await waitFor(() => expect(confirmButton()).toBeEnabled());
    expect(
      within(row).getByText("Refused (RequirementsUnmet): Proof of delivery is missing"),
    ).toBeInTheDocument();
  });

  it("lists a record the preview does not reach by its label alone", async () => {
    fetchProposalPreview.mockImplementation(async (request) => transferPreview(request));
    renderEditor();

    const refused = shipmentBox("PRO-1002").closest("[data-slot='subset-row']") as HTMLElement;
    await within(refused).findByText("Refused (RequirementsUnmet): Proof of delivery is missing");
    for (const label of ["PRO-1001", SHP_3]) {
      const row = shipmentBox(label).closest("[data-slot='subset-row']") as HTMLElement;
      expect(row.textContent).toBe(label);
    }
  });

  it("filters a long list by label without changing what is kept", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockImplementation(async (request) => transferPreview(request));
    const ids = Array.from(
      { length: 12 },
      (_, index) => `shp_01K00000000000000000000${String(index + 10)}`,
    );
    const { onConfirm } = renderEditor(
      [
        subsetField({
          choices: ids.map((id, index) => ({ id, label: `PRO-${2000 + index}` })),
        }),
      ],
      { shipmentIds: ids },
    );

    await user.type(screen.getByRole("searchbox", { name: "Filter shipment ids" }), "PRO-2011");

    expect(screen.getAllByRole("checkbox")).toHaveLength(1);
    await user.click(shipmentBox("PRO-2011"));
    expect(screen.getByText("11 of 12 shipments")).toBeInTheDocument();

    await waitFor(() => expect(confirmButton()).toBeEnabled());
    await user.click(confirmButton());
    await waitFor(() =>
      expect(onConfirm).toHaveBeenCalledWith(
        { shipmentIds: ids.slice(0, 11) },
        "",
        `sha256:${ids.slice(0, 11).join("+")}`,
      ),
    );
  });
});

/*
The chat card reads its proposals from /assistant/threads/{id}/proposals/,
which marshals toolschema.Field with `resource` and `choices` omitted when
empty.
*/
describe("assistant proposal fields", () => {
  it("keeps a subset field's choices from the thread's proposals", () => {
    const parsed = assistantProposalSchema.parse({
      id: "aprop_1",
      toolName: "transfer_to_billing",
      arguments: { shipmentIds: [SHP_1, SHP_2] },
      rationale: "",
      autonomyTier: "Propose",
      status: "Pending",
      createdAt: 1_790_000_000,
      fields: [
        {
          name: "shipmentIds",
          label: "Shipment ids",
          description: "",
          kind: "RecordSubset",
          required: true,
          options: [],
          readOnly: false,
          resource: "shipment",
          choices: [
            { id: SHP_1, label: "PRO-1001" },
            { id: SHP_2, label: SHP_2 },
          ],
        },
        {
          name: "billType",
          label: "Bill type",
          description: "",
          kind: "Choice",
          required: false,
          options: ["Invoice"],
          readOnly: false,
        },
      ],
    });

    expect(parsed.fields[0]?.choices).toEqual([
      { id: SHP_1, label: "PRO-1001" },
      { id: SHP_2, label: SHP_2 },
    ]);
    expect(parsed.fields[1]?.choices ?? null).toBeNull();
  });
});
