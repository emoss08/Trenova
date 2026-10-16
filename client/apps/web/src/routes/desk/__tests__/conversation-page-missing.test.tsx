import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
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

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/desk/t/thread-1"]}>
      <Routes>
        <Route path="/desk/t/:threadId" element={<DeskConversationPage />} />
        <Route path="/desk" element={<p>Today</p>} />
      </Routes>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  desk.deletingActive = false;
  desk.openSearch.mockReset();
});

describe("Desk conversation page for a conversation that is not there", () => {
  it("says what happened and offers the way back and a search", () => {
    renderPage();

    expect(
      screen.getByRole("region", { name: "This conversation is not here" }),
    ).toHaveTextContent("It may have been deleted, or it belongs to someone else.");
    expect(screen.getByRole("link", { name: "Back to the desk" })).toHaveAttribute(
      "href",
      "/desk",
    );

    fireEvent.click(screen.getByRole("button", { name: "Search conversations" }));

    expect(desk.openSearch).toHaveBeenCalledTimes(1);
  });

  it("leads back to the desk", () => {
    renderPage();

    fireEvent.click(screen.getByRole("link", { name: "Back to the desk" }));

    expect(screen.getByText("Today")).toBeInTheDocument();
  });

  // Deleting the open conversation empties it from the lists before the Desk
  // has moved off it; that is not a missing conversation.
  it("draws nothing while the open conversation is being deleted", () => {
    desk.deletingActive = true;

    const { container } = renderPage();

    expect(container).toBeEmptyDOMElement();
  });
});
