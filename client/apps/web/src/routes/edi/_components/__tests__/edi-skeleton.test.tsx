import { cleanup, render, screen, within } from "@testing-library/react";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { afterEach, describe, expect, it } from "vitest";
import { EDIPageSkeleton } from "../edi-skeleton";
import type { EDIPageKind } from "../edi-types";

const initialPermissions = usePermissionStore.getState();

afterEach(() => {
  cleanup();
  usePermissionStore.setState(initialPermissions, true);
});

const hidden = { hidden: true } as const;

function renderKind(kind: EDIPageKind, title = "EDI page", searchParams = "") {
  return render(
    <NuqsTestingAdapter searchParams={searchParams}>
      <EDIPageSkeleton kind={kind} title={title} />
    </NuqsTestingAdapter>,
  );
}

function allowUpdate(allowed: boolean) {
  usePermissionStore.setState({ hasPermission: () => allowed } as never);
}

describe("EDIPageSkeleton", () => {
  it("draws the operations overview for the overview page", () => {
    renderKind("overview");

    expect(screen.getByLabelText("Loading EDI operations")).toHaveAttribute("aria-busy", "true");
    expect(screen.queryByRole("table")).toBeNull();
  });

  // A table page's outline has the table's columns, with the selection column
  // only where the reader may act on rows in bulk.
  it.each([
    ["messages", false, 7],
    ["messages", true, 8],
    ["outbound", true, 6],
    ["inbound", true, 7],
    ["partners", true, 9],
    ["flows", true, 7],
  ] as const)("draws the %s table (update %s) with %i columns", (kind, canUpdate, columns) => {
    allowUpdate(canUpdate);
    renderKind(kind, "EDI Messages");

    const root = screen.getByLabelText("Loading EDI Messages");
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(within(root).getAllByRole("columnheader", hidden)).toHaveLength(columns);
  });

  it("frames a table page rather than bleeding it to the page edge", () => {
    allowUpdate(false);
    renderKind("mapping-profiles", "EDI Mapping Profiles");

    const root = screen.getByLabelText("Loading EDI Mapping Profiles");
    expect(root).not.toHaveAttribute("data-slot", "data-table");
    expect(root).toHaveTextContent("");
  });
});
