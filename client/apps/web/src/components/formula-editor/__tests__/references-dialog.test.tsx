import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { rulesSync } from "@trenova/shared/rules/runtime";
import { tariffReferences } from "@trenova/shared/rules/tariff";
import { EMPTY_TARIFF_CONTEXT, type TariffContext } from "@trenova/shared/types/tariff";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ReferencesDialog, referencePreview } from "../references-dialog";

afterEach(cleanup);

const LIBRARY = 'library "org.fees"\n\n/// A stop fee.\nfn fee() -> Money = $5\n';
const SOURCE = `tariff "Fees"
use org.fees.{ fee }
use std.ltl.{ deficit }

charge "Stop" = fee()
charge "Second stop" = fee() + $1
`;
const context: TariffContext = {
  ...EMPTY_TARIFF_CONTEXT,
  libraries: [{ path: "org.fees", version: "v1", source: LIBRARY }],
};

const referencesAt = (needle: string) =>
  tariffReferences(rulesSync(), {
    source: SOURCE,
    context,
    kind: "tariff",
    offset: SOURCE.indexOf(needle) + 1,
  });

describe("referencePreview", () => {
  it("cuts the use's line around it", () => {
    expect(
      referencePreview('let a = 1\n  charge "x" = total + a\n', {
        range: { from: 33, to: 34 },
        kind: "read",
        module: null,
      }),
    ).toEqual({ line: 2, before: 'charge "x" = total + ', match: "a", after: "" });
  });

  it("has no line for a use whose source is not here", () => {
    expect(
      referencePreview(null, { range: { from: 1, to: 2 }, kind: "declaration", module: "std.ltl" }),
    ).toEqual({ line: null, before: "", match: "", after: "" });
  });
});

describe("ReferencesDialog", () => {
  it("groups the uses in this file apart from the declaration in its library", () => {
    const references = referencesAt("fee()");
    render(
      <ReferencesDialog
        request={{ name: "fee", offset: 0, references }}
        source={SOURCE}
        context={context}
        canOpenLibraries
        onPick={vi.fn()}
        onClose={vi.fn()}
      />,
    );
    expect(screen.getByText("References to fee")).toBeInTheDocument();
    const here = screen.getByRole("region", { name: "In this file" });
    expect(
      within(here)
        .getAllByRole("button")
        .map((row) => row.textContent),
    ).toEqual([
      "Ln 2use org.fees.{ fee }Read",
      'Ln 5charge "Stop" = fee()Read',
      'Ln 6charge "Second stop" = fee() + $1Read',
    ]);
    const libraries = screen.getByRole("region", { name: "In libraries" });
    expect(within(libraries).getByText("org.fees")).toBeInTheDocument();
    expect(within(libraries).getByRole("button")).toHaveTextContent(
      "Ln 4fn fee() -> Money = $5Declared",
    );
  });

  it("goes to the chosen use, and moves between uses with the arrow keys", async () => {
    const user = userEvent.setup();
    const pick = vi.fn();
    const references = referencesAt("fee()");
    render(
      <ReferencesDialog
        request={{ name: "fee", offset: 0, references }}
        source={SOURCE}
        context={context}
        canOpenLibraries
        onPick={pick}
        onClose={vi.fn()}
      />,
    );
    const rows = screen
      .getAllByRole("button")
      .filter((row) => row.hasAttribute("data-reference-row"));
    rows[0]!.focus();
    await user.keyboard("{ArrowDown}");
    expect(document.activeElement).toBe(rows[1]);
    await user.keyboard("{End}");
    expect(document.activeElement).toBe(rows.at(-1));
    await user.keyboard("{Home}{Enter}");
    expect(pick).toHaveBeenCalledWith(references.find((reference) => reference.module === null));
  });

  it("names a standard library declaration from the catalog and leaves it unopened without a page to open it", () => {
    const references = referencesAt("deficit");
    render(
      <ReferencesDialog
        request={{ name: "deficit", offset: 0, references }}
        source={SOURCE}
        context={context}
        canOpenLibraries={false}
        onPick={vi.fn()}
        onClose={vi.fn()}
      />,
    );
    const libraries = screen.getByRole("region", { name: "In libraries" });
    expect(within(libraries).getByText("std.ltl")).toBeInTheDocument();
    expect(within(libraries).queryByRole("button")).toBeNull();
    expect(within(libraries).getByText(/^fn deficit\(/)).toBeInTheDocument();
  });

  it("is closed without a request", () => {
    render(
      <ReferencesDialog
        request={null}
        source={SOURCE}
        context={context}
        canOpenLibraries
        onPick={vi.fn()}
        onClose={vi.fn()}
      />,
    );
    expect(screen.queryByRole("dialog")).toBeNull();
  });
});
