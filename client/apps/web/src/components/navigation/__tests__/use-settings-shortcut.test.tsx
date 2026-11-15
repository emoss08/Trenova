import { createTestRouter, TestRouter } from "@trenova/shared/testing/test-router";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { useLocation, useRouterState } from "@tanstack/react-router";
import { afterEach, describe, expect, it } from "vitest";
import { useSettingsShortcut } from "../use-settings-shortcut";

function Probe() {
  useSettingsShortcut();
  const pathname = useLocation({ select: (location) => location.pathname });
  // The router numbers each history entry it writes: a push moves to the next one.
  const entry = useRouterState({ select: (state) => state.location.state.__TSR_index ?? 0 });
  return (
    <>
      <p data-testid="where">{pathname}</p>
      <p data-testid="entry">{entry}</p>
      <input aria-label="Search" />
    </>
  );
}

function renderAt(path: string) {
  render(
    <TestRouter router={createTestRouter(undefined, [path])}>
      <Probe />
    </TestRouter>,
  );
}

function pressShortcut(target: Element = document.body) {
  fireEvent.keyDown(target, { key: "S", code: "KeyS", ctrlKey: true, shiftKey: true });
  fireEvent.keyDown(target, { key: "S", code: "KeyS", metaKey: true, shiftKey: true });
}

afterEach(() => {
  cleanup();
});

describe("useSettingsShortcut", () => {
  it("opens Settings as a page from wherever the person is", async () => {
    renderAt("/shipment-management/shipments");

    pressShortcut();

    await waitFor(() => expect(screen.getByTestId("where")).toHaveTextContent("/settings"));
    expect(screen.getByTestId("entry")).toHaveTextContent("1");
  });

  it("works while a field has focus", async () => {
    renderAt("/billing/queue");
    const field = screen.getByRole("textbox", { name: "Search" });
    field.focus();

    pressShortcut(field);

    await waitFor(() => expect(screen.getByTestId("where")).toHaveTextContent("/settings"));
  });

  it("adds no history entry when Settings is already open", async () => {
    renderAt("/settings");

    pressShortcut();
    await new Promise((resolve) => setTimeout(resolve, 0));

    expect(screen.getByTestId("where")).toHaveTextContent("/settings");
    expect(screen.getByTestId("entry")).toHaveTextContent("0");
  });
});
