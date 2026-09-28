import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import { MessagePreview } from "../proposal-preview/message-preview";
import { MoneyPreview } from "../proposal-preview/money-preview";
import { PlanPreview } from "../proposal-preview/plan-preview";
import { ProposalPreview } from "../proposal-preview/proposal-preview";
import { ValueChange } from "../proposal-preview/value-change";
import {
  field,
  message,
  money,
  planPreview,
  preview,
  reason,
  record,
  warning,
} from "./preview-fixtures";

function renderIn(ui: ReactElement) {
  return render(<MemoryRouter>{ui}</MemoryRouter>);
}

describe("ValueChange", () => {
  it("strikes the old value through and shows the new one after it", () => {
    renderIn(<ValueChange change={field()} operation="Update" />);

    const before = screen.getByText("Old value");
    expect(before).toHaveClass("line-through");
    expect(screen.getByText("New value")).not.toHaveClass("line-through");
    expect(screen.getByText("becomes")).toBeInTheDocument();
  });

  // A create has no "before": an em dash struck through would read as a
  // value that existed and is going away.
  it("shows only the new value for a record the write creates", () => {
    const { container } = renderIn(
      <ValueChange change={field({ before: null })} operation="Create" />,
    );

    expect(screen.getByText("New value")).toBeInTheDocument();
    expect(container.textContent).not.toContain("—");
    expect(screen.queryByText("becomes")).not.toBeInTheDocument();
  });

  it("draws an absent side as an em dash rather than a blank", () => {
    renderIn(<ValueChange change={field({ before: null })} operation="Update" />);

    expect(screen.getByText("—")).toBeInTheDocument();
  });

  // A value above the reader's data access carries neither side, and says so
  // rather than disappearing — a blank row would read as "nothing changes".
  it("says a withheld value is hidden by the reader's data access", () => {
    const { container } = renderIn(
      <ValueChange
        change={field({ withheld: true, before: null, after: null })}
        operation="Update"
      />,
    );

    expect(screen.getByText("Hidden by your data access")).toBeInTheDocument();
    expect(container.textContent).not.toContain("becomes");
  });

  it("says what a value was when proposed when it has moved since", () => {
    renderIn(
      <ValueChange
        change={field({
          valueType: "money",
          before: 42000,
          after: 45000,
          changedSinceProposed: true,
          proposedBefore: 40000,
        })}
        operation="Update"
      />,
    );

    expect(screen.getByText("Was $40,000.00 when proposed")).toBeInTheDocument();
    expect(screen.getByText("$42,000.00")).toHaveClass("line-through");
  });

  it("says so when a moved value was empty when proposed", () => {
    renderIn(
      <ValueChange
        change={field({ changedSinceProposed: true, proposedBefore: null })}
        operation="Update"
      />,
    );

    expect(screen.getByText("Was empty when proposed")).toBeInTheDocument();
  });

  it("names a referenced record by its label and links it through the record registry", () => {
    renderIn(
      <ValueChange
        change={field({
          path: "customerId",
          label: "Customer",
          before: "cus_old",
          after: "cus_new",
          beforeRef: {
            resource: "customer",
            id: "cus_old",
            label: "Old Freight Co",
            record: { entityType: "customer", id: "cus_old" },
            withheld: false,
          },
          afterRef: {
            resource: "customer",
            id: "cus_new",
            label: "Peak Distributing",
            record: { entityType: "customer", id: "cus_new" },
            withheld: false,
          },
        })}
        operation="Update"
      />,
    );

    const link = screen.getByRole("link", { name: "Peak Distributing" });
    expect(link).toHaveAttribute(
      "href",
      "/billing/configuration-files/customers?panelType=edit&panelEntityId=cus_new",
    );
    expect(screen.getByText("Old Freight Co")).toHaveClass("line-through");
    expect(document.body.textContent).not.toContain("cus_new");
  });

  it("hides a referenced record the reader may not read", () => {
    renderIn(
      <ValueChange
        change={field({
          after: "wrk_1",
          afterRef: {
            resource: "worker",
            id: "wrk_1",
            label: null,
            record: null,
            withheld: true,
          },
        })}
        operation="Update"
      />,
    );

    expect(screen.getByText("Hidden by your data access")).toBeInTheDocument();
    expect(document.body.textContent).not.toContain("wrk_1");
  });

  it("says which plan step a projected value starts from", () => {
    renderIn(<ValueChange change={field({ projectedFromStep: 2 })} operation="Update" />);

    expect(screen.getByText("As step 2 leaves it")).toBeInTheDocument();
  });
});

