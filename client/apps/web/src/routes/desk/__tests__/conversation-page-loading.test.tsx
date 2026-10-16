import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { describe, expect, it, vi } from "vitest";
import { DeskConversationPage } from "../conversation-page";

vi.mock("../_components/desk-layout", () => ({
  useDesk: () => ({ isLoading: true, activeThread: null }),
}));

vi.mock("../_components/desk-conversation", () => ({ DeskConversation: () => null }));

describe("Desk conversation page while the Desk loads", () => {
  it("holds its place with skeletons, not the desk visitor", () => {
    const { container } = render(
      <MemoryRouter initialEntries={["/desk/t/thread-1"]}>
        <Routes>
          <Route path="/desk/t/:threadId" element={<DeskConversationPage />} />
        </Routes>
      </MemoryRouter>,
    );

    // The conversation's own room: two turns on the transcript's grid in its
    // scrolling column, and the composer docked under it with its hint.
    const busy = container.querySelector("[aria-busy]");
    expect(busy).toHaveClass("dk-stage");
    const transcript = busy?.querySelector(".dk-room > .dk-scroll > .dk-grid.dk-flow");
    expect(transcript?.querySelectorAll(".dk-skt")).toHaveLength(2);
    expect(transcript?.querySelector(".dk-sk-q")).not.toBeNull();
    expect(busy?.querySelector(".dk-room > .dk-dock .dk-c > .dk-sk-cmp")).not.toBeNull();
    expect(busy?.querySelector(".dk-room > .dk-dock .dk-c > .dk-sk-hint")).not.toBeNull();
    expect(screen.getByRole("status")).toHaveTextContent("Reading the conversation…");
    expect(container.querySelector('[data-slot="desk-loading-mark"]')).toBeNull();
  });
});
