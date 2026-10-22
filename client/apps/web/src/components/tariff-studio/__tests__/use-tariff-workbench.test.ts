import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import { rulesSync } from "@trenova/shared/rules/runtime";
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
