import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { describe, expect, it, vi } from "vitest";
import { DeskArtifactLinkPage } from "../artifact-link-page";

vi.mock("@/lib/queries/assistant", () => ({
  assistant: {
    artifacts: (threadId: string) => ({
      _ctx: {
        slug: (slug: string) => ({
          queryKey: ["assistant", "artifacts", threadId, "slug", slug],
          queryFn: () => Promise.reject(new Error("artifact not found")),
        }),
      },
    }),
  },
}));

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/desk/c/thread-1/a/rate-sheet"]}>
        <Routes>
          <Route path="/desk/c/:threadId/a/:slug" element={<DeskArtifactLinkPage />} />
          <Route path="/desk" element={<p>Today</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("Desk artifact link to an artifact that is not there", () => {
  it("offers a link back to the desk", async () => {
    renderPage();

    expect(await screen.findByText("This artifact is not here")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Back to the desk" })).toHaveAttribute("href", "/desk");
    expect(screen.queryByRole("button", { name: "Back to the desk" })).not.toBeInTheDocument();
  });

  it("leads back to the desk", async () => {
    renderPage();

    fireEvent.click(await screen.findByRole("link", { name: "Back to the desk" }));

    expect(screen.getByText("Today")).toBeInTheDocument();
  });
});
