import { render } from "@testing-library/react";
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

    expect(container.querySelectorAll('[aria-busy] [data-slot="skeleton"]')).toHaveLength(2);
    expect(container.querySelector('[data-slot="desk-loading-mark"]')).toBeNull();
  });
});
