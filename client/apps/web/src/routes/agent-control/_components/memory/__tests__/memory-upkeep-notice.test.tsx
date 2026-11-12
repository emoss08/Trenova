import type { AgentMemoryUpkeep } from "@/lib/graphql/agent-memories";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryUpkeepNotice, upkeepReasonLabel } from "../memory-upkeep-notice";

const api = vi.hoisted(() => ({
  fetchAgentMemoryUpkeep: vi.fn(),
  confirmAgentMemory: vi.fn(),
  setAgentMemoryStatus: vi.fn(),
}));

vi.mock("@/lib/graphql/agent-memories", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-memories")>()),
  ...api,
}));

const NOW = 1_800_000_000;
const DAY = 86_400;

function upkeep(
  reason: AgentMemoryUpkeep["reason"],
  memory: Partial<AgentMemoryUpkeep["memory"]> = {},
): AgentMemoryUpkeep {
  return {
    reason,
    memory: {
      id: "amem_1",
      kind: "Fact",
      status: "Active",
      scope: "Organization",
      subjectLabel: "",
      content: "The receiving dock closes at 4pm on Fridays.",
      useCount: 12,
      confirmedAt: null,
      expiresAt: null,
      version: 3,
      createdAt: NOW - 200 * DAY,
      ...memory,
    } as AgentMemoryUpkeep["memory"],
  };
}

const t = ((text: string, ...args: unknown[]) =>
  text.replace(/\{(\d+)\}/g, (_, index: string) => String(args[Number(index)]))) as TranslateFn;

function renderNotice(canDecide = true) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryUpkeepNotice canDecide={canDecide} />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  api.confirmAgentMemory.mockResolvedValue({});
  api.setAgentMemoryStatus.mockResolvedValue({});
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("upkeepReasonLabel", () => {
  it("says an aging fact was never confirmed, from when it was recorded", () => {
    expect(upkeepReasonLabel(upkeep("Aging"), NOW, t)).toBe(
      "A fact with no end date, recorded 7 months ago and never confirmed",
    );
  });

  it("counts an aging fact from its last confirmation, not from when it was recorded", () => {
    expect(upkeepReasonLabel(upkeep("Aging", { confirmedAt: NOW - 190 * DAY }), NOW, t)).toBe(
      "A fact with no end date, last confirmed 6 months ago",
    );
  });

  it("says no agent has used a memory since it was recorded", () => {
    expect(
      upkeepReasonLabel(
        upkeep("NeverUsed", { kind: "Instruction", useCount: 0, createdAt: NOW - 40 * DAY }),
        NOW,
        t,
      ),
    ).toBe("No agent has used it since it was recorded last month");
  });
});

describe("MemoryUpkeepNotice", () => {
  it("renders nothing when no memory is worth another look", async () => {
    api.fetchAgentMemoryUpkeep.mockResolvedValue([]);
    renderNotice();

    await waitFor(() => expect(api.fetchAgentMemoryUpkeep).toHaveBeenCalled());
    expect(screen.queryByTestId("memory-upkeep-notice")).not.toBeInTheDocument();
  });

  it("lists each memory under the record it is about and counts them", async () => {
    api.fetchAgentMemoryUpkeep.mockResolvedValue([
      upkeep("Aging", { subjectLabel: "Acme Manufacturing" }),
      upkeep("NeverUsed", {
        id: "amem_2",
        kind: "Instruction",
        content: "CC the AP inbox on every invoice.",
        useCount: 0,
      }),
    ]);
    renderNotice();

    expect(await screen.findByText("2 memories are worth another look")).toBeInTheDocument();
    const rows = within(screen.getByRole("list")).getAllByRole("listitem");
    expect(rows).toHaveLength(2);
    expect(
      within(rows[0]).getByText("Acme Manufacturing: The receiving dock closes at 4pm on Fridays."),
    ).toBeInTheDocument();
    expect(within(rows[1]).getByText("CC the AP inbox on every invoice.")).toBeInTheDocument();
  });

  it("confirms a memory at the version read and refreshes the list", async () => {
    api.fetchAgentMemoryUpkeep.mockResolvedValueOnce([upkeep("Aging")]).mockResolvedValue([]);
    renderNotice();

    await userEvent.click(await screen.findByRole("button", { name: "Still true" }));

    expect(api.confirmAgentMemory).toHaveBeenCalledWith("amem_1", 3);
    expect(api.setAgentMemoryStatus).not.toHaveBeenCalled();
    await waitFor(() =>
      expect(screen.queryByTestId("memory-upkeep-notice")).not.toBeInTheDocument(),
    );
  });

  it("retires a memory that no longer holds", async () => {
    api.fetchAgentMemoryUpkeep.mockResolvedValueOnce([upkeep("Aging")]).mockResolvedValue([]);
    renderNotice();

    await userEvent.click(await screen.findByRole("button", { name: "Retire" }));

    expect(api.setAgentMemoryStatus).toHaveBeenCalledWith("amem_1", "Retired");
    expect(api.confirmAgentMemory).not.toHaveBeenCalled();
  });

  it("offers no decision to someone who may not manage memory", async () => {
    api.fetchAgentMemoryUpkeep.mockResolvedValue([upkeep("Aging")]);
    renderNotice(false);

    expect(await screen.findByText("1 memory is worth another look")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Still true" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Retire" })).not.toBeInTheDocument();
  });

  it("says why a decision did not go through and leaves the memory listed", async () => {
    api.fetchAgentMemoryUpkeep.mockResolvedValue([upkeep("Aging")]);
    api.confirmAgentMemory.mockRejectedValue(new Error("Someone changed this memory"));
    renderNotice();

    await userEvent.click(await screen.findByRole("button", { name: "Still true" }));

    expect(await screen.findByText("Someone changed this memory")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Still true" })).toBeEnabled();
  });
});
