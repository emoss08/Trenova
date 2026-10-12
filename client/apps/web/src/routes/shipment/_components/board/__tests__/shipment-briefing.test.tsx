import { act, cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ShipmentBriefing } from "../briefing/shipment-briefing";

const { setUrl, openRecord, authState, briefing, loads } = vi.hoisted(() => ({
  setUrl: vi.fn(),
  openRecord: vi.fn(),
  authState: { user: { name: "Marcus Rivera" } as { name?: string } | null },
  briefing: {
    current: {
      segments: [
        { text: "3 loads are late", filter: "Late" },
        { text: ". 2 loads still need a driver.", filter: null },
      ],
      narrated: false,
      generatedAt: 1_791_000_000,
      generation: 1,
    },
  },
  loads: {
    current: {
      results: [
        {
          id: "shp_1",
          proNumber: "PRO-101",
          customer: { name: "Acme Foods" },
          stage: "Late",
          moves: [
            {
              stops: [
                { type: "Pickup", location: { city: "Dallas" } },
                { type: "Delivery", location: { city: "Tulsa" } },
              ],
            },
          ],
        },
        {
          id: "shp_2",
          proNumber: "PRO-102",
          customer: { name: "Blue Freight" },
          stage: "Late",
          moves: [
            {
              stops: [
                { type: "Pickup", location: { city: "Austin" } },
                { type: "Delivery", location: { city: "Houston" } },
              ],
            },
          ],
        },
      ],
      count: 3,
    },
  },
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: ({ queryKey }: { queryKey: unknown[] }) =>
    queryKey[0] === "briefing-loads"
      ? { data: loads.current, isLoading: false, isError: false }
      : { data: briefing.current, isLoading: false, isError: false },
}));

vi.mock("@/lib/queries", () => ({
  queries: {
    shipmentBoard: {
      briefing: () => ({ queryKey: ["briefing"] }),
      briefingLoads: (filter: string) => ({ queryKey: ["briefing-loads", filter] }),
    },
  },
}));

vi.mock("motion/react", () => ({ useReducedMotion: () => true }));
vi.mock("../url-state", () => ({ useShipmentBoardUrl: () => [{}, setUrl] }));
vi.mock("../use-open-shipment-record", () => ({ useOpenShipmentRecord: () => openRecord }));
vi.mock("@/hooks/use-user-timezone", () => ({ useUserTimezone: () => "UTC" }));
vi.mock("@trenova/shared/stores/auth-store", () => ({
  useAuthStore: (select: (state: typeof authState) => unknown) => select(authState),
}));
vi.mock("@/lib/shipment-board/capabilities", () => ({
  useShipmentCapabilities: () => ({ ai: true, operationType: "asset" }),
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  authState.user = { name: "Marcus Rivera" };
});

describe("ShipmentBriefing", () => {
  it("greets the dispatcher by first name ahead of the day's brief", () => {
    render(<ShipmentBriefing />);

    const sentence = screen.getByText(/Hello Marcus\./).closest("p");
    expect(sentence?.textContent).toBe(
      "Hello Marcus. 3 loads are late. 2 loads still need a driver.",
    );
    expect(screen.getByText(/^Brief as of /)).toBeTruthy();
  });

  it("greets without a name when the session does not know one", () => {
    authState.user = null;
    render(<ShipmentBriefing />);

    expect(screen.getByText(/Hello\./).closest("p")?.textContent).toMatch(/^Hello\. 3 loads/);
  });

  it("lists the loads a phrase names when it is hovered", async () => {
    render(<ShipmentBriefing />);

    const phrase = screen.getByRole("button", { name: "Show Late" });
    await act(async () => {
      fireEvent.mouseEnter(phrase);
      fireEvent.pointerEnter(phrase);
      await new Promise((resolve) => setTimeout(resolve, 400));
    });

    const card = await screen.findByText("PRO-101");
    const preview = card.closest("ul")!;
    expect(within(preview).getByText("PRO-102")).toBeTruthy();
    expect(within(preview).getByText("Acme Foods")).toBeTruthy();
    expect(within(preview).getByText("Dallas")).toBeTruthy();

    fireEvent.click(within(preview).getByText("PRO-102"));
    expect(openRecord).toHaveBeenCalledWith("shp_2");

    fireEvent.click(screen.getByRole("button", { name: "Show all 3 on the board" }));
    expect(setUrl).toHaveBeenCalledWith({ qf: [{ filter: "Late" }], view: "table", expanded: null });
  });

  it("filters the board when a phrase is clicked", () => {
    render(<ShipmentBriefing />);

    fireEvent.click(screen.getByRole("button", { name: "Show Late" }));
    expect(setUrl).toHaveBeenCalledWith({ qf: [{ filter: "Late" }], view: "table", expanded: null });
  });
});
