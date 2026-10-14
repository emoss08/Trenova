import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useGoToChords } from "../use-go-to-chords";

const TARGETS = new Map([
  ["S", "/shipments"],
  ["B", "/billing"],
]);

function Probe() {
  useGoToChords(TARGETS);
  const { pathname } = useLocation();
  return (
    <>
      <p data-testid="where">{pathname}</p>
      <input aria-label="Search" />
    </>
  );
}

function renderProbe() {
  return render(
    <MemoryRouter initialEntries={["/"]}>
      <Routes>
        <Route path="*" element={<Probe />} />
      </Routes>
    </MemoryRouter>,
  );
}

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

/**
 * G then a module's letter goes there from anywhere, the way it does in a
 * mail client or an issue tracker. The chord is a moment long, it is ignored
 * while a field has focus, and a modifier makes it some other shortcut.
 */
describe("go-to chords", () => {
  it("goes to the module whose letter follows G", () => {
    renderProbe();

    fireEvent.keyDown(document.body, { key: "g" });
    fireEvent.keyDown(document.body, { key: "s" });

    expect(screen.getByTestId("where")).toHaveTextContent("/shipments");
  });

  it("takes the letter in either case and only once per G", () => {
    renderProbe();

    fireEvent.keyDown(document.body, { key: "G" });
    fireEvent.keyDown(document.body, { key: "B" });
    expect(screen.getByTestId("where")).toHaveTextContent("/billing");

    fireEvent.keyDown(document.body, { key: "s" });
    expect(screen.getByTestId("where")).toHaveTextContent("/billing");
  });

  it("ignores a letter that follows nothing, an unknown letter, and a modified G", () => {
    renderProbe();

    fireEvent.keyDown(document.body, { key: "s" });
    expect(screen.getByTestId("where")).toHaveTextContent("/");

    fireEvent.keyDown(document.body, { key: "g" });
    fireEvent.keyDown(document.body, { key: "x" });
    fireEvent.keyDown(document.body, { key: "s" });
    expect(screen.getByTestId("where")).toHaveTextContent("/");

    fireEvent.keyDown(document.body, { key: "g", ctrlKey: true });
    fireEvent.keyDown(document.body, { key: "s" });
    expect(screen.getByTestId("where")).toHaveTextContent("/");
  });

  it("never moves the page while a field has focus", () => {
    renderProbe();
    const input = screen.getByRole("textbox", { name: "Search" });

    fireEvent.keyDown(input, { key: "g" });
    fireEvent.keyDown(input, { key: "s" });
    expect(screen.getByTestId("where")).toHaveTextContent("/");

    fireEvent.keyDown(document.body, { key: "g" });
    fireEvent.keyDown(input, { key: "s" });
    expect(screen.getByTestId("where")).toHaveTextContent("/");
  });

  it("forgets the G after a moment", () => {
    const now = vi.spyOn(performance, "now");
    now.mockReturnValue(1000);
    renderProbe();

    fireEvent.keyDown(document.body, { key: "g" });
    now.mockReturnValue(1000 + 1500 + 1);
    fireEvent.keyDown(document.body, { key: "s" });

    expect(screen.getByTestId("where")).toHaveTextContent("/");
  });
});
