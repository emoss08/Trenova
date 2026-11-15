import { renderAtRoutes } from "@trenova/shared/testing/test-router";
import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DeskConversationPage } from "../conversation-page";

const desk = vi.hoisted(() => ({
  isLoading: false,
  activeThread: null,
  deletingActive: false,
  openSearch: vi.fn(),
  openArtifact: vi.fn(),
  agentsById: new Map(),
}));

vi.mock("../_components/desk-layout", () => ({ useDesk: () => desk }));

vi.mock("../_components/desk-conversation", () => ({ DeskConversation: () => null }));

async function renderPage() {
  return await renderAtRoutes(
    [
      { path: "/desk/t/$threadId", component: () => <DeskConversationPage /> },
      { path: "/desk", component: () => <p>Today</p> },
    ],
    { initialEntries: ["/desk/t/thread-1"] },
  );
}

beforeEach(() => {
  desk.deletingActive = false;
  desk.openSearch.mockReset();
});

describe("Desk conversation page for a conversation that is not there", () => {
  it("says what happened and offers the way back and a search", async () => {
    await renderPage();

    expect(screen.getByRole("region", { name: "This conversation is not here" })).toHaveTextContent(
      "It may have been deleted, or it belongs to someone else.",
    );
    expect(screen.getByRole("link", { name: "Back to the desk" })).toHaveAttribute("href", "/desk");

    fireEvent.click(screen.getByRole("button", { name: "Search conversations" }));

    expect(desk.openSearch).toHaveBeenCalledTimes(1);
  });

  it("leads back to the desk", async () => {
    await renderPage();

    fireEvent.click(screen.getByRole("link", { name: "Back to the desk" }));

    expect(await screen.findByText("Today")).toBeInTheDocument();
  });

  // Deleting the open conversation empties it from the lists before the Desk
  // has moved off it; that is not a missing conversation.
  it("draws nothing while the open conversation is being deleted", async () => {
    desk.deletingActive = true;

    const { container } = await renderPage();

    expect(container).toBeEmptyDOMElement();
  });
});
