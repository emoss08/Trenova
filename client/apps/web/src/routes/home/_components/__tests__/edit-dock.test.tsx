import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ActionDock } from "@/components/action-dock";
import { HomeEditDock } from "../edit-dock";

/**
 * A transform on any ancestor becomes the containing block for a
 * `position: fixed` descendant. The edit dock is `fixed bottom-6`, so an
 * animated wrapper around it parks the dock wherever the wrapper sits in the
 * document flow — high on the page — until the transform settles to none, at
 * which point it jumps to the viewport bottom. The fix is that the fixed
 * element is the animated element, with nothing wrapping it.
 */
function renderDock() {
  return render(
    <HomeEditDock
      dirty={false}
      saving={false}
      addDisabled={false}
      maxWidgets={12}
      onAdd={() => undefined}
      onDiscard={() => undefined}
      onSave={() => undefined}
    />,
  );
}

describe("HomeEditDock", () => {
  it("makes the fixed dock the outermost element it renders", () => {
    const { container } = renderDock();

    const root = container.firstElementChild;
    expect(root).not.toBeNull();
    expect(root).toHaveClass("fixed");
  });

  it("renders no element between the mount point and the fixed dock", () => {
    const { container } = renderDock();

    const fixed = container.querySelector(".fixed");
    expect(fixed).not.toBeNull();
    expect(fixed?.parentElement).toBe(container);
  });
});

describe("ActionDock", () => {
  it("keeps the fixed element as its root when animated", () => {
    const { container } = render(
      <ActionDock animated>
        <button type="button">Save</button>
      </ActionDock>,
    );

    const root = container.firstElementChild;
    expect(root).toHaveClass("fixed", "bottom-6");
    expect(container.querySelectorAll(".fixed")).toHaveLength(1);
  });

  it("still renders the fixed shell without the animation", () => {
    const { container } = render(
      <ActionDock>
        <button type="button">Save</button>
      </ActionDock>,
    );

    expect(container.firstElementChild).toHaveClass("fixed", "bottom-6");
  });
});
