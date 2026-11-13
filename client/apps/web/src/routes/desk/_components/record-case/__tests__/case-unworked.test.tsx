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

vi.mock("../case-play-dialog", () => ({
  CasePlayDialog: ({ itemId, play }: { itemId: string; play: string }) => (
    <div role="dialog" aria-label={`play ${play} for ${itemId}`} />
  ),
}));

/** A condition with no first move of its own: a person decides, or puts an agent on it. */
const ITEM = {
  itemId: "wt_1",
  kind: "BillingException" as const,
  working: false,
  agent: null,
  note: "",
  play: null,
  request: null,
};

/** A missing proof of delivery, whose first move is to ask for it. */
const POD = { ...ITEM, itemId: "wt_2", kind: "PodMissing" as const, play: "AskForPod" as const };

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

  it("starts no agent by opening: the hand-off waits for the person to ask", async () => {
    render(<CaseUnworked handOff={ITEM} canHandOff onHanded={vi.fn()} onAcceptRisk={null} />);

    await Promise.resolve();
    expect(handOffWatchtowerItem).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Hand it to an agent" })).toBeTruthy();
  });

  it("offers the condition's first move first, for the person to read and send themselves", async () => {
    const onHanded = vi.fn();
    render(<CaseUnworked handOff={POD} canHandOff onHanded={onHanded} onAcceptRisk={null} />);

    expect(screen.queryByRole("dialog")).toBeNull();
    await userEvent.click(screen.getByRole("button", { name: "Ask for the POD" }));

    expect(screen.getByRole("dialog", { name: "play AskForPod for wt_2" })).toBeTruthy();
    expect(handOffWatchtowerItem).not.toHaveBeenCalled();
    expect(onHanded).not.toHaveBeenCalled();
    // An agent is still theirs to ask for, as the second thing.
    expect(screen.getByRole("button", { name: "Hand it to an agent" })).toBeTruthy();
  });

  it("offers no first move for a condition that has none", () => {
    render(<CaseUnworked handOff={ITEM} canHandOff onHanded={vi.fn()} onAcceptRisk={null} />);

    expect(screen.queryByRole("button", { name: "Ask for the POD" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Ask where it is" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Tender it again" })).toBeNull();
  });

  it.each([
    ["AskForStatus" as const, "Ask where it is"],
    ["TenderAgain" as const, "Tender it again"],
  ])("names the %s move as a person would", (play, label) => {
    render(
      <CaseUnworked
        handOff={{ ...POD, kind: "MissedDelivery", play }}
        canHandOff
        onHanded={vi.fn()}
        onAcceptRisk={null}
      />,
    );

    expect(screen.getByRole("button", { name: label })).toBeTruthy();
  });

  it("says who was asked what, and when, once the move was made, and offers to ask again", () => {
    const twoHoursAgo = Math.floor(Date.now() / 1000) - 2 * 3600;
    render(
      <CaseUnworked
        handOff={{
          ...POD,
          request: {
            at: twoHoursAgo,
            by: "Dana Reyes",
            summary: "Asked Laura Wright for the signed proof of delivery in Dash",
          },
        }}
        canHandOff
        onHanded={vi.fn()}
        onAcceptRisk={vi.fn()}
      />,
    );

    expect(
      screen.getByRole("heading", {
        name: "Asked Laura Wright for the signed proof of delivery in Dash",
      }),
    ).toBeTruthy();
    expect(screen.getByText("Dana Reyes, 2 hours ago")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Ask again" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Ask for the POD" })).toBeNull();
    expect(screen.queryByText("No agent has worked this yet.")).toBeNull();
  });

  it("offers a reader who may not work the case no first move", () => {
    render(
      <CaseUnworked
        handOff={POD}
        canHandOff={false}
        locked="Dana Reyes owns this case, so only they can act on it."
        onHanded={vi.fn()}
        onAcceptRisk={null}
      />,
    );

    expect(screen.queryByRole("button", { name: "Ask for the POD" })).toBeNull();
    expect(screen.getByText("Dana Reyes owns this case, so only they can act on it.")).toBeTruthy();
  });

  it("offers no agent for an item an agent raised to ask for a person", () => {
    render(
      <CaseUnworked
        handOff={{ ...ITEM, note: "I could not reach the driver." }}
        canHandOff
        agentRaised
        onHanded={vi.fn()}
        onAcceptRisk={vi.fn()}
      />,
    );

    expect(screen.queryByRole("button", { name: "Hand it to another agent" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Hand it to an agent" })).toBeNull();
    expect(screen.getByRole("button", { name: "Accept the risk…" })).toBeTruthy();
    expect(
      screen.getByText(
        "An agent raised this because it needs you. Do it yourself, ask about it below, or accept the risk.",
      ),
    ).toBeTruthy();
  });
});
