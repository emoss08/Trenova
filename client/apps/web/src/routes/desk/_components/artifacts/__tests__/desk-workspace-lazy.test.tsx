import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { LazyDeskWorkspace, preloadDeskWorkspace } from "../desk-workspace-lazy";

const loads = vi.hoisted(() => ({ count: 0 }));

vi.mock("../desk-workspace", () => {
  loads.count += 1;
  return {
    DeskWorkspace: ({ threadId }: { threadId: string }) => <p>Workspace for {threadId}</p>,
  };
});

/**
 * The workspace's code is read when it is first wanted, not with every
 * conversation, and its outline holds the pane while the code arrives.
 */
describe("LazyDeskWorkspace", () => {
  it("does not read the workspace's code until it is wanted", () => {
    expect(loads.count).toBe(0);
  });

  it("holds the pane with the workspace's outline, then draws the workspace", async () => {
    const { container } = render(
      <LazyDeskWorkspace
        threadId="athr_1"
        liveArtifacts={{ ids: [], revision: 0 }}
        onClose={vi.fn()}
      />,
    );

    expect(container.querySelector(".dk-apx[aria-busy]")).not.toBeNull();
    expect(await screen.findByText("Workspace for athr_1")).toBeInTheDocument();
    expect(container.querySelector(".dk-apx[aria-busy]")).toBeNull();
  });

  it("reads the code once however often it is asked for", async () => {
    const first = preloadDeskWorkspace();
    const second = preloadDeskWorkspace();

    expect(second).toBe(first);
    await first;
    expect(loads.count).toBe(1);
  });
});
