import { useGuardedRowActions } from "@/hooks/use-pending-actions";
import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { Row, RowAction } from "@trenova/shared/types/data-table";
import type { Shipment } from "@trenova/shared/types/shipment";
import { afterEach, describe, expect, it, vi } from "vitest";
import { QuickActionsBlock } from "../expanded-row/quick-actions-block";

vi.mock("../expanded-row/comment-stack", () => ({
  CommentBlock: () => null,
}));

afterEach(cleanup);

function deferred() {
  let resolve!: () => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<void>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

function makeRow(id: string): Row<Shipment> {
  return { id, original: { id, status: "ReadyToInvoice" } as Shipment } as Row<Shipment>;
}

function Harness({ rows, actions }: { rows: Row<Shipment>[]; actions: RowAction<Shipment>[] }) {
  const guarded = useGuardedRowActions(actions);
  return (
    <>
      {rows.map((row) => (
        <section key={row.id} aria-label={row.id}>
          <QuickActionsBlock row={row} actions={guarded} />
        </section>
      ))}
    </>
  );
}

function buttonIn(rowId: string, name: string) {
  const section = screen.getByRole("region", { name: rowId });
  return Array.from(section.querySelectorAll("button")).find((button) =>
    button.textContent?.includes(name),
  )!;
}

describe("guarded quick actions", () => {
  it("fires an async action once no matter how often it is clicked before it settles", async () => {
    const pending = deferred();
    const transfer = vi.fn(() => pending.promise);
    const edit = vi.fn();
    const actions: RowAction<Shipment>[] = [
      { id: "edit", label: "Edit", onClick: edit },
      { id: "transfer-to-billing", label: "Transfer to Billing", onClick: transfer },
    ];
    render(<Harness rows={[makeRow("shp_1")]} actions={actions} />);

    fireEvent.click(buttonIn("shp_1", "Transfer to Billing"));
    fireEvent.click(buttonIn("shp_1", "Transfer to Billing"));
    fireEvent.click(buttonIn("shp_1", "Transfer to Billing"));

    expect(transfer).toHaveBeenCalledOnce();
    expect(buttonIn("shp_1", "Transfer to Billing")).toBeDisabled();
    expect(buttonIn("shp_1", "Transfer to Billing")).toHaveAttribute("aria-busy", "true");
    expect(buttonIn("shp_1", "Edit")).toBeDisabled();
    expect(buttonIn("shp_1", "Edit")).not.toHaveAttribute("aria-busy", "true");

    await act(async () => {
      pending.resolve();
      await pending.promise;
    });

    expect(buttonIn("shp_1", "Transfer to Billing")).toBeEnabled();
    expect(buttonIn("shp_1", "Edit")).toBeEnabled();

    fireEvent.click(buttonIn("shp_1", "Transfer to Billing"));
    expect(transfer).toHaveBeenCalledTimes(2);
  });

  it("releases the row when the action's promise rejects", async () => {
    const pending = deferred();
    const transfer = vi.fn(() => pending.promise);
    render(
      <Harness
        rows={[makeRow("shp_1")]}
        actions={[{ id: "transfer-to-billing", label: "Transfer to Billing", onClick: transfer }]}
      />,
    );

    fireEvent.click(buttonIn("shp_1", "Transfer to Billing"));
    expect(buttonIn("shp_1", "Transfer to Billing")).toBeDisabled();

    await act(async () => {
      pending.reject(new Error("Shipment has already been transferred to billing"));
      await pending.promise.catch(() => undefined);
    });

    expect(buttonIn("shp_1", "Transfer to Billing")).toBeEnabled();
  });

  it("locks only the row whose action is running", () => {
    const pending = deferred();
    const transfer = vi.fn(() => pending.promise);
    render(
      <Harness
        rows={[makeRow("shp_1"), makeRow("shp_2")]}
        actions={[{ id: "transfer-to-billing", label: "Transfer to Billing", onClick: transfer }]}
      />,
    );

    fireEvent.click(buttonIn("shp_1", "Transfer to Billing"));

    expect(buttonIn("shp_1", "Transfer to Billing")).toBeDisabled();
    expect(buttonIn("shp_2", "Transfer to Billing")).toBeEnabled();

    fireEvent.click(buttonIn("shp_2", "Transfer to Billing"));
    expect(transfer).toHaveBeenCalledTimes(2);
  });

  it("does not lock the row for an action that returns nothing", () => {
    const edit = vi.fn();
    render(
      <Harness
        rows={[makeRow("shp_1")]}
        actions={[{ id: "edit", label: "Edit", onClick: edit }]}
      />,
    );

    fireEvent.click(buttonIn("shp_1", "Edit"));
    fireEvent.click(buttonIn("shp_1", "Edit"));

    expect(edit).toHaveBeenCalledTimes(2);
    expect(buttonIn("shp_1", "Edit")).toBeEnabled();
  });
});
