import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { AssistantThread } from "@/types/assistant";
import { AssistantFullHeader, AssistantHeader, type AssistantView } from "../assistant-header";

afterEach(cleanup);

const thread: AssistantThread = {
  id: "athr_01M3034Q2N7JD99RA1D8DGH1ZF",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  userId: "usr_1",
  agentDefinitionId: "agd_1",
  title: "Run the revenue report",
  status: "Active",
  lastMessageAt: 1,
  preferredProviderId: "",
  origin: "Desk",
  pinned: false,
  subjectType: "",
  subjectId: "",
  canContinue: true,
  version: 1,
  createdAt: 1,
  updatedAt: 1,
};

function renderHeader(view: AssistantView, layout: "compact" | "side" = "compact") {
  const handlers = {
    onDelete: vi.fn(),
    onDownloadTranscript: vi.fn(),
    onOpenInDesk: vi.fn(),
    onBack: vi.fn(),
    onToggleHistory: vi.fn(),
    onNew: vi.fn(),
    onLayout: vi.fn(),
    onClose: vi.fn(),
  };
  render(
    <AssistantHeader
      layout={layout}
      view={view}
      thread={view === "thread" ? thread : null}
      actions={{
        onDelete: handlers.onDelete,
        onDownloadTranscript: handlers.onDownloadTranscript,
        onOpenInDesk: handlers.onOpenInDesk,
      }}
      onBack={handlers.onBack}
      onToggleHistory={handlers.onToggleHistory}
      onNew={handlers.onNew}
      onLayout={handlers.onLayout}
      onClose={handlers.onClose}
    />,
  );

  return handlers;
}

async function openThreadMenu() {
  fireEvent.click(screen.getByRole("button", { name: "Conversation actions" }));
  return screen.findByRole("menu");
}

/**
 * The header says where the panel is (the assistant, the conversations, or
 * the open conversation by its title) and keeps the panel's controls in one
 * row: the conversations, a new one, docking, full screen and close.
 */
describe("AssistantHeader", () => {
  it("titles the home 'Assistant', with no way back and no conversation actions", () => {
    renderHeader("home");

    expect(screen.getByText("Assistant")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Back" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Conversation actions" })).toBeNull();
  });

  it("titles an open conversation by its title and goes back from it", () => {
    const { onBack } = renderHeader("thread");

    expect(screen.getByText("Run the revenue report")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back" }));
    expect(onBack).toHaveBeenCalledTimes(1);
  });

  it("marks the conversations button pressed while the list is showing", () => {
    const { onToggleHistory } = renderHeader("history");

    const button = screen.getByRole("button", { name: "Conversations" });
    expect(button).toHaveAttribute("aria-pressed", "true");
    fireEvent.click(button);
    expect(onToggleHistory).toHaveBeenCalledTimes(1);
  });

  it("docks to the side from the corner and floats again from the side", () => {
    const floating = renderHeader("home", "compact");
    fireEvent.click(screen.getByRole("button", { name: "Dock to side" }));
    expect(floating.onLayout).toHaveBeenCalledWith("side");
    cleanup();

    const docked = renderHeader("home", "side");
    fireEvent.click(screen.getByRole("button", { name: "Float" }));
    expect(docked.onLayout).toHaveBeenCalledWith("compact");
  });

  it("fills the screen and closes", () => {
    const { onLayout, onClose } = renderHeader("home");

    fireEvent.click(screen.getByRole("button", { name: "Full screen" }));
    fireEvent.click(screen.getByRole("button", { name: "Close · Esc" }));

    expect(onLayout).toHaveBeenCalledWith("full");
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});

/**
 * A conversation worth keeping is worth taking out of the panel, and one that
 * outgrew it belongs at the Desk: both sit behind ⋯ beside the title, with
 * deleting it.
 */
describe("AssistantHeader conversation actions", () => {
  it("downloads the open conversation", async () => {
    const { onDownloadTranscript } = renderHeader("thread");

    await openThreadMenu();
    fireEvent.click(await screen.findByRole("menuitem", { name: "Download transcript" }));

    expect(onDownloadTranscript).toHaveBeenCalledWith(thread);
  });

  it("opens the conversation in the Desk", async () => {
    const { onOpenInDesk } = renderHeader("thread");

    await openThreadMenu();
    fireEvent.click(await screen.findByRole("menuitem", { name: "Open in Desk" }));

    expect(onOpenInDesk).toHaveBeenCalledWith(thread);
  });

  it("asks to delete the conversation", async () => {
    const { onDelete } = renderHeader("thread");

    await openThreadMenu();
    fireEvent.click(await screen.findByRole("menuitem", { name: "Delete conversation" }));

    expect(onDelete).toHaveBeenCalledWith(thread);
  });
});

describe("AssistantFullHeader", () => {
  function renderFull(sidebarOpen: boolean) {
    const handlers = {
      onToggleSidebar: vi.fn(),
      onNew: vi.fn(),
      onShrink: vi.fn(),
      onClose: vi.fn(),
    };
    render(
      <AssistantFullHeader
        sidebarOpen={sidebarOpen}
        thread={thread}
        actions={{ onDelete: vi.fn(), onDownloadTranscript: vi.fn(), onOpenInDesk: vi.fn() }}
        {...handlers}
      />,
    );
    return handlers;
  }

  it("folds the conversations away and shrinks back to the corner", () => {
    const { onToggleSidebar, onShrink } = renderFull(true);

    fireEvent.click(screen.getByRole("button", { name: "Hide conversations · ⌘\\" }));
    fireEvent.click(screen.getByRole("button", { name: "Shrink" }));

    expect(onToggleSidebar).toHaveBeenCalledTimes(1);
    expect(onShrink).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("button", { name: "New conversation" })).toBeNull();
  });

  it("offers a new conversation in the bar while the sidebar is folded away", () => {
    const { onNew } = renderFull(false);

    fireEvent.click(screen.getByRole("button", { name: "New conversation" }));

    expect(onNew).toHaveBeenCalledTimes(1);
    expect(screen.getByText("Run the revenue report")).toBeInTheDocument();
  });
});
