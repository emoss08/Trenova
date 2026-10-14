import { QueryClient, QueryClientProvider, useSuspenseQuery } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { Suspense, type ReactNode } from "react";
import { createMemoryRouter, RouterProvider } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AdminLayout } from "./admin-layout";

function Settings({ load }: { load: () => Promise<string> }) {
  const { data } = useSuspenseQuery({ queryKey: ["admin-settings"], queryFn: load });
  return <p>{data}</p>;
}

function renderAdminPage(page: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const router = createMemoryRouter(
    [{ path: "/admin", Component: AdminLayout, children: [{ path: "settings", element: page }] }],
    { initialEntries: ["/admin/settings"] },
  );
  return render(
    <QueryClientProvider client={client}>
      <Suspense fallback={<p>Shell is holding the screen</p>}>
        <RouterProvider router={router} />
      </Suspense>
    </QueryClientProvider>,
  );
}

let consoleError: ReturnType<typeof vi.spyOn>;

beforeEach(() => {
  consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
});

afterEach(() => {
  consoleError.mockRestore();
});

describe("AdminLayout", () => {
  // Admin pages draw their own skeletons; a stray suspension must not be
  // painted over with one generic shape for every admin page.
  it("draws no loading shape of its own while a page waits", async () => {
    let answer: (value: string) => void = () => {};
    const load = vi.fn(
      () =>
        new Promise<string>((resolve) => {
          answer = resolve;
        }),
    );
    const { container } = renderAdminPage(<Settings load={load} />);

    expect(await screen.findByText("Shell is holding the screen")).toBeVisible();
    expect(container.querySelector("[data-slot=skeleton]")).toBeNull();

    answer("Billing controls");
    expect(await screen.findByText("Billing controls")).toBeVisible();
  });

  it("draws the failure in place of the page that failed", async () => {
    const load = vi.fn().mockRejectedValue(new TypeError("Failed to fetch"));
    renderAdminPage(<Settings load={load} />);

    expect(await screen.findByRole("heading", { name: "Can't reach Trenova" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Try again" })).toBeVisible();
  });
});
