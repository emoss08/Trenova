import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AnalysisResult, type AnalysisView } from "../analysis-result";

afterEach(cleanup);

const CODE = "df = ar.copy()\nprint('Open', df['Open'].sum())\n";

function analysis(overrides: Partial<AnalysisView> = {}): AnalysisView {
  return {
    runId: "",
    status: "Succeeded",
    title: "Receivables stress by customer",
    code: CODE,
    datasets: [],
    output: "Open 21637.5\n",
    charts: [],
    tables: [],
    figures: [{ label: "Open balance", value: 21637.5, kind: "number", currency: "", note: "" }],
    layout: [],
    error: null,
    failure: "",
    truncated: false,
    ...overrides,
  };
}

describe("AnalysisResult workings", () => {
  it("keeps the code folded, and names its language and length on the header", () => {
    render(<AnalysisResult analysis={analysis()} />);

    const trigger = screen.getByRole("button", { name: /Code/ });
    expect(trigger).toHaveTextContent("Codepython·2 lines");
    expect(trigger).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("df = ar.copy()")).not.toBeInTheDocument();
  });

  it("shows the code highlighted and line by line once unfolded", () => {
    render(<AnalysisResult analysis={analysis()} />);

    fireEvent.click(screen.getByRole("button", { name: /Code/ }));

    const block = document.querySelector('pre[data-language="python"]');
    expect(block).not.toBeNull();
    expect(block?.querySelectorAll("[data-line]")).toHaveLength(2);
    const strings = screen.getAllByText("'Open'", { selector: "span" });
    expect(strings).toHaveLength(2);
    for (const string of strings) expect(string).toHaveClass("text-syntax-string");
  });

  it("copies the code without its trailing blank line, and says so", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { value: { writeText }, configurable: true });
    render(<AnalysisResult analysis={analysis()} />);

    fireEvent.click(screen.getByRole("button", { name: /Code/ }));
    fireEvent.click(screen.getByRole("button", { name: "Copy" }));

    expect(await screen.findByRole("button", { name: "Copied" })).toBeInTheDocument();
    expect(writeText).toHaveBeenCalledWith(CODE.trimEnd());
  });

  it("opens what it printed when the analysis showed nothing else", () => {
    render(<AnalysisResult analysis={analysis({ figures: [] })} />);

    expect(screen.getByRole("button", { name: /What it printed/ })).toHaveAttribute(
      "aria-expanded",
      "true",
    );
    expect(screen.getByText("Open 21637.5")).toBeInTheDocument();
  });

  it("draws no workings panel when there is neither output nor code", () => {
    render(<AnalysisResult analysis={analysis({ code: "\n", output: "" })} />);

    expect(screen.queryByRole("button", { name: /Code/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /What it printed/ })).not.toBeInTheDocument();
  });
});

describe("AnalysisResult charts", () => {
  const chart = {
    kind: "bar" as const,
    title: "Open by customer",
    x: { label: "Customer", kind: "text" as const, values: ["Acme", "Peak"] },
    series: [{ name: "Open", values: [36912.16, 1200] }],
    valueKind: "money" as const,
    currency: "USD",
    stacked: false,
    horizontal: false,
  };

  it("draws each chart the script showed, in the order it showed it", async () => {
    render(
      <AnalysisResult
        analysis={analysis({
          figures: [],
          charts: [chart, { ...chart, title: "Billed by customer" }],
          layout: [
            { kind: "chart", index: 1 },
            { kind: "chart", index: 0 },
          ],
        })}
      />,
    );

    const figures = await screen.findAllByRole("figure");
    expect(figures.map((figure) => figure.getAttribute("aria-label"))).toEqual([
      "Billed by customer",
      "Open by customer",
    ]);
  });

  it("says a chart with no values has nothing to draw", async () => {
    render(
      <AnalysisResult
        analysis={analysis({
          figures: [],
          charts: [{ ...chart, series: [{ name: "Open", values: [null, null] }] }],
        })}
      />,
    );

    expect(await screen.findByText("This chart has no values to draw.")).toBeInTheDocument();
  });
});
