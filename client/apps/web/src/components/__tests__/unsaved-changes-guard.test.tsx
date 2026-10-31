import { render } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

const leaving = vi.hoisted(() => ({ value: false }));

vi.mock("@/lib/reload-guards", () => ({
  addReloadGuard: () => () => {},
  isLeavingRegardlessOfWork: () => leaving.value,
}));

const { UnsavedChangesGuard } = await import("../unsaved-changes-guard");

function renderGuard(when: boolean) {
  const router = createMemoryRouter([{ path: "/", element: <UnsavedChangesGuard when={when} /> }]);
  return render(<RouterProvider router={router} />);
}

function unload(): Event {
  const event = new Event("beforeunload", { cancelable: true });
  window.dispatchEvent(event);
  return event;
}

describe("UnsavedChangesGuard", () => {
  beforeEach(() => {
    leaving.value = false;
  });

  it("asks the browser to confirm leaving while the form holds unsaved edits", () => {
    renderGuard(true);

    expect(unload().defaultPrevented).toBe(true);
  });

  it("lets the page go without asking when it has nothing unsaved", () => {
    renderGuard(false);

    expect(unload().defaultPrevented).toBe(false);
  });

  it("stands aside when the app leaves because the session the edits belonged to has ended", () => {
    renderGuard(true);
    leaving.value = true;

    expect(unload().defaultPrevented).toBe(false);
  });
});
