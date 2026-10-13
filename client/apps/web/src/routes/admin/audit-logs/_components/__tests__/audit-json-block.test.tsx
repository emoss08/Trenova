import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AuditJsonBlock } from "../audit-json-block";

const copy = vi.hoisted(() => vi.fn());

vi.mock("@/hooks/use-copy-to-clipboard", () => ({
  useCopyToClipboard: () => ({ copy, text: null }),
}));

beforeEach(() => {
  vi.useFakeTimers();
  copy.mockReset();
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

function line(container: HTMLElement, index: number): HTMLElement {
  return container.querySelector<HTMLElement>(`[data-line="${index}"]`) as HTMLElement;
}

describe("AuditJsonBlock", () => {
  it("shows the record as indented JSON, coloured, with nothing to search or copy by default", () => {
    const { container } = render(<AuditJsonBlock value={{ status: "Ready", stops: 2 }} />);

    expect(container.querySelector("pre")?.textContent).toBe(
      '{\n  "status": "Ready",\n  "stops": 2\n}',
    );
    expect(container.querySelector(".text-syntax-property")?.textContent).toBe('"status"');
    expect(screen.queryByPlaceholderText("Search JSON...")).toBeNull();
  });

  it("marks a search's matches after the typing settles and says how many there are", () => {
    const { container } = render(
      <AuditJsonBlock value={{ status: "Ready", readyAt: 1, note: "not yet" }} searchable />,
    );

    fireEvent.change(screen.getByPlaceholderText("Search JSON..."), { target: { value: "READY" } });
    expect(container.querySelectorAll("mark")).toHaveLength(0);

    act(() => {
      vi.advanceTimersByTime(150);
    });

    expect([...container.querySelectorAll("mark")].map((m) => m.textContent)).toEqual([
      "Ready",
      "ready",
    ]);
    expect(screen.getByText("2 matches")).toBeTruthy();

    fireEvent.change(screen.getByPlaceholderText("Search JSON..."), { target: { value: "note" } });
    act(() => {
      vi.advanceTimersByTime(150);
    });
    expect(screen.getByText("1 match")).toBeTruthy();
  });

  it("badges each line whose value was redacted, and only those", () => {
    const { container } = render(
      <AuditJsonBlock value={{ password: "[REDACTED]", token: "********", name: "Ann" }} />,
    );

    expect(line(container, 1).textContent).toBe('  "password": "[REDACTED]",Sensitive');
    expect(line(container, 2).textContent).toBe('  "token": "********",Sensitive');
    expect(line(container, 3).textContent).toBe('  "name": "Ann"');
    expect(screen.getAllByTitle("Sensitive data omitted")).toHaveLength(2);
  });

  it("copies the path of the line clicked, past an empty array above it", () => {
    const { container } = render(
      <AuditJsonBlock value={{ tags: [], customer: { name: "Acme" } }} copyPath />,
    );

    fireEvent.click(line(container, 3));
    fireEvent.click(line(container, 2));
    fireEvent.click(line(container, 0));

    expect(copy.mock.calls).toEqual([
      ["$.customer.name", { withToast: true }],
      ["$.customer", { withToast: true }],
    ]);
  });
});
