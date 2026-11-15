import { renderAtRoutes } from "@trenova/shared/testing/test-router";
import { act, screen } from "@testing-library/react";
import { useNavigate } from "@tanstack/react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

const leaving = vi.hoisted(() => ({ value: false }));

vi.mock("@/lib/reload-guards", () => ({
  addReloadGuard: () => () => {},
  isLeavingRegardlessOfWork: () => leaving.value,
}));

const { UnsavedChangesGuard } = await import("../unsaved-changes-guard");

function Form({ when }: { when: boolean }) {
  const navigate = useNavigate();
  return (
    <>
      <UnsavedChangesGuard when={when} />
      <button type="button" onClick={() => void navigate({ href: "/elsewhere" })}>
        Leave
      </button>
      <button type="button" onClick={() => void navigate({ to: "/", search: { tab: "notes" } })}>
        Switch tab
      </button>
    </>
  );
}

function renderGuard(when: boolean) {
  return renderAtRoutes(
    [
      { path: "/", component: () => <Form when={when} /> },
      { path: "/elsewhere", component: () => <p>Elsewhere</p> },
    ],
    { initialEntries: ["/"] },
  );
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

  it("asks the browser to confirm leaving while the form holds unsaved edits", async () => {
    await renderGuard(true);

    expect(unload().defaultPrevented).toBe(true);
  });

  it("lets the page go without asking when it has nothing unsaved", async () => {
    await renderGuard(false);

    expect(unload().defaultPrevented).toBe(false);
  });

  it("stands aside when the app leaves because the session the edits belonged to has ended", async () => {
    await renderGuard(true);
    leaving.value = true;

    expect(unload().defaultPrevented).toBe(false);
  });

  it("holds a move to another page until the person chooses, and stays when they do", async () => {
    const { router } = await renderGuard(true);

    await act(async () => screen.getByRole("button", { name: "Leave" }).click());
    expect(screen.getByText("Discard unsaved changes?")).toBeInTheDocument();

    await act(async () => screen.getByRole("button", { name: "Stay" }).click());
    expect(router.state.location.pathname).toBe("/");
    expect(screen.queryByText("Elsewhere")).not.toBeInTheDocument();
  });

  it("leaves the page when the person discards the changes", async () => {
    const { router } = await renderGuard(true);

    await act(async () => screen.getByRole("button", { name: "Leave" }).click());
    await act(async () => screen.getByRole("button", { name: "Discard changes" }).click());

    expect(router.state.location.pathname).toBe("/elsewhere");
    expect(await screen.findByText("Elsewhere")).toBeInTheDocument();
  });

  it("never holds a change of the query on the same page", async () => {
    const { router } = await renderGuard(true);

    await act(async () => screen.getByRole("button", { name: "Switch tab" }).click());

    expect(screen.queryByText("Discard unsaved changes?")).not.toBeInTheDocument();
    expect(router.state.location.href).toBe("/?tab=notes");
  });
});
