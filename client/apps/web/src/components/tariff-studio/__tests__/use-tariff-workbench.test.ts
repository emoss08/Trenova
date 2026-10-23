import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { rulesSync } from "@trenova/shared/rules/runtime";
import { history, undo } from "@codemirror/commands";
import { EditorState } from "@codemirror/state";
import { EditorView } from "@codemirror/view";
import { describe, expect, it, vi } from "vitest";
import { createElement, type ReactNode } from "react";
import { useTariffEditorBinding, useTariffWorkbench } from "../use-tariff-workbench";

function wrapper({ children }: { children: ReactNode }) {
  return createElement(QueryClientProvider, { client: new QueryClient() }, children);
}

describe("useTariffEditorBinding before the editor has loaded", () => {
  it("writes a fix's changes to the form, last change first so offsets hold", () => {
    let source = "let a = 1\nlet b = 2\n";
    const write = vi.fn((next: string) => {
      source = next;
    });
    const { result } = renderHook(() => useTariffEditorBinding({ read: () => source, write }));
    result.current.applyFix({
      changes: [
        { range: { from: 4, to: 5 }, insert: "first" },
        { range: { from: 14, to: 15 }, insert: "second" },
      ],
    });
    expect(write).toHaveBeenCalledTimes(1);
    expect(source).toBe("let first = 1\nlet second = 2\n");
  });

  it("appends a block after one blank line", () => {
    let source = 'library "org.x"\n\n\n';
    const { result } = renderHook(() =>
      useTariffEditorBinding({ read: () => source, write: (next) => (source = next) }),
    );
    result.current.appendToSource('test "t" {\n  expect 1 == 1\n}\n');
    expect(source).toBe('library "org.x"\n\ntest "t" {\n  expect 1 == 1\n}\n');
  });

  it("writes an edit made outside the editor to the form", () => {
    let source = 'tariff "Old"\ncharge "A" = $1\n';
    const { result } = renderHook(() =>
      useTariffEditorBinding({ read: () => source, write: (next) => (source = next) }),
    );
    result.current.applyEdit({ range: { from: 7, to: 12 }, insert: '"New name"' });
    expect(source).toBe('tariff "New name"\ncharge "A" = $1\n');
  });

  it("keeps the same functions from render to render", () => {
    const { result, rerender } = renderHook(
      ({ text }) => useTariffEditorBinding({ read: () => text, write: () => undefined }),
      { initialProps: { text: "a" } },
    );
    const first = result.current;
    rerender({ text: "b" });
    expect(result.current).toBe(first);
  });
});

describe("useTariffWorkbench", () => {
  it("counts a library's findings and tests from the module's check", () => {
    const source = `library "org.fees"

fn fee() -> Money {
  return $10
}

test "the fee" {
  expect fee() == $10
}
`;
    const { result } = renderHook(
      () =>
        useTariffWorkbench({
          rules: rulesSync(),
          source,
          kind: "library",
          context: null,
          tablesStamp: "",
        }),
      { wrapper },
    );
    expect(result.current.errors).toBe(0);
    expect(result.current.warnings).toBe(0);
    expect(result.current.testCount).toBe(1);
    expect(result.current.check.header).toEqual({ kind: "library", name: "org.fees" });
  });

  it("blocks the tests while the source has errors", () => {
    const { result } = renderHook(
      () =>
        useTariffWorkbench({
          rules: rulesSync(),
          source: 'library "org.fees"\n\nfn fee() -> Money {\n  return nope\n}\n',
          kind: "library",
          context: null,
          tablesStamp: "",
        }),
      { wrapper },
    );
    expect(result.current.errors).toBeGreaterThan(0);
    expect(result.current.tests).toEqual({ status: "blocked", errors: result.current.errors });
  });
});

describe("useTariffWorkbench on an empty file", () => {
  it("reports no findings, so the Studio shows where to start instead", () => {
    const { result } = renderHook(
      () =>
        useTariffWorkbench({
          rules: rulesSync(),
          source: "  \n",
          kind: "tariff",
          context: null,
          tablesStamp: "",
        }),
      { wrapper },
    );
    expect(result.current.check.diagnostics).toEqual([]);
    expect(result.current.errors).toBe(0);
    expect(result.current.tests).toEqual({ status: "empty" });
  });
});

describe("useTariffWorkbench with properties", () => {
  const source = `tariff "LTL"
charge "Freight" = (shipment.weight ?? 0 lb) * $0.30/lb
if shipment.distance > 5_000 mi {
  charge "Long haul" = $100
}

property "heavier never bills less" {
  vary shipment.weight in 1 lb..40_000 lb
  expect total nondecreasing in shipment.weight
}
`;

  it("counts a property as a test, runs it, and hands the editor the coverage", async () => {
    vi.useFakeTimers();
    try {
      const { result } = renderHook(
        () =>
          useTariffWorkbench({
            rules: rulesSync(),
            source,
            kind: "tariff",
            context: null,
            tablesStamp: "",
          }),
        { wrapper },
      );
      expect(result.current.testCount).toBe(1);
      await act(async () => {
        vi.advanceTimersByTime(500);
      });
      const tests = result.current.tests;
      expect(tests.status).toBe("ran");
      if (tests.status !== "ran") return;
      expect(tests.run.tests.map((test) => [test.kind, test.status])).toEqual([
        ["property", "passed"],
      ]);
      expect(result.current.coverage).toEqual(tests.run.coverage);
      expect(result.current.coverage?.regions.some((region) => region.hits === 0)).toBe(true);
    } finally {
      vi.useRealTimers();
    }
  });

  it("hands the editor no coverage while the source has errors", () => {
    const { result } = renderHook(
      () =>
        useTariffWorkbench({
          rules: rulesSync(),
          source: `${source}\nlet broken = nope\n`,
          kind: "tariff",
          context: null,
          tablesStamp: "",
        }),
      { wrapper },
    );
    expect(result.current.coverage).toBeNull();
  });
});

describe("useTariffEditorBinding with the editor", () => {
  it("applies an edit made outside the editor as one undoable change, leaving focus where it is", () => {
    const write = vi.fn();
    const view = new EditorView({
      state: EditorState.create({ doc: 'tariff "Old"\n', extensions: [history()] }),
      parent: document.body,
    });
    const { result } = renderHook(() =>
      useTariffEditorBinding({ read: () => view.state.doc.toString(), write }),
    );
    result.current.setEditorView(view);

    result.current.applyEdit({ range: { from: 7, to: 12 }, insert: '"N"' });
    result.current.applyEdit({ range: { from: 7, to: 10 }, insert: '"Ne"' });

    expect(view.state.doc.toString()).toBe('tariff "Ne"\n');
    expect(view.hasFocus).toBe(false);
    expect(write).not.toHaveBeenCalled();
    undo(view);
    expect(view.state.doc.toString()).toBe('tariff "Old"\n');
    view.destroy();
  });
});
