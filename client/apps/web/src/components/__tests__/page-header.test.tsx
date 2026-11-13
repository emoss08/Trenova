import { usePageTourStore } from "@/stores/page-tour-store";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it } from "vitest";
import { PageHeader } from "../page-header";

function renderHeader(tour?: string) {
  return render(
    <PageHeader
      title="Freight outlook"
      description="Where trucks will be short."
      includeMetadata={false}
      tour={tour}
    />,
  );
}

describe("PageHeader tour", () => {
  beforeEach(() => {
    usePageTourStore.setState({
      available: [],
      requested: null,
      showing: null,
      always: false,
    });
  });

  it("offers the page's tour in its About note and closes the note as the tour is asked for", async () => {
    const user = userEvent.setup();
    usePageTourStore.setState({ available: ["freight-outlook"] });
    renderHeader("freight-outlook");

    await user.click(screen.getByRole("button", { name: "About Freight outlook" }));
    const note = await screen.findByRole("dialog", { name: "Freight outlook" });
    expect(note).toHaveTextContent("Where trucks will be short.");

    await user.click(screen.getByRole("button", { name: "Take the tour" }));

    expect(usePageTourStore.getState().requested).toBe("freight-outlook");
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  });

  it("offers no tour while the page's tour is not on screen", async () => {
    const user = userEvent.setup();
    renderHeader("freight-outlook");

    await user.click(screen.getByRole("button", { name: "About Freight outlook" }));

    expect(await screen.findByRole("dialog", { name: "Freight outlook" })).toHaveTextContent(
      "Where trucks will be short.",
    );
    expect(screen.queryByRole("button", { name: "Take the tour" })).not.toBeInTheDocument();
  });

  it("offers no tour on a page that has none", async () => {
    const user = userEvent.setup();
    usePageTourStore.setState({ available: ["freight-outlook"] });
    renderHeader();

    await user.click(screen.getByRole("button", { name: "About Freight outlook" }));

    expect(await screen.findByRole("dialog", { name: "Freight outlook" })).toHaveTextContent(
      "Where trucks will be short.",
    );
    expect(screen.queryByRole("button", { name: "Take the tour" })).not.toBeInTheDocument();
  });
});
