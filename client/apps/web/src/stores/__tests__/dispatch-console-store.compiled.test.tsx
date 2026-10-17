import { act, fireEvent, render, screen } from "@testing-library/react";
import { StrictMode, useState } from "react";
import { afterEach, describe, expect, it } from "vitest";
import { useDispatchConsoleStore } from "@/stores/dispatch-console-store";

// Runs under the React Compiler, as the app is built. The compiler caches any call whose
// name does not read as a hook, so a store read through a property such as `.use.x()`
// ran only on the first render and every hook after it moved one slot: the console threw
// "Cannot read properties of undefined (reading 'next')" on its second render.

function Console() {
  const dragPreview = useDispatchConsoleStore((s) => s.dragPreview);
  const setDragPreview = useDispatchConsoleStore((s) => s.setDragPreview);
  const reset = useDispatchConsoleStore((s) => s.reset);
  const [renders, setRenders] = useState(0);

  return (
    <div>
      <output aria-label="Preview">{dragPreview ?? "none"}</output>
      <output aria-label="Renders">{renders}</output>
      <button type="button" onClick={() => setDragPreview("PRO-1 · Dallas → Tulsa")}>
        Drag
      </button>
      <button type="button" onClick={reset}>
        Drop
      </button>
      <button type="button" onClick={() => setRenders((count) => count + 1)}>
        Render again
      </button>
    </div>
  );
}

function renderConsole() {
  render(
    <StrictMode>
      <Console />
    </StrictMode>,
  );
}

afterEach(() => {
  act(() => useDispatchConsoleStore.getState().reset());
});

describe("dispatch console store under the React Compiler", () => {
  it("renders the console again after its own state changes", () => {
    renderConsole();

    fireEvent.click(screen.getByRole("button", { name: "Render again" }));
    fireEvent.click(screen.getByRole("button", { name: "Render again" }));

    expect(screen.getByLabelText("Renders")).toHaveTextContent("2");
    expect(screen.getByLabelText("Preview")).toHaveTextContent("none");
  });

  it("shows each change the store makes, through the actions it hands out", () => {
    renderConsole();

    fireEvent.click(screen.getByRole("button", { name: "Drag" }));
    expect(screen.getByLabelText("Preview")).toHaveTextContent("PRO-1 · Dallas → Tulsa");

    fireEvent.click(screen.getByRole("button", { name: "Drop" }));
    expect(screen.getByLabelText("Preview")).toHaveTextContent("none");
  });

  it("shows a change made outside the component", () => {
    renderConsole();

    act(() => useDispatchConsoleStore.getState().setDragPreview("PRO-2 · Austin → Waco"));

    expect(screen.getByLabelText("Preview")).toHaveTextContent("PRO-2 · Austin → Waco");
  });
});
