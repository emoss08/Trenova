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
    onRename: vi.fn(),
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

/**
 * The conversation is renamed where its name is shown: a click on the name
 * in the bar turns it into a field.
 */
describe("DeskTopBar rename", () => {
  const field = () => screen.getByRole("textbox", { name: "Conversation name" });

  it("renames from the name in the bar: click, type, Enter", () => {
    const handlers = renderBar();

    fireEvent.click(screen.getByRole("button", { name: "Late loads" }));
    expect(field()).toHaveValue("Late loads");
    fireEvent.change(field(), { target: { value: "  Detention claims  " } });
    fireEvent.keyDown(field(), { key: "Enter" });

    expect(handlers.onRename).toHaveBeenCalledTimes(1);
    expect(handlers.onRename).toHaveBeenCalledWith("Detention claims");
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
  });

  // Enter ends the edit, and the field losing focus afterwards must not send
  // the same name a second time.
  it("sends a name once when Enter is followed by the field losing focus", () => {
    const handlers = renderBar();

    fireEvent.click(screen.getByRole("button", { name: "Late loads" }));
    const input = field();
    fireEvent.change(input, { target: { value: "Detention claims" } });
    fireEvent.keyDown(input, { key: "Enter" });
    fireEvent.blur(input);

    expect(handlers.onRename).toHaveBeenCalledTimes(1);
  });

  it("keeps the new name when the field loses focus", () => {
    const handlers = renderBar();

    fireEvent.click(screen.getByRole("button", { name: "Late loads" }));
    fireEvent.change(field(), { target: { value: "Detention claims" } });
    fireEvent.blur(field());

    expect(handlers.onRename).toHaveBeenCalledWith("Detention claims");
  });

  it("leaves the name alone on Escape", () => {
    const handlers = renderBar();

    fireEvent.click(screen.getByRole("button", { name: "Late loads" }));
    fireEvent.change(field(), { target: { value: "Something else" } });
    fireEvent.keyDown(field(), { key: "Escape" });

    expect(handlers.onRename).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Late loads" })).toBeInTheDocument();
  });

  it("sends nothing for a blank or unchanged name", () => {
    const handlers = renderBar();

    fireEvent.click(screen.getByRole("button", { name: "Late loads" }));
    fireEvent.change(field(), { target: { value: "   " } });
    fireEvent.keyDown(field(), { key: "Enter" });
    fireEvent.click(screen.getByRole("button", { name: "Late loads" }));
    fireEvent.keyDown(field(), { key: "Enter" });

    expect(handlers.onRename).not.toHaveBeenCalled();
  });

  it("can be renamed again after a rename", () => {
    const handlers = renderBar();

    fireEvent.click(screen.getByRole("button", { name: "Late loads" }));
    fireEvent.change(field(), { target: { value: "First" } });
    fireEvent.keyDown(field(), { key: "Enter" });
    fireEvent.click(screen.getByRole("button", { name: "Late loads" }));
    fireEvent.change(field(), { target: { value: "Second" } });
    fireEvent.keyDown(field(), { key: "Enter" });

    expect(handlers.onRename).toHaveBeenNthCalledWith(1, "First");
    expect(handlers.onRename).toHaveBeenNthCalledWith(2, "Second");
  });

  it("names an untitled conversation and still offers to rename it", () => {
    renderBar({ title: "" });

    fireEvent.click(screen.getByRole("button", { name: "Untitled conversation" }));
    expect(field()).toHaveValue("");
  });

  it("caps a name at the length the server keeps", () => {
    renderBar();

    fireEvent.click(screen.getByRole("button", { name: "Late loads" }));
    expect(field()).toHaveAttribute("maxLength", "200");
  });
});
