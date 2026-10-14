import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { createMemoryRouter, RouterProvider, useLocation, useNavigationType } from "react-router";
import { afterEach, describe, expect, it } from "vitest";
import { useSettingsShortcut } from "../use-settings-shortcut";

function Probe() {
  useSettingsShortcut();
  const { pathname } = useLocation();
  const navigationType = useNavigationType();
  return (
    <>
      <p data-testid="where">{pathname}</p>
      <p data-testid="how">{navigationType}</p>
      <input aria-label="Search" />
    </>
  );
}

function renderAt(path: string) {
  const router = createMemoryRouter([{ path: "*", element: <Probe /> }], {
    initialEntries: [path],
  });
  render(<RouterProvider router={router} />);
  return router;
}

function pressShortcut(target: Element = document.body) {
  fireEvent.keyDown(target, { key: "S", code: "KeyS", ctrlKey: true, shiftKey: true });
  fireEvent.keyDown(target, { key: "S", code: "KeyS", metaKey: true, shiftKey: true });
}

afterEach(() => {
  cleanup();
});

describe("useSettingsShortcut", () => {
  it("opens Settings as a page from wherever the person is", () => {
    renderAt("/shipment-management/shipments");

    pressShortcut();

    expect(screen.getByTestId("where")).toHaveTextContent("/settings");
    expect(screen.getByTestId("how")).toHaveTextContent("PUSH");
  });

  it("works while a field has focus", () => {
    renderAt("/billing/queue");
    const field = screen.getByRole("textbox", { name: "Search" });
    field.focus();

    pressShortcut(field);

    expect(screen.getByTestId("where")).toHaveTextContent("/settings");
  });

  it("adds no history entry when Settings is already open", () => {
    const router = renderAt("/settings");

    pressShortcut();

    expect(screen.getByTestId("where")).toHaveTextContent("/settings");
    expect(router.state.historyAction).toBe("POP");
  });
});
