import { routes } from "@/router";
import { DeskLoadingScreen } from "@/routes/desk/desk-loading-screen";
import { DeskShellLayout } from "@/routes/desk/shell-layout";
import { render, screen } from "@testing-library/react";
import { isValidElement } from "react";
import { createMemoryRouter, Outlet, RouterProvider, type RouteObject } from "react-router";
import { describe, expect, it } from "vitest";

function routeContaining(entries: RouteObject[], path: string): RouteObject | null {
  for (const entry of entries) {
    if (entry.children?.some((child) => child.path === path)) return entry;
    if (entry.children) {
      const found = routeContaining(entry.children, path);
      if (found) return found;
    }
  }
  return null;
}

function AppWideFallback() {
  return <p>App-wide loading card</p>;
}

describe("Desk first load", () => {
  it("gives the Desk shell its own hydrate fallback", () => {
    const shell = routeContaining(routes, "/desk");

    expect(shell).not.toBeNull();
    expect(isValidElement(shell?.element) && shell.element.type).toBe(DeskShellLayout);
    expect(shell?.HydrateFallback).toBe(DeskLoadingScreen);
  });

  it("shows the desk visitor rather than the app-wide card while the Desk's loaders run", async () => {
    const router = createMemoryRouter(
      [
        {
          element: <Outlet />,
          HydrateFallback: AppWideFallback,
          children: [
            {
              element: <Outlet />,
              HydrateFallback: DeskLoadingScreen,
              loader: () => new Promise(() => undefined),
              children: [{ path: "/desk", element: <p>The Desk</p> }],
            },
          ],
        },
      ],
      { initialEntries: ["/desk"] },
    );

    render(<RouterProvider router={router} />);

    const status = await screen.findByRole("status");
    expect(status).toHaveTextContent("Opening the desk");
    expect(status.querySelector('[data-slot="desk-loading-mark"]')).not.toBeNull();
    expect(screen.queryByText("App-wide loading card")).toBeNull();
    expect(screen.queryByText("The Desk")).toBeNull();
    expect(document.querySelector('[data-slot="desk-loading-screen"]')).toHaveClass(
      "bg-desk-canvas",
    );
  });
});
