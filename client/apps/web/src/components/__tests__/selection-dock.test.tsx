import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SelectionDock } from "../selection-dock";

afterEach(cleanup);

describe("SelectionDock", () => {
  it("names an action by how many rows are selected when it asks to", () => {
    render(
      <SelectionDock
        selectedCount={3}
        actions={[
          {
            id: "download",
            label: "Download",
            labelFor: (count) => `Download ${count} transcripts`,
            onClick: vi.fn(),
          },
        ]}
        getSelected={() => []}
        onClearSelection={vi.fn()}
      />,
    );

    expect(screen.getByRole("button", { name: "Download 3 transcripts" })).toBeTruthy();
  });

  it("keeps an action's own label when it names no count", () => {
    render(
      <SelectionDock
        selectedCount={2}
        actions={[{ id: "archive", label: "Archive", onClick: vi.fn() }]}
        getSelected={() => []}
        onClearSelection={vi.fn()}
      />,
    );

    expect(screen.getByRole("button", { name: "Archive" })).toBeTruthy();
  });
});
