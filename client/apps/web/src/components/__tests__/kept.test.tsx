import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it } from "vitest";
import { Kept } from "../kept";

function Counter() {
  const [count, setCount] = useState(0);

  return (
    <button type="button" onClick={() => setCount((n) => n + 1)}>
      {`pressed ${count}`}
    </button>
  );
}

/** A kept view is there when a person comes back to it, as they left it. */
describe("a kept view", () => {
  it("is not mounted until it is first shown", () => {
    render(
      <Kept shown={false}>
        <Counter />
      </Kept>,
    );

    expect(screen.queryByText(/pressed/)).toBeNull();
  });

  it("keeps its state while hidden, and is out of sight and out of reach meanwhile", async () => {
    const user = userEvent.setup();
    const view = (shown: boolean) => (
      <Kept shown={shown}>
        <Counter />
      </Kept>
    );
    const { rerender } = render(view(true));
    await user.click(screen.getByRole("button", { name: "pressed 0" }));

    rerender(view(false));
    expect(screen.queryByRole("button")).toBeNull();

    rerender(view(true));
    expect(screen.getByRole("button", { name: "pressed 1" })).toBeVisible();
  });
});
