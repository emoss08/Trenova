import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CaseUnworked } from "../case-unworked";

const handOffWatchtowerItem = vi.fn();

vi.mock("@/lib/graphql/watchtower", () => ({
  handOffWatchtowerItem: (...args: unknown[]) => handOffWatchtowerItem(...args),
}));
vi.mock("@/components/elements/ai-markdown", () => ({
  AiMarkdown: ({ content }: { content: string }) => <p>{content}</p>,
}));
vi.mock("@/components/desk-chat/desk-agent-tile", () => ({
  DeskAgentTile: () => <i data-testid="tile" />,
}));

const ITEM = { itemId: "wt_1", kind: "PodMissing" as const, working: false, agent: null, note: "" };

function result(overrides: Record<string, unknown> = {}) {
  return { item: {}, runId: null, subscribers: [], candidates: [], templates: [], ...overrides };
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("a case nobody has worked", () => {
  it("hands it to the agents that take this kind of thing and says who is working it", async () => {
    handOffWatchtowerItem.mockResolvedValue(
      result({ runId: "ar_1", subscribers: [{ id: "agdef_1", name: "Billing exception agent" }] }),
    );
    const onHanded = vi.fn();
    render(<CaseUnworked handOff={ITEM} canHandOff onHanded={onHanded} onAcceptRisk={null} />);

    expect(screen.getByText("No agent has worked this yet.")).toBeTruthy();
    await userEvent.click(screen.getByRole("button", { name: "Hand it to an agent" }));

    await waitFor(() => expect(onHanded).toHaveBeenCalledTimes(1));
    expect(handOffWatchtowerItem).toHaveBeenCalledWith(
      "wt_1",
      undefined,
      expect.objectContaining({ idempotencyKey: expect.any(String) }),
    );
    expect(
      screen.getByText(
        "Billing exception agent is working this case. What it proposes shows here for you to approve.",
      ),
    ).toBeTruthy();
  });

  it("shows what the agent said, not a spinner, once the case says it finished", async () => {
    handOffWatchtowerItem.mockResolvedValue(
      result({ runId: "ar_1", subscribers: [{ id: "agdef_1", name: "Detention desk" }] }),
    );
    const props = { canHandOff: true, onHanded: vi.fn(), onAcceptRisk: null };
    const { rerender } = render(<CaseUnworked {...props} handOff={ITEM} />);

    await userEvent.click(screen.getByRole("button", { name: "Hand it to an agent" }));
    await screen.findByText(/Detention desk is working this case/);

    rerender(
      <CaseUnworked
        {...props}
        handOff={{
          ...ITEM,
          agent: { id: "agdef_1", name: "Detention desk" } as never,
          note: "The notice already went out.",
        }}
      />,
    );

    expect(screen.queryByText(/is working this case/)).toBeNull();
    expect(screen.getByText("Detention desk looked at this and proposed nothing.")).toBeTruthy();
  });

  it("asks who works it when no agent takes it on its own", async () => {
    handOffWatchtowerItem
      .mockResolvedValueOnce(result({ candidates: [{ id: "agdef_2", name: "Dispatch desk" }] }))
      .mockResolvedValueOnce(result({ runId: "ar_1" }));
    const onHanded = vi.fn();
    render(<CaseUnworked handOff={ITEM} canHandOff onHanded={onHanded} onAcceptRisk={null} />);

    await userEvent.click(screen.getByRole("button", { name: "Hand it to an agent" }));
    expect(onHanded).not.toHaveBeenCalled();
    await userEvent.click(await screen.findByRole("button", { name: "Dispatch desk" }));

    await waitFor(() => expect(onHanded).toHaveBeenCalledTimes(1));
    expect(handOffWatchtowerItem).toHaveBeenLastCalledWith(
      "wt_1",
      "agdef_2",
      expect.objectContaining({ idempotencyKey: expect.any(String) }),
    );
  });

  it("says so when no agent is set up to work it", async () => {
    handOffWatchtowerItem.mockResolvedValue(result());
    render(<CaseUnworked handOff={ITEM} canHandOff onHanded={vi.fn()} onAcceptRisk={null} />);

    await userEvent.click(screen.getByRole("button", { name: "Hand it to an agent" }));

    expect(
      await screen.findByText(
        "No agent is set up to work this. Ask about it below, or turn one on in AI control.",
      ),
    ).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Hand it to an agent" })).toBeNull();
  });

  it("offers what a person can do themselves beside the hand-off", async () => {
    const upload = vi.fn();
    render(
      <CaseUnworked
        handOff={ITEM}
        canHandOff
        onHanded={vi.fn()}
        onAcceptRisk={null}
        actions={[{ label: "Upload the POD", onClick: upload }]}
      />,
    );

    await userEvent.click(screen.getByRole("button", { name: "Upload the POD" }));

    expect(upload).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("button", { name: "Hand it to an agent" })).toBeTruthy();
  });

  it("shows what an agent said when it finished and proposed nothing, and offers another", () => {
    render(
      <CaseUnworked
        handOff={{
          ...ITEM,
          agent: { id: "agdef_1", name: "Billing exception agent" } as never,
          note: "No billing item exists for this load.",
        }}
        canHandOff
        onHanded={vi.fn()}
        onAcceptRisk={null}
      />,
    );

    expect(
      screen.getByText("Billing exception agent looked at this and proposed nothing."),
    ).toBeTruthy();
    expect(screen.getByText("No billing item exists for this load.")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Hand it to another agent" })).toBeTruthy();
  });

  it("shows an agent already on it, and offers no hand-off to someone who cannot work the case", () => {
    const { rerender } = render(
      <CaseUnworked
        handOff={{ ...ITEM, working: true }}
        canHandOff
        onHanded={vi.fn()}
        onAcceptRisk={null}
      />,
    );
    expect(
      screen.getByText(
        "An agent is working this case. What it proposes shows here for you to approve.",
      ),
    ).toBeTruthy();

    const onAcceptRisk = vi.fn();
    rerender(
      <CaseUnworked
        handOff={ITEM}
        canHandOff={false}
        onHanded={vi.fn()}
        onAcceptRisk={onAcceptRisk}
      />,
    );
    expect(screen.queryByRole("button", { name: "Hand it to an agent" })).toBeNull();
    expect(screen.getByRole("button", { name: "Accept the risk…" })).toBeTruthy();
  });

  it("is handed over as it opens when the person came from the Watchtower to work it", async () => {
    handOffWatchtowerItem.mockResolvedValue(
      result({ runId: "ar_1", subscribers: [{ id: "agdef_1", name: "Billing exception agent" }] }),
    );
    const onHanded = vi.fn();
    render(
      <CaseUnworked handOff={ITEM} canHandOff autoStart onHanded={onHanded} onAcceptRisk={null} />,
    );

    expect(screen.queryByRole("button", { name: "Hand it to an agent" })).toBeNull();
    await waitFor(() => expect(onHanded).toHaveBeenCalledTimes(1));
    expect(handOffWatchtowerItem).toHaveBeenCalledTimes(1);
    expect(handOffWatchtowerItem).toHaveBeenCalledWith(
      "wt_1",
      undefined,
      expect.objectContaining({ idempotencyKey: expect.any(String) }),
    );
    expect(
      screen.getByText(
        "Billing exception agent is working this case. What it proposes shows here for you to approve.",
      ),
    ).toBeTruthy();
  });

  it("asks who works it when it opens to be worked and no agent takes it alone", async () => {
    handOffWatchtowerItem.mockResolvedValue(
      result({ candidates: [{ id: "agdef_2", name: "Dispatch desk" }] }),
    );
    render(
      <CaseUnworked handOff={ITEM} canHandOff autoStart onHanded={vi.fn()} onAcceptRisk={null} />,
    );

    expect(await screen.findByRole("button", { name: "Dispatch desk" })).toBeTruthy();
    expect(handOffWatchtowerItem).toHaveBeenCalledTimes(1);
  });

  it.each([
    ["an agent already answered", { ...ITEM, note: "Nothing to propose." }, true],
    ["an agent is on it", { ...ITEM, working: true }, true],
    ["the reader may not work it", ITEM, false],
  ])("starts nothing on opening when %s", async (_name, handOff, canHandOff) => {
    render(
      <CaseUnworked
        handOff={handOff}
        canHandOff={canHandOff}
        autoStart
        onHanded={vi.fn()}
        onAcceptRisk={null}
      />,
    );

    await Promise.resolve();
    expect(handOffWatchtowerItem).not.toHaveBeenCalled();
  });
});
