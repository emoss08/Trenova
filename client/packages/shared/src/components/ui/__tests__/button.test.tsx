import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { Button } from "../button";

afterEach(cleanup);

describe("a button whose request is in flight", () => {
  it("keeps its content in place, hidden, so it holds the size it had at rest", () => {
    const { rerender } = render(
      <Button>
        <svg data-testid="icon" />
        Save changes
      </Button>,
    );
    const button = screen.getByRole("button", { name: "Save changes" });
    const resting = button.textContent;

    rerender(
      <Button isLoading>
        <svg data-testid="icon" />
        Save changes
      </Button>,
    );

    expect(button).toHaveAttribute("data-loading");
    expect(button.textContent).toBe(resting);
    expect(screen.getByTestId("icon").parentElement).toBe(button);
    expect(button.className).toContain("data-loading:[-webkit-text-fill-color:transparent]");
    expect(button.className).toContain(
      "data-loading:[&>*:not([data-slot=button-spinner])]:invisible",
    );
  });

  it("lays the spinner over its content, out of the flow that sizes it", () => {
    render(<Button isLoading>Save changes</Button>);

    const spinner = screen.getByRole("button").querySelector('[data-slot="button-spinner"]');
    expect(spinner).toHaveClass("absolute", "inset-0");
    expect(spinner?.querySelector("svg")).not.toBeNull();
    expect(screen.getByRole("button")).toHaveClass("relative");
  });

  it("cannot be pressed again and says it is busy", () => {
    render(<Button isLoading>Save changes</Button>);

    const button = screen.getByRole("button");
    expect(button).toBeDisabled();
    expect(button).toHaveAttribute("aria-busy", "true");
  });

  it("tells a screen reader what it is doing without drawing the words", () => {
    render(
      <Button isLoading loadingText="Saving...">
        Save changes
      </Button>,
    );

    expect(screen.getByText("Saving...")).toHaveClass("sr-only");
    expect(screen.getByRole("button", { name: "Saving..." })).toBeDisabled();
  });

  it("keeps its shortcut in place too", () => {
    render(
      <Button isLoading shortcut="⌘S">
        Save
      </Button>,
    );

    expect(screen.getByRole("button")).toHaveTextContent("Save⌘S");
  });
});

describe("a button at rest", () => {
  it("has no spinner and is not marked as loading", () => {
    render(<Button>Save changes</Button>);

    const button = screen.getByRole("button", { name: "Save changes" });
    expect(button).not.toHaveAttribute("data-loading");
    expect(button.querySelector('[data-slot="button-spinner"]')).toBeNull();
    expect(button).toBeEnabled();
  });

  it("stays where a caller positions it, since a caller's position replaces its own", () => {
    render(<Button className="absolute top-2 right-2">Close</Button>);

    const button = screen.getByRole("button", { name: "Close" });
    expect(button).toHaveClass("absolute");
    expect(button).not.toHaveClass("relative");
  });
});
