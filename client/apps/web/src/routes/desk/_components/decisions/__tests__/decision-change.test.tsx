import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { PendingProposalFieldsFragment } from "@trenova/graphql/generated/graphql";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import {
  field,
  message,
  preview,
  reason,
  record,
  warning,
} from "@/components/assistant/__tests__/preview-fixtures";
import type { ProposalPreview } from "@/lib/graphql/agent-preview";
import { DecisionChange } from "../decision-change";
import type { PendingProposalNode } from "../use-pending-decisions";
import { proposalNode } from "./pending-fixtures";

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

type ParameterField = PendingProposalFieldsFragment["parameterFields"][number];

const SHIPMENT_FIELD: ParameterField = {
  name: "shipment",
  label: "Shipment",
  description: "The shipment, in the shape the shipment form sends.",
  kind: "JSON",
  required: true,
  options: [],
  minimum: null,
  maximum: null,
  maxLength: null,
  readOnly: false,
  resource: null,
  choices: null,
};

function node(overrides: Partial<PendingProposalNode> = {}): PendingProposalNode {
  return proposalNode({
    reviewRequired: false,
    toolName: "shipment_create",
    toolParams: { shipment: { customerId: "cus_1", bol: "SEED-BOL-009", moves: [] } },
    rationale: "A copy of SEED-DET-009 for next week.",
    parameterFields: [SHIPMENT_FIELD],
    ...overrides,
  });
}

function renderChange(proposal: PendingProposalNode, data: ProposalPreview | null) {
  const onModify = vi.fn();
  const onRetry = vi.fn();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <DecisionChange
          node={proposal}
          entry={
            data === null ? undefined : { proposalId: proposal.id, preview: data, error: null }
          }
          reading={false}
          failed={data === null}
          onRetry={onRetry}
          leaving={null}
          onModify={onModify}
          canChange
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return { onModify, onRetry };
}

const refused = () =>
  preview({ tool: "shipment_create", warnings: [warning({ reasons: [reason()] })] });

describe("DecisionChange, one change read whole", () => {
  it("lists each record with the value it sets and says the check came back clear", () => {
    renderChange(
      node({ toolName: "assign_driver" }),
      preview({
        changes: [
          record({
            label: "S-1001",
            fields: [field({ label: "Driver", before: null, after: "Marcus Hill" })],
          }),
          record({
            label: "S-1002",
            entityId: "shp_2",
            fields: [field({ label: "Driver", before: null, after: "Dana Ortiz" })],
          }),
        ],
      }),
    );

    expect(screen.getByText("S-1001")).toBeInTheDocument();
    expect(screen.getByText("Driver")).toBeInTheDocument();
    expect(screen.getByText("Marcus Hill")).toBeInTheDocument();
    expect(screen.getByText("All 2 shipments would go through as shown.")).toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "Before you decide" })).toBeNull();
  });

  it("marks the records the write would refuse and says how many go through", () => {
    renderChange(
      node({ toolName: "invoice_post" }),
      preview({
        changes: [
          record({ label: "INV-1", fields: [field({ label: "Status", after: "Posted" })] }),
          record({
            label: "INV-2",
            entityId: "inv_2",
            fields: [
              field({ label: "Status", after: "Posted" }),
              field({
                path: "outcome",
                label: "Outcome",
                after: "Refused: Missing bill-to address",
              }),
            ],
          }),
        ],
      }),
    );

    expect(screen.getByText("Refused")).toBeInTheDocument();
    expect(
      screen.getByText("1 would go through. 1 would be refused: missing bill-to address"),
    ).toBeInTheDocument();
  });

  it("says once why a change can't run, and leaves out the line that says it checked out", () => {
    renderChange(node(), refused());

    const block = screen.getByRole("region", { name: "This can't run as it stands" });
    expect(block).toHaveTextContent("BOL: BOL is already in use by shipment SEED-DET-009");
    expect(screen.getAllByText("BOL is already in use by shipment SEED-DET-009")).toHaveLength(1);
    expect(screen.queryByText(/would go through/)).toBeNull();
  });

  it("opens the editor on the value a reason names", async () => {
    const user = userEvent.setup();
    const { onModify } = renderChange(node(), refused());

    await user.click(screen.getByRole("button", { name: "Change BOL" }));

    expect(onModify).toHaveBeenCalledWith({ param: "shipment.bol", label: "BOL" });
  });

  it("draws a message as it goes out, a warning once above it, and never the raw arguments", () => {
    renderChange(
      node({ toolName: "worker_send_message", parameterFields: [] }),
      preview({
        tool: "worker_send_message",
        warnings: [
          warning({
            code: "sensitive_content",
            message: "The message contains details your organization marks as sensitive.",
          }),
        ],
        changes: [
          record({
            resource: "worker",
            label: "Pamela Rivera",
            operation: "Send",
            fields: [],
            message: message({
              channel: "Dash",
              from: "",
              to: ["Pamela Rivera"],
              subject: "Signed POD needed: PRO S100",
              body: "Hi Pamela, please send the signed proof of delivery for PRO S100.",
            }),
          }),
        ],
      }),
    );

    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Pamela Rivera");
    expect(screen.getByText("Driver portal")).toBeInTheDocument();
    expect(screen.getByText("to Pamela Rivera")).toBeInTheDocument();
    expect(screen.getByText("Signed POD needed: PRO S100")).toBeInTheDocument();
    expect(
      screen.getByText("Hi Pamela, please send the signed proof of delivery for PRO S100."),
    ).toBeInTheDocument();
    expect(screen.getAllByRole("status")).toHaveLength(1);
    expect(screen.queryByText("Arguments the agent sent")).toBeNull();
  });

  it("folds a long reason to its first lines until it is asked for", async () => {
    const user = userEvent.setup();
    renderChange(node({ rationale: "The remittance names both invoices. ".repeat(12) }), preview());

    const more = screen.getByRole("button", { name: "Show more" });
    expect(more).toHaveAttribute("aria-expanded", "false");
    await user.click(more);

    expect(screen.getByRole("button", { name: "Show less" })).toHaveAttribute(
      "aria-expanded",
      "true",
    );
  });

  it("offers to read the change again when it could not be read", async () => {
    const user = userEvent.setup();
    const { onRetry } = renderChange(node(), null);

    expect(
      screen.getByText(
        "What this would change could not be loaded. Approving now is recorded as approved without a preview.",
      ),
    ).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Try again" }));

    expect(onRetry).toHaveBeenCalledTimes(1);
  });
});
