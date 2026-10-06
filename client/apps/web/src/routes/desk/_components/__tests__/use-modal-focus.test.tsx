import { fireEvent, render, screen } from "@testing-library/react";
import { useRef, useState } from "react";
import { describe, expect, it } from "vitest";
import { onRadioArrows, useModalFocus } from "../use-modal-focus";

function Dialog() {
  const ref = useRef<HTMLDivElement>(null);
  useModalFocus(ref);
  return (
    <div ref={ref} role="dialog" tabIndex={-1}>
      <button type="button">First</button>
      <button type="button">Last</button>
    </div>
  );
}

function Host() {
  const [open, setOpen] = useState(false);
  return (
    <>
      <button type="button" onClick={() => setOpen((value) => !value)}>
        Opener
      </button>
      {open && <Dialog />}
    </>
  );
}

describe("useModalFocus", () => {
  it("gives focus back to what opened the dialog when it goes", () => {
    render(<Host />);
    const opener = screen.getByRole("button", { name: "Opener" });
    opener.focus();
    fireEvent.click(opener);
    screen.getByRole("button", { name: "Last" }).focus();

    fireEvent.click(opener);
    expect(document.activeElement).toBe(opener);
  });

  it("keeps Tab going round inside the dialog", () => {
    render(<Dialog />);
    const first = screen.getByRole("button", { name: "First" });
    const last = screen.getByRole("button", { name: "Last" });

    last.focus();
    fireEvent.keyDown(last, { key: "Tab" });
    expect(document.activeElement).toBe(first);

    fireEvent.keyDown(first, { key: "Tab", shiftKey: true });
    expect(document.activeElement).toBe(last);
  });
});

function Radios() {
  const [value, setValue] = useState("a");
  const values = ["a", "b", "c"];
  return (
    <div role="radiogroup" aria-label="Letters">
      {values.map((letter) => (
        <button
          key={letter}
          type="button"
          role="radio"
          aria-checked={value === letter}
          tabIndex={value === letter ? 0 : -1}
          onKeyDown={(event) => onRadioArrows(event, values, value, setValue)}
        >
          {letter}
        </button>
      ))}
    </div>
  );
}

describe("onRadioArrows", () => {
  it("chooses and focuses the next choice, wrapping at the ends", () => {
    render(<Radios />);
    const a = screen.getByRole("radio", { name: "a" });
    a.focus();

    fireEvent.keyDown(a, { key: "ArrowLeft" });
    const c = screen.getByRole("radio", { name: "c" });
    expect(c).toHaveAttribute("aria-checked", "true");
    expect(document.activeElement).toBe(c);

    fireEvent.keyDown(c, { key: "ArrowRight" });
    expect(screen.getByRole("radio", { name: "a" })).toHaveAttribute("aria-checked", "true");
  });
});
