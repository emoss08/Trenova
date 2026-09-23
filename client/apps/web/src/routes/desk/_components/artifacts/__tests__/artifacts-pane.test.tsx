import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ArtifactsPane } from "../artifacts-pane";

const listArtifacts = vi.fn();

vi.mock("@/services/api", () => ({
  apiService: {
    assistantService: {
      listArtifacts: (...args: unknown[]) => listArtifacts(...args),
      pinArtifact: vi.fn(),
    },
  },
}));

afterEach(() => {
  cleanup();
  listArtifacts.mockReset();
});

function renderPane() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <ArtifactsPane threadId="athr_1" liveArtifactIds={[]} onClose={() => undefined} />
    </QueryClientProvider>,
  );
}

describe("the artifacts pane", () => {
  it("names what to ask for, with the question itself rather than a placeholder", async () => {
    listArtifacts.mockResolvedValue({ results: [] });

    renderPane();

    expect(await screen.findByText("Nothing here yet")).toBeInTheDocument();
    expect(screen.getByText("“Which shipments are in transit?”")).toBeInTheDocument();
    expect(screen.queryByText(/\{example\}/)).toBeNull();
  });

  /**
   * A list that could not be read is not an empty one.
   *
   * The pane used to say "Nothing here yet" whatever went wrong, which is the
   * one thing a person who just watched a table being made cannot believe.
   */
  it("says the list could not be read, and reads it again on request", async () => {
    listArtifacts.mockRejectedValueOnce(new Error("offline"));
    listArtifacts.mockResolvedValueOnce({ results: [] });

    renderPane();

    expect(
      await screen.findByText("This conversation's artifacts could not be loaded."),
    ).toBeInTheDocument();
    expect(screen.queryByText("Nothing here yet")).toBeNull();

    await userEvent.click(screen.getByRole("button", { name: "Try again" }));

    expect(await screen.findByText("Nothing here yet")).toBeInTheDocument();
  });
});
