import { routeById } from "@/test/route-tree";
import { DeskLoadingScreen } from "@/routes/desk/desk-loading-screen";
import { DeskShellLayout } from "@/routes/desk/shell-layout";
import { createTestRouter } from "@trenova/shared/testing/test-router";
import { render, screen } from "@testing-library/react";
import { createRootRoute, createRoute, Outlet, RouterProvider } from "@tanstack/react-router";
import { describe, expect, it } from "vitest";

// The shell's own timing, so the screen under test opens when the app's does.
const shellPending = {
  pendingMs: routeById("/_desk").options.pendingMs,
  pendingMinMs: routeById("/_desk").options.pendingMinMs,
};

function AppWideFallback() {
  return <p>App-wide loading card</p>;
}

// The app's first-load shape: an app-wide card at the root, the Desk shell with its own
// loading screen, and a session gate that has not answered yet.
function deskStillOpening(path: string) {
  const rootRoute = createRootRoute({ component: Outlet, pendingComponent: AppWideFallback });
  const shell = createRoute({
    getParentRoute: () => rootRoute,
    id: "deskShell",
    component: Outlet,
    pendingComponent: DeskLoadingScreen,
    ...shellPending,
    beforeLoad: () => new Promise<void>(() => undefined),
  });
  const routeTree = rootRoute.addChildren([
    shell.addChildren([
      createRoute({
        getParentRoute: () => shell,
        path: "/desk/$",
        component: () => <p>The Desk</p>,
      }),
    ]),
  ]);
  const router = createTestRouter(routeTree, [path]);
  void router.load();
  render(<RouterProvider router={router} />);
}

describe("Desk first load", () => {
  it("gives the Desk shell its own loading screen, drawn at once", () => {
    const shell = routeById("/_desk");

    expect(shell.options.component).toBe(DeskShellLayout);
    expect(shell.options.pendingComponent).toBe(DeskLoadingScreen);
    expect(shell.options.pendingMs).toBe(0);
    expect(shell.options.pendingMinMs).toBe(0);
  });

  it("shows the Desk's own frame rather than the app-wide card while its gates run", async () => {
    deskStillOpening("/desk");

    const status = await screen.findByRole("status");
    expect(status).toHaveTextContent("Opening the desk");
    expect(screen.queryByText("App-wide loading card")).toBeNull();
    expect(screen.queryByText("The Desk")).toBeNull();
    const frame = document.querySelector('[data-slot="desk-loading-screen"]');
    expect(frame).toHaveClass("dsk");
    expect(frame?.querySelector("aside")).not.toBeNull();
    expect(frame?.querySelector(".dk-tb")).not.toBeNull();
  });
});

async function openAt(path: string) {
  deskStillOpening(path);
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
    // The rail is drawn with utilities, so it is found by what it is: the frame's aside,
    // its top row first and its list after.
    const rail = frame.querySelector("aside") as HTMLElement;
    const top = rail.firstElementChild as HTMLElement;

    expect(top.querySelectorAll(".dk-sk-ic")).toHaveLength(3);
    expect(top.firstElementChild).toHaveClass("dk-sk-ic");
    expect(rail.querySelectorAll(".dk-skr .dk-sk-glyph")).toHaveLength(5);
    expect(rail.querySelectorAll(".dk-skr-gh")).toHaveLength(6);
    expect(rail.querySelectorAll(".dk-skr .dk-sk-dot")).toHaveLength(34);
    expect(top.nextElementSibling).toHaveClass("dk-skr-list");
    expect(rail.querySelector(".dk-skr-me .dk-sk-me")).not.toBeNull();
  });

  it("opens Today on the greeting over the composer", async () => {
    const frame = await openAt("/desk");
    const home = frame.querySelector(".dk-dy") as HTMLElement;

    expect([...home.children].map((child) => child.className.replace("dk-sk ", ""))).toEqual([
      "dk-sk-date",
      "dk-sk-greet",
      "dk-sk-headline",
      "dk-sk-cmp",
    ]);
    expect(frame.querySelector(".dk-tb .dk-sk-place")).not.toBeNull();
    expect(frame.querySelector(".dk-stage")).toBeNull();
  });

  it("opens a conversation's address on the conversation, its agent and title above", async () => {
    const frame = await openAt("/desk/t/athr_1");

    const room = frame.querySelector(".dk-mainc > .dk-stage > .dk-room") as HTMLElement;
    expect(room.querySelectorAll(".dk-scroll > .dk-grid.dk-flow > .dk-skt")).toHaveLength(2);
    expect(room.querySelector(".dk-dock .dk-c > .dk-sk-cmp")).not.toBeNull();
    expect(room.querySelector(".dk-dock .dk-c > .dk-sk-hint")).not.toBeNull();
    expect(frame.querySelector(".dk-tb .dk-sk-tile")).not.toBeNull();
    expect(frame.querySelector(".dk-tb .dk-sk-title")).not.toBeNull();
    expect(frame.querySelector(".dk-dy")).toBeNull();
    expect(screen.getAllByRole("status")).toHaveLength(1);
  });

  it("opens any other place on its strip and leaves the page to it", async () => {
    const frame = await openAt("/desk/scheduled");

    expect(frame.querySelector(".dk-tb .dk-sk-place")).not.toBeNull();
    expect(frame.querySelector(".dk-dy")).toBeNull();
    expect(frame.querySelector(".dk-stage")).toBeNull();
  });
});