describe("MoneyPreview", () => {
  it("shows each line, the totals and a signed difference", () => {
    renderIn(<MoneyPreview money={money()} />);

    expect(screen.getByText("Detention")).toBeInTheDocument();
    expect(screen.getByText("$1,200.00")).toBeInTheDocument();
    expect(screen.getByText("$1,350.00")).toBeInTheDocument();
    expect(screen.getByText("+$150.00")).toBeInTheDocument();
  });

  it("signs a reduction", () => {
    renderIn(
      <MoneyPreview
        money={money({ totalBefore: "500.00", totalAfter: "450.00", delta: "-50.00" })}
      />,
    );

    expect(screen.getByText("-$50.00")).toBeInTheDocument();
  });

  it("shows no amount at all when the money is above the reader's data access", () => {
    renderIn(<MoneyPreview money={money({ withheld: true })} />);

    expect(screen.getByText("Hidden by your data access")).toBeInTheDocument();
    expect(document.body.textContent).not.toContain("$");
  });

  // Intl throws on an empty currency code; the server may leave it blank.
  it("reads a blank currency as dollars rather than failing to draw", () => {
    renderIn(<MoneyPreview money={money({ currency: "" })} />);

    expect(screen.getByText("+$150.00")).toBeInTheDocument();
  });
});

describe("MessagePreview", () => {
  it("shows the recipients the send would reach and the rendered body", () => {
    renderIn(
      <MessagePreview
        message={message({ to: ["ap@customer.test", "ops@customer.test"], cc: ["me@x.test"] })}
      />,
    );

    expect(screen.getByText("Email")).toBeInTheDocument();
    expect(screen.getByText("ap@customer.test, ops@customer.test")).toBeInTheDocument();
    expect(screen.getByText("me@x.test")).toBeInTheDocument();
    expect(screen.getByText("Delivery update")).toBeInTheDocument();
    expect(screen.getByText("Your load delivered at 14:02.")).toBeInTheDocument();
  });

  it("names who can see a comment and a body cut short", () => {
    renderIn(
      <MessagePreview
        message={message({ channel: "Comment", visibility: "customer", bodyTruncated: true })}
      />,
    );

    expect(screen.getByText("The customer")).toBeInTheDocument();
    expect(
      screen.getByText("The message is longer than a preview keeps; the rest is not shown."),
    ).toBeInTheDocument();
  });
});

