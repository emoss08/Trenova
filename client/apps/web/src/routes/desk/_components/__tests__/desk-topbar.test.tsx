import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";
import { DeskTopBar } from "../desk-topbar";

const agent = { id: "agd_1", name: "Shipment Assistant" } as AgentChoice;

function thread(overrides: Partial<AssistantThread> = {}): AssistantThread {
  return {
    id: "athr_1",
    businessUnitId: "bu",
    organizationId: "org",
    userId: "u",
    agentDefinitionId: agent.id,
    preferredProviderId: "",
    origin: "Desk",
    pinned: false,
    subjectType: "",
    subjectId: "",
    canContinue: true,
    title: "Late loads",
    status: "Active",
    lastMessageAt: 0,
    version: 0,
    createdAt: 0,
    updatedAt: 0,
    ...overrides,
  };
}

function renderBar(overrides: Partial<AssistantThread> = {}) {
  const handlers = {
    onOpenAgent: vi.fn(),
    onToggleWorkspace: vi.fn(),
    onTogglePin: vi.fn(),
    onDownload: vi.fn(),
  };
  render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter>
        <DeskTopBar
          place="thread"
          thread={thread(overrides)}
          agent={agent}
          agents={[agent]}
          workspaceOpen={false}
          artifactCount={0}
          newArtifact={false}
          pending={false}
          {...handlers}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );

  return handlers;
}

/**
 * The bar keeps in view what a conversation is worked with: its case, the
 * hand-off and the workspace. Pinning, the transcript and what the agent can
 * do are a menu away, so the corner stays four controls wide.
 */
describe("DeskTopBar", () => {
  it("keeps the case, hand-off, menu and workspace in view and nothing else", () => {
    renderBar();

    expect(screen.getByRole("button", { name: "Make this a case" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Hand off to another agent" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "More actions" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Workspace" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Download transcript" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Pin conversation" })).toBeNull();
  });

  it("pins, downloads and opens the agent from the menu", async () => {
    const handlers = renderBar({ pinned: true });

    fireEvent.click(screen.getByRole("button", { name: "More actions" }));
    fireEvent.click(await screen.findByRole("menuitem", { name: "Unpin conversation" }));
    expect(handlers.onTogglePin).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByRole("button", { name: "More actions" }));
    fireEvent.click(await screen.findByRole("menuitem", { name: "Download transcript" }));
    expect(handlers.onDownload).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByRole("button", { name: "More actions" }));
    fireEvent.click(await screen.findByRole("menuitem", { name: "What this agent can do" }));
    expect(handlers.onOpenAgent).toHaveBeenCalledTimes(1);
  });

  it("names its controls with tooltips rather than browser titles", () => {
    renderBar();

    for (const name of ["Make this a case", "Hand off to another agent", "More actions"]) {
      expect(screen.getByRole("button", { name })).not.toHaveAttribute("title");
    }
  });
});
