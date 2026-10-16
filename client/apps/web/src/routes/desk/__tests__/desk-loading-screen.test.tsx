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

  it("shows the Desk's own frame rather than the app-wide card while its loaders run", async () => {
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
    expect(screen.queryByText("App-wide loading card")).toBeNull();
    expect(screen.queryByText("The Desk")).toBeNull();
    const frame = document.querySelector('[data-slot="desk-loading-screen"]');
    expect(frame).toHaveClass("dsk");
    expect(frame?.querySelector(".dk-sb")).not.toBeNull();
    expect(frame?.querySelector(".dk-top")).not.toBeNull();
  });
});

async function openAt(path: string) {
  const router = createMemoryRouter(
    [
      {
        element: <Outlet />,
        HydrateFallback: DeskLoadingScreen,
        loader: () => new Promise(() => undefined),
        children: [{ path: "/desk/*", element: <p>The Desk</p> }],
      },
    ],
    { initialEntries: [path] },
  );
  render(<RouterProvider router={router} />);
  await screen.findByRole("status");

  return document.querySelector('[data-slot="desk-loading-screen"]') as HTMLElement;
}

/**
 * The frame is the room about to be drawn, so nothing moves when the Desk
 * takes over: the rail as the Desk draws it, and the page the address opens.
 */
describe("Desk loading screen", () => {
  it("fills the rail: its places, every shelf of conversations, the person's row", async () => {
    const frame = await openAt("/desk");
    const rail = frame.querySelector(".dk-sb") as HTMLElement;

    expect(rail.querySelectorAll(".dk-sb-top .dk-sk-ic")).toHaveLength(2);
    expect(rail.querySelectorAll(".dk-skr .dk-sk-glyph")).toHaveLength(4);
    expect(rail.querySelectorAll(".dk-skr-gh")).toHaveLength(6);
    expect(rail.querySelectorAll(".dk-skr .dk-sk-dot")).toHaveLength(34);
    expect(rail.querySelector(".dk-sb-list")).toHaveClass("dk-skr-list");
    expect(rail.querySelector(".dk-skr-me .dk-sk-me")).not.toBeNull();
  });

  it("opens Today on the greeting over the composer", async () => {
    const frame = await openAt("/desk");
    const home = frame.querySelector(".dk-home-in") as HTMLElement;

    expect(
      [...home.children].map((child) => child.className.replace("dk-sk ", "")),
    ).toEqual(["dk-sk-date", "dk-sk-greet", "dk-sk-headline", "dk-sk-cmp"]);
    expect(frame.querySelector(".dk-top .dk-sk-place")).not.toBeNull();
    expect(frame.querySelector(".dk-stage")).toBeNull();
  });

  it("opens a conversation's address on the conversation, its agent and title above", async () => {
    const frame = await openAt("/desk/t/athr_1");

    const room = frame.querySelector(".dk-mainc > .dk-stage > .dk-room") as HTMLElement;
    expect(room.querySelectorAll(".dk-scroll > .dk-grid.dk-flow > .dk-skt")).toHaveLength(2);
    expect(room.querySelector(".dk-dock .dk-c > .dk-sk-cmp")).not.toBeNull();
    expect(room.querySelector(".dk-dock .dk-c > .dk-sk-hint")).not.toBeNull();
    expect(frame.querySelector(".dk-top .dk-sk-tile")).not.toBeNull();
    expect(frame.querySelector(".dk-top .dk-sk-title")).not.toBeNull();
    expect(frame.querySelector(".dk-home-in")).toBeNull();
    expect(screen.getAllByRole("status")).toHaveLength(1);
  });

  it("opens any other place on its strip and leaves the page to it", async () => {
    const frame = await openAt("/desk/watchtower");

    expect(frame.querySelector(".dk-top .dk-sk-place")).not.toBeNull();
    expect(frame.querySelector(".dk-home-in")).toBeNull();
    expect(frame.querySelector(".dk-stage")).toBeNull();
  });
});
