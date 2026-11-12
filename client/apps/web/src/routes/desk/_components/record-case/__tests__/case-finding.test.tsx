import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render as renderBare, screen } from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CaseExhibits } from "../case-exhibits";
import { CaseFinding } from "../case-finding";
import { CaseOptions } from "../case-options";
import { caseOption, deskCase } from "./case-fixture";

// The organization here has no Google Maps key, so the route is the sketch:
// the map is drawn only where a key and the stops' places both exist.
vi.mock("@/lib/queries/integration", () => ({
  integration: {
    runtimeConfig: (type: string) => ({
      queryKey: ["integration", "runtimeConfig", type],
      queryFn: async () => ({ config: {} }),
    }),
  },
}));

function render(ui: ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );

  return renderBare(ui, { wrapper });
}

afterEach(cleanup);

describe("what was found about a record", () => {
  it("draws the reading with the deciding fact marked and each claim numbered", () => {
    const onLight = vi.fn();
    const { container } = render(
      <CaseFinding data={deskCase()} timezone="UTC" lit={null} onLight={onLight} />,
    );

    expect(screen.getByText("Dispatch desk")).toBeTruthy();
    expect(screen.getByText(/found this at .* from 2 sources/)).toBeTruthy();
    expect(container.querySelector(".dk-cz-hl")?.textContent).toBe("2h 10m of driving left");
    expect(screen.getByRole("button", { name: "Evidence 2" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Evidence 1" })).toBeTruthy();

    fireEvent.mouseEnter(screen.getByRole("button", { name: "Evidence 2" }));
    expect(onLight).toHaveBeenLastCalledWith(2);
    fireEvent.mouseLeave(screen.getByRole("button", { name: "Evidence 2" }));
    expect(onLight).toHaveBeenLastCalledWith(null);
  });

  it("names the four things that make it safe to act on, the agent's by its name", () => {
    render(<CaseFinding data={deskCase()} timezone="UTC" lit={null} onLight={vi.fn()} />);

    expect(screen.getByText("Why it matters")).toBeTruthy();
    expect(screen.getByText("If nothing changes")).toBeTruthy();
    expect(screen.getByText("Dispatch desk proposes")).toBeTruthy();
    expect(screen.getByText("Needs your approval because")).toBeTruthy();
  });

  it("holds the reading's place while Nova writes it", () => {
    const base = deskCase();
    render(
      <CaseFinding
        data={deskCase({
          finding: { ...base.finding!, readingStatus: "Writing", reading: [] },
        })}
        timezone="UTC"
        lit={null}
        onLight={vi.fn()}
      />,
    );

    expect(screen.getByLabelText("Nova is reading the case")).toBeTruthy();
    expect(screen.queryByText(/Nova's reading is unavailable/)).toBeNull();
  });

  it("says what the Watchtower raised when there is no reading, and that the plan still stands", () => {
    const base = deskCase();
    render(
      <CaseFinding
        data={deskCase({
          finding: { ...base.finding!, readingStatus: "Unavailable", reading: [] },
        })}
        timezone="UTC"
        lit={null}
        onLight={vi.fn()}
      />,
    );

    expect(screen.getByText("Watchtower raised this at 6:31 AM from eta at risk.")).toBeTruthy();
    expect(screen.getByText(/the plan below can still be decided/)).toBeTruthy();
    expect(screen.queryByRole("button", { name: /Evidence/ })).toBeNull();
  });

  it("says nothing is open when the record has no finding", () => {
    render(
      <CaseFinding
        data={deskCase({ finding: null, decision: null })}
        timezone="UTC"
        lit={null}
        onLight={vi.fn()}
      />,
    );

    expect(screen.getByText(/Nothing is open about this record/)).toBeTruthy();
  });
});

describe("what the finding rests on", () => {
  const partlyCited = () => {
    const base = deskCase();
    const finding = base.finding!;

    return deskCase({
      finding: {
        ...finding,
        sources: base.exhibits.length,
        reading: [
          { text: "John Smith is 215 miles out", highlight: false, cite: null },
          { text: "", highlight: false, cite: 1 },
          { text: ".", highlight: false, cite: null },
        ],
      },
    });
  };

  it("says how much of the evidence the reading cites when it does not cite it all", () => {
    const data = partlyCited();
    render(<CaseFinding data={data} timezone="UTC" lit={null} onLight={vi.fn()} />);

    expect(
      screen.getByText(new RegExp(`citing 1 of ${data.exhibits.length} sources`)),
    ).toBeTruthy();
  });

  it("marks the evidence the reading did not draw on, and only that", () => {
    const data = partlyCited();
    const { container } = render(<CaseExhibits data={data} lit={null} now={0} timezone="UTC" />);

    const uncited = [...container.querySelectorAll(".dk-cz-exh.dk-uncited")].map((el) => el.id);
    expect(uncited).toEqual(
      data.exhibits.filter((exhibit) => exhibit.number !== 1).map((e) => `dk-cz-exh-${e.number}`),
    );
    expect(screen.getAllByText(/not cited in the finding/)).toHaveLength(uncited.length);
  });

  it("marks nothing as left out while there is no reading yet", () => {
    const base = deskCase();
    const data = deskCase({
      finding: { ...base.finding!, readingStatus: "Writing", reading: [] },
    });
    const { container } = render(<CaseExhibits data={data} lit={null} now={0} timezone="UTC" />);

    expect(container.querySelector(".dk-uncited")).toBeNull();
  });
});

describe("what could happen", () => {
  const options = [
    caseOption(),
    caseOption({
      id: "rco_2",
      label: "Relay at Bloomington",
      recommended: false,
      chosen: false,
      outcome: "5:10 AM",
      outcomeNote: "on time",
    }),
    caseOption({
      id: "rco_3",
      kind: "Leave",
      label: "Leave it as it is",
      recommended: false,
      chosen: false,
      tone: "Bad",
      outcome: "1:18 PM",
      outcomeNote: "6h 33m late",
    }),
  ];

  it("offers the options as one choice, the chosen one checked and one recommended", () => {
    render(<CaseOptions options={options} disabled={false} pendingId={null} onChoose={vi.fn()} />);

    const radios = screen.getAllByRole("radio");
    expect(radios).toHaveLength(3);
    expect(radios.map((radio) => radio.getAttribute("aria-checked"))).toEqual([
      "true",
      "false",
      "false",
    ]);
    expect(screen.getAllByText("Recommended")).toHaveLength(1);
    expect(screen.getByText("Press 1–3")).toBeTruthy();
  });

  it("hands back the option a person picks", () => {
    const onChoose = vi.fn();
    render(<CaseOptions options={options} disabled={false} pendingId={null} onChoose={onChoose} />);

    fireEvent.click(screen.getByRole("radio", { name: /Relay at Bloomington/ }));

    expect(onChoose).toHaveBeenCalledWith(options[1]);
  });

  it("closes the choice once the plan is decided, and while one is being chosen", () => {
    const { rerender } = render(
      <CaseOptions options={options} disabled pendingId={null} onChoose={vi.fn()} />,
    );
    expect(
      screen.getAllByRole("radio").every((radio) => (radio as HTMLButtonElement).disabled),
    ).toBe(true);
    expect(screen.queryByText("Press 1–3")).toBeNull();

    rerender(
      <CaseOptions options={options} disabled={false} pendingId="rco_2" onChoose={vi.fn()} />,
    );
    expect(screen.getAllByRole("radio").map((radio) => radio.getAttribute("aria-checked"))).toEqual(
      ["false", "true", "false"],
    );
  });

  it("draws nothing when no options were offered", () => {
    const { container } = render(
      <CaseOptions options={[]} disabled={false} pendingId={null} onChoose={vi.fn()} />,
    );

    expect(container.firstChild).toBeNull();
  });
});

describe("the evidence beside a case", () => {
  it("numbers each exhibit, lights the one pointed at and says when its source was read", () => {
    const { container } = render(
      <CaseExhibits data={deskCase()} lit={2} now={1_800_000_600} timezone="UTC" />,
    );

    const exhibits = container.querySelectorAll(".dk-cz-exh");
    expect(exhibits).toHaveLength(2);
    expect(exhibits[0]?.classList.contains("dk-lit")).toBe(false);
    expect(exhibits[1]?.classList.contains("dk-lit")).toBe(true);
    expect(screen.getByText("Where it is")).toBeTruthy();
    expect(screen.getByText("John Smith's hours")).toBeTruthy();
    expect(screen.getAllByText(/^read /)).toHaveLength(2);
  });

  it("draws the route with its stops, the truck and the break that is due", () => {
    render(<CaseExhibits data={deskCase()} lit={null} now={1_800_000_600} timezone="UTC" />);

    expect(screen.getByRole("img", { name: "Route from Dallas to Chicago" })).toBeTruthy();
    expect(screen.getByText("break due ~8:50 AM")).toBeTruthy();
    expect(screen.getByText("Springfield, IL · moving 61 mph · 215 miles to Chicago")).toBeTruthy();
  });

  it("draws hours as what is left of each limit", () => {
    const { container } = render(
      <CaseExhibits data={deskCase()} lit={null} now={1_800_000_600} timezone="UTC" />,
    );

    expect(screen.getByText("2h 10m of 11h")).toBeTruthy();
    const bars = container.querySelectorAll<HTMLElement>(".dk-cz-meter-tr i");
    expect(bars[0]?.style.width).toBe("20%");
    expect(bars[0]?.classList.contains("dk-warn")).toBe(true);
    expect(bars[1]?.style.width).toBe("29%");
  });

  it("puts the Watchtower's own line at the foot", () => {
    render(<CaseExhibits data={deskCase()} lit={null} now={1_800_000_600} timezone="UTC" />);

    expect(
      screen.getByText("Watchtower: critical, raised 6:31 AM by eta at risk monitoring."),
    ).toBeTruthy();
  });
});
