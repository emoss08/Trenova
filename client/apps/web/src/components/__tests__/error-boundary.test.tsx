import {
  DataTableLazyComponent,
  NotFoundRoute,
  RouteErrorBoundary,
} from "@trenova/shared/components/error-boundary";
import { QueryClient, QueryClientProvider, useSuspenseQuery } from "@tanstack/react-query";
import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createMemoryRouter, Outlet, RouterProvider } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

function renderWithQueryClient(ui: React.ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>);
}

function Rows({ load }: { load: () => Promise<string> }) {
  const { data } = useSuspenseQuery({ queryKey: ["rows"], queryFn: load });
  return <p>{data}</p>;
}

let consoleError: ReturnType<typeof vi.spyOn>;

beforeEach(() => {
  consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
});

afterEach(() => {
  consoleError.mockRestore();
  vi.restoreAllMocks();
});

describe("DataTableLazyComponent", () => {
  it("names a failed fetch as a connection problem instead of printing the TypeError", async () => {
    const load = vi.fn().mockRejectedValue(new TypeError("Failed to fetch"));

    renderWithQueryClient(
      <DataTableLazyComponent>
        <Rows load={load} />
      </DataTableLazyComponent>,
    );

    expect(await screen.findByRole("heading", { name: "Can't reach Trenova" })).toBeVisible();
    expect(screen.queryByText("TypeError")).not.toBeInTheDocument();
  });

  it("refetches the failed query when the person tries again", async () => {
    const load = vi
      .fn()
      .mockRejectedValueOnce(new TypeError("Failed to fetch"))
      .mockResolvedValue("12 shipments");

    renderWithQueryClient(
      <DataTableLazyComponent>
        <Rows load={load} />
      </DataTableLazyComponent>,
    );

    await userEvent.click(await screen.findByRole("button", { name: "Try again" }));

    expect(await screen.findByText("12 shipments")).toBeVisible();
    expect(load).toHaveBeenCalledTimes(2);
  });

  it("retries on its own when the browser comes back online", async () => {
    const onLine = vi.spyOn(navigator, "onLine", "get").mockReturnValue(false);
    const load = vi
      .fn()
      .mockRejectedValueOnce(new TypeError("Failed to fetch"))
      .mockResolvedValue("12 shipments");

    renderWithQueryClient(
      <DataTableLazyComponent>
        <Rows load={load} />
      </DataTableLazyComponent>,
    );

    expect(await screen.findByRole("heading", { name: "You're offline" })).toBeVisible();

    onLine.mockReturnValue(true);
    act(() => {
      window.dispatchEvent(new Event("online"));
    });

    expect(await screen.findByText("12 shipments")).toBeVisible();
  });
});

function Shell() {
  return (
    <div>
      <nav>App navigation</nav>
      <Outlet />
    </div>
  );
}

function renderRoutes(initialPath: string, loader?: () => never) {
  const router = createMemoryRouter(
    [
      {
        element: <Shell />,
        errorElement: <RouteErrorBoundary />,
        children: [
          { path: "/", element: <p>Home page</p> },
          {
            element: <Outlet />,
            errorElement: <RouteErrorBoundary embedded />,
            children: [
              { path: "/shipments", loader, element: <p>Shipments</p> },
              { path: "*", element: <NotFoundRoute /> },
            ],
          },
        ],
      },
    ],
    { initialEntries: [initialPath] },
  );
  render(<RouterProvider router={router} />);
  return router;
}

describe("route errors", () => {
  it("keeps the navigation on screen for an unknown address and shows the path", async () => {
    renderRoutes("/admin/agent-control4");

    expect(await screen.findByRole("heading", { name: "We can't find that page" })).toBeVisible();
    expect(screen.getByText("App navigation")).toBeVisible();
    expect(screen.getByText("/admin/agent-control4")).toBeVisible();
  });

  it("goes to the dashboard from the not-found state", async () => {
    const router = renderRoutes("/nowhere");

    await userEvent.click(await screen.findByRole("button", { name: "Go to dashboard" }));

    expect(router.state.location.pathname).toBe("/");
    expect(await screen.findByText("Home page")).toBeVisible();
  });

  it("offers going back only when there is an in-app entry to go back to", async () => {
    const router = renderRoutes("/nowhere");

    expect(await screen.findByRole("heading", { name: "We can't find that page" })).toBeVisible();
    expect(screen.queryByRole("button", { name: "Go back" })).not.toBeInTheDocument();

    await act(() => router.navigate("/elsewhere"));

    expect(await screen.findByRole("button", { name: "Go back" })).toBeVisible();
  });

  it("explains a refused permission without a reload that cannot help", async () => {
    renderRoutes("/shipments", () => {
      throw new Response("Missing permission: shipment:read", {
        status: 403,
        statusText: "Missing shipment:read",
      });
    });

    expect(
      await screen.findByRole("heading", { name: "You don't have access to this" }),
    ).toBeVisible();
    expect(screen.getByText("App navigation")).toBeVisible();
    expect(screen.queryByRole("button", { name: "Reload page" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Go to dashboard" })).toBeVisible();
  });
});
