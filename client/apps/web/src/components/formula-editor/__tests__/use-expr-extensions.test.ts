import { rulesSync } from "@trenova/shared/rules/runtime";
import { renderHook } from "@testing-library/react";
import type { EditorView } from "@codemirror/view";
import { describe, expect, it, vi } from "vitest";
import { useExprExtensions } from "../use-expr-extensions";
import { knownFor } from "@/components/formula-editor/__tests__/formula-fixtures";

describe("useExprExtensions", () => {
  it("keeps the extension array stable and reconfigures the live view when identifiers change", () => {
    const dispatch = vi.fn();
    const viewRef = { current: { dispatch } as unknown as EditorView };
    const first = knownFor();
    const second = knownFor([{ name: "fuelPct", type: "Number" }]);

    const { result, rerender } = renderHook(
      ({ known }) => useExprExtensions(rulesSync(), known, true, viewRef),
      { initialProps: { known: first } },
    );
    const initial = result.current;
    expect(dispatch).not.toHaveBeenCalled();

    rerender({ known: second });

    expect(result.current).toBe(initial);
    expect(dispatch).toHaveBeenCalledTimes(1);
    const effects = dispatch.mock.calls[0]?.[0]?.effects;
    expect(effects).toBeTruthy();
  });
});
