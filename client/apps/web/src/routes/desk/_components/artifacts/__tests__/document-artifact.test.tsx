import type { AssistantArtifact } from "@/types/assistant";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DocumentArtifact } from "../document-artifact";

const downloadTextFile = vi.fn();

vi.mock("@trenova/shared/lib/utils", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@trenova/shared/lib/utils")>()),
  downloadTextFile: (...args: unknown[]) => downloadTextFile(...args),
}));

afterEach(() => {
  cleanup();
  downloadTextFile.mockReset();
});

function documentArtifact(body: unknown): AssistantArtifact {
  return {
    id: "art_1",
    threadId: "athr_1",
    messageId: "",
    runId: "",
    proposalId: "",
    planId: "",
    kind: "document",
    status: "Ready",
    title: "SEED-SHP-007 brief",
    payload: { format: "markdown", body },
    sourceToolCallId: "call_1",
    pinned: false,
    createdAt: 1,
    updatedAt: 1,
  };
}

describe("a published document", () => {
  it("reads as the markdown the agent wrote", () => {
    render(
      <DocumentArtifact
        artifact={documentArtifact("## Route\n\n| Stop | City |\n|---|---|\n| Pickup | Chicago |")}
      />,
    );

    expect(screen.getByRole("heading", { name: "Route" })).toBeInTheDocument();
    expect(screen.getByRole("cell", { name: "Chicago" })).toBeInTheDocument();
  });

  it("saves as a markdown file named after its title", async () => {
    render(<DocumentArtifact artifact={documentArtifact("# Brief")} />);

    await userEvent.click(screen.getByRole("button", { name: "Download" }));

    expect(downloadTextFile).toHaveBeenCalledWith(
      "seed-shp-007-brief.md",
      "# Brief",
      "text/markdown",
    );
  });

  it("says so when there is nothing in it, rather than showing a blank pane", () => {
    render(<DocumentArtifact artifact={documentArtifact(42)} />);

    expect(screen.getByText("This document is empty.")).toBeInTheDocument();
  });
});
