import { renderAtRoutes } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, screen } from "@testing-library/react";
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

async function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return await renderAtRoutes(
    [
      { path: "/desk/c/$threadId/a/$slug", component: () => <DeskArtifactLinkPage /> },
      { path: "/desk", component: () => <p>Today</p> },
    ],
    {
      initialEntries: ["/desk/c/thread-1/a/rate-sheet"],
      wrapper: ({ children }) => (
        <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
      ),
    },
  );
}

describe("Desk artifact link to an artifact that is not there", () => {
  it("offers a link back to the desk", async () => {
    await renderPage();

    expect(await screen.findByText("This artifact is not here")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Back to the desk" })).toHaveAttribute("href", "/desk");
    expect(screen.queryByRole("button", { name: "Back to the desk" })).not.toBeInTheDocument();
  });

  it("leads back to the desk", async () => {
    await renderPage();

    fireEvent.click(await screen.findByRole("link", { name: "Back to the desk" }));

    expect(await screen.findByText("Today")).toBeInTheDocument();
  });
});