describe("ProposalPreview", () => {
  it("heads each record with what happens to it and its linked label", () => {
    renderIn(<ProposalPreview preview={preview()} />);

    expect(screen.getByText("Change")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "S-1001" })).toHaveAttribute(
      "href",
      "/shipment-management/shipments?expanded=shp_1&panelType=edit&panelEntityId=shp_1",
    );
  });

  // Unavailable: the tool cannot say what it would change. The values it
  // would run with are shown under a notice that says exactly that.
  it("puts a warning over the parameters a tool without a preview would run with", () => {
    renderIn(
      <ProposalPreview
        preview={preview({
          coverage: "Unavailable",
          tool: "flag_for_manual_review",
          changes: [
            record({
              resource: "",
              record: null,
              entityId: null,
              label: "flag_for_manual_review",
              operation: "Run",
              version: null,
              fields: [field({ path: "severity", label: "Severity", before: null, after: "High" })],
            }),
          ],
        })}
      />,
    );

    expect(
      screen.getByText(
        "This action can't say exactly what it would change. It would run with the values below.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText("Flag for manual review")).toBeInTheDocument();
    expect(screen.getByText("High")).toBeInTheDocument();
  });

  it("says a stale proposal can only be rejected, and why", () => {
    renderIn(
      <ProposalPreview
        preview={preview({
          stale: true,
          staleness: { pinned: true, proposedVersion: 4, currentVersion: 6, missing: false },
        })}
      />,
    );

    expect(screen.getByText("Changed since it was proposed")).toBeInTheDocument();
    expect(
      screen.getByText(/It can only be rejected now; ask the agent again/u),
    ).toBeInTheDocument();
  });

  it("says the record is gone when it is", () => {
    renderIn(
      <ProposalPreview
        preview={preview({
          stale: true,
          staleness: { pinned: true, proposedVersion: 4, currentVersion: 0, missing: true },
        })}
      />,
    );

    expect(
      screen.getByText("The record it would change is gone. It can only be rejected now."),
    ).toBeInTheDocument();
  });

  // Owner decision: approving with hidden parts is allowed; the count is shown.
  it("counts what is hidden from the reader and hides a withheld record whole", () => {
    renderIn(
      <ProposalPreview
        preview={preview({
          withheldCount: 3,
          changes: [record({ withheld: true, label: "", record: null, fields: [] }), record()],
        })}
      />,
    );

    expect(screen.getByText("3 parts hidden by your data access")).toBeInTheDocument();
    expect(screen.getByText("A record hidden by your data access")).toBeInTheDocument();
  });

  it("translates a warning by its code and falls back to the server's words for an unknown one", () => {
    renderIn(
      <ProposalPreview
        preview={preview({
          warnings: [
            warning({ code: "already_told_customer", message: "ignored English" }),
            warning({ code: "brand_new_code", message: "Something the client has not met." }),
          ],
        })}
      />,
    );

    expect(screen.getByText("The customer has already been told about this.")).toBeInTheDocument();
    expect(screen.queryByText("ignored English")).not.toBeInTheDocument();
    expect(screen.getByText("Something the client has not met.")).toBeInTheDocument();
  });

  // The old card said only "This would not go through as it stands." and
  // dropped the reason the server sent. The person needs each reason and a
  // way forward: change the value it names, or have the agent fix it.
  it("lists each reason a refusal names, with a way to change the field or ask the agent", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    const onAskAgent = vi.fn();
    const credit = reason({
      field: "",
      label: "",
      param: "",
      message: "The customer is on credit hold",
    });
    renderIn(
      <ProposalPreview
        preview={preview({ warnings: [warning({ reasons: [reason(), credit] })] })}
        wouldFail={{
          canChange: (param) => param === "shipment.bol",
          onChange,
          onAskAgent,
        }}
      />,
    );

    expect(screen.getByText("This would not go through as it stands")).toBeInTheDocument();
    expect(screen.getByText("BOL is already in use by shipment SEED-DET-009")).toBeInTheDocument();
    expect(screen.getByText("The customer is on credit hold")).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: /^Change / })).toHaveLength(1);

    await user.click(screen.getByRole("button", { name: "Change BOL" }));
    expect(onChange).toHaveBeenCalledWith(reason());

    await user.click(screen.getByRole("button", { name: "Ask the agent to fix it" }));
    expect(onAskAgent).toHaveBeenCalledWith([reason(), credit]);
  });

  it("shows the server's reasons when it structured none, and offers nothing to change", () => {
    renderIn(
      <ProposalPreview
        preview={preview({
          warnings: [
            warning({
              reasons: [],
              message:
                "This would be refused as it stands: validation failed:\n- Rate not found\n- Customer is inactive",
            }),
          ],
        })}
        wouldFail={{ canChange: () => true, onChange: vi.fn() }}
      />,
    );

    expect(screen.getByText("Rate not found")).toBeInTheDocument();
    expect(screen.getByText("Customer is inactive")).toBeInTheDocument();
    expect(screen.queryByText("This would not go through as it stands.")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^Change / })).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Ask the agent to fix it" }),
    ).not.toBeInTheDocument();
  });

  it("keeps a compact preview short and opens the rest on request", async () => {
    const user = userEvent.setup();
    const fields = Array.from({ length: 6 }, (_, index) =>
      field({ path: `f${index}`, label: `Field ${index}`, after: `Value ${index}` }),
    );
    renderIn(
      <ProposalPreview
        preview={preview({
          changes: [
            record({ fields }),
            record({ entityId: "shp_2", label: "S-1002" }),
            record({ entityId: "shp_3", label: "S-1003" }),
          ],
        })}
        density="compact"
      />,
    );

    expect(screen.getByText("Field 3")).toBeInTheDocument();
    expect(screen.queryByText("Field 4")).not.toBeInTheDocument();
    expect(screen.queryByText("S-1003")).not.toBeInTheDocument();
    expect(screen.getByText("2 more values")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Show all changes" }));

    expect(screen.getByText("Field 5")).toBeInTheDocument();
    expect(screen.getByText("S-1003")).toBeInTheDocument();
  });

  it("renders the rendered message and the money a write would move", () => {
    renderIn(
      <ProposalPreview
        preview={preview({
          changes: [
            record({ operation: "Send", fields: [], message: message() }),
            record({ fields: [], money: money() }),
          ],
        })}
      />,
    );

    expect(screen.getByText("Your load delivered at 14:02.")).toBeInTheDocument();
    expect(screen.getByText("+$150.00")).toBeInTheDocument();
  });

  it("says when nothing would change", () => {
    renderIn(<ProposalPreview preview={preview({ changes: [] })} />);

    expect(screen.getByText("Nothing would change.")).toBeInTheDocument();
  });
});

describe("PlanPreview", () => {
  // A later step on a record an earlier one changes starts from what that
  // step leaves. The step says so beside it; the warning that says the same
  // is not repeated.
  it("notes each step's dependency once and in place of the warning", () => {
    renderIn(<PlanPreview plan={planPreview()} />);

    const steps = screen.getAllByRole("listitem");
    expect(steps).toHaveLength(2);
    expect(within(steps[0]).queryByText(/Uses the record step/u)).not.toBeInTheDocument();
    expect(within(steps[1]).getByText("Uses the record step 1 changes")).toBeInTheDocument();
    expect(within(steps[1]).getByText("As step 1 leaves it")).toBeInTheDocument();
    expect(
      within(steps[1]).queryByText(/it is shown as it would be after that step/u),
    ).not.toBeInTheDocument();
  });

  it("uses the surface's sentence for each step when it has one", () => {
    renderIn(
      <PlanPreview plan={planPreview()} stepTitle={(id) => <span>{`Sentence for ${id}`}</span>} />,
    );

    expect(screen.getByText("Sentence for aprop_2")).toBeInTheDocument();
  });
});
