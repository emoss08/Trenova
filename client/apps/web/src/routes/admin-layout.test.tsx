import { QueryClient, QueryClientProvider, useSuspenseQuery } from "@tanstack/react-query";
import { screen } from "@testing-library/react";
import { Suspense, type ReactNode } from "react";
import { renderAtRoutes } from "@trenova/shared/testing/test-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AdminLayout } from "./admin-layout";

function Settings({ load }: { load: () => Promise<string> }) {
  const { data } = useSuspenseQuery({ queryKey: ["admin-settings"], queryFn: load });
  return <p>{data}</p>;
}

async function renderAdminPage(page: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderAtRoutes([{ path: "/admin/settings", component: () => page }], {
    initialEntries: ["/admin/settings"],
    layout: AdminLayout,
    wrapper: ({ children }) => (
      <QueryClientProvider client={client}>
        <Suspense fallback={null}>{children}</Suspense>
      </QueryClientProvider>
    ),
  });
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
    const { container } = await renderAdminPage(<Settings load={load} />);

    await new Promise((resolve) => setTimeout(resolve, 0));
    expect(load).toHaveBeenCalled();
    expect(screen.queryByText("Billing controls")).not.toBeInTheDocument();
    expect(container.querySelector("[data-slot=skeleton]")).toBeNull();

    answer("Billing controls");
    expect(await screen.findByText("Billing controls")).toBeVisible();
  });

  it("draws the failure in place of the page that failed", async () => {
    const load = vi.fn().mockRejectedValue(new TypeError("Failed to fetch"));
    await renderAdminPage(<Settings load={load} />);

    expect(await screen.findByRole("heading", { name: "Can't reach Trenova" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Try again" })).toBeVisible();
  });
});
