import type { EditorView } from "@codemirror/view";
import { act, cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { rulesSync } from "@trenova/shared/rules/runtime";
import { tariffTest } from "@trenova/shared/rules/tariff";
import { EMPTY_TARIFF_CONTEXT } from "@trenova/shared/types/tariff";
import { Suspense } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { TariffEditor } from "../tariff-editor";
import { findReferences } from "../tariff-references";

vi.mock("@/hooks/use-resolved-theme", () => ({ useResolvedTheme: () => "light" }));

afterEach(cleanup);

const SOURCE = `tariff "Regional"
input rate: Money/mi = $2.85/mi
let miles = shipment.distance
if miles > 5_000 mi {
  charge "Long haul" = $100
}
charge "Linehaul" (quantity: miles, rate: rate)

test "a short haul" {
  given shipment.distance = 100 mi
  expect total == $285
}
`;

function mount(props: Partial<Parameters<typeof TariffEditor>[0]> = {}) {
  let view: EditorView | null = null;
  render(
    <Suspense fallback={null}>
      <TariffEditor
        kind="tariff"
        value={SOURCE}
        onChange={() => undefined}
        context={EMPTY_TARIFF_CONTEXT}
        onView={(next) => {
          view = next;
        }}
        {...props}
      />
    </Suspense>,
  );
  return () => view!;
}

describe("TariffEditor", () => {
  it("lists every use of a name in a dialog and moves the cursor to the chosen one", async () => {
    const user = userEvent.setup();
    const view = mount();
    const declaration = SOURCE.indexOf("miles =");
    act(() => {
      view().dispatch({ selection: { anchor: SOURCE.indexOf("miles,") + 1 } });
      findReferences(view());
    });
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByText("References to miles")).toBeInTheDocument();
    const rows = within(dialog)
      .getAllByRole("button")
      .filter((row) => row.hasAttribute("data-reference-row"));
    expect(rows).toHaveLength(3);
    await user.click(rows[0]!);
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(view().state.selection.main.from).toBe(declaration);
    expect(view().state.selection.main.to).toBe(declaration + "miles".length);
  });

  it("marks the lines no test reaches from the coverage it is given", () => {
    const coverage = tariffTest(rulesSync(), {
      source: SOURCE,
      context: EMPTY_TARIFF_CONTEXT,
    }).coverage;
    const view = mount({ coverage });
    expect(view().dom.querySelectorAll(".cm-tariff-coverage-gap").length).toBeGreaterThan(0);
  });

  it("draws no coverage gutter without coverage", () => {
    const view = mount();
    expect(view().dom.querySelector(".cm-tariff-coverage-gutter")).toBeNull();
  });

  it("keeps what was typed after a value that rendered behind the editor", async () => {
    let view: EditorView | null = null;
    const editor = (value: string) => (
      <Suspense fallback={null}>
        <TariffEditor
          kind="tariff"
          value={value}
          onChange={() => undefined}
          context={EMPTY_TARIFF_CONTEXT}
          onView={(next) => {
            view = next;
          }}
        />
      </Suspense>
    );
    const type = (text: string) =>
      act(() => {
        const at = view!.state.doc.length;
        view!.dispatch({
          changes: { from: at, insert: text },
          selection: { anchor: at + text.length },
          userEvent: "input.type",
        });
      });
    const { rerender } = render(editor("round to"));
    type(" c");
    type("e");
    rerender(editor("round to c"));
    type("nts");
    rerender(editor("round to cents"));
    await act(() => new Promise((resolve) => setTimeout(resolve, 600)));
    expect(view!.state.doc.toString()).toBe("round to cents");
  });
});
