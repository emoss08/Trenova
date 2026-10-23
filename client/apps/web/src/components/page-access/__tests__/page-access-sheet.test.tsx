import { useAppDialogsStore } from "@/stores/app-dialogs-store";
import { act, cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { PageAccessPanelSkeleton, PageAccessSheet } from "../page-access-sheet";

const allowed = vi.fn<() => boolean>();

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: allowed(), isLoading: false }),
}));
vi.mock("../page-access-panel", () => ({
  PageAccessPanel: ({ location, canPreview }: { location: string; canPreview: boolean }) => (
    <p>{`panel ${location} preview:${canPreview}`}</p>
  ),
}));

beforeEach(() => {
  useAppDialogsStore.setState({ active: "page-access", prepared: [] });
});

afterEach(() => {
  cleanup();
  allowed.mockReset();
});

function renderAt(path: string) {
  render(
    <MemoryRouter initialEntries={[path]}>
      <PageAccessSheet />
    </MemoryRouter>,
  );
}

describe("PageAccessSheet", () => {
  it("describes the screen it was opened on, with the preview for whoever may read roles", async () => {
    allowed.mockReturnValue(true);
    renderAt("/shipments");

    expect(await screen.findByText("panel /shipments preview:true")).toBeInTheDocument();
    expect(screen.queryByLabelText("Loading page access")).not.toBeInTheDocument();
  });

  it("offers no preview to someone who may not read roles", async () => {
    allowed.mockReturnValue(false);
    renderAt("/billing/queue");

    expect(await screen.findByText("panel /billing/queue preview:false")).toBeInTheDocument();
  });

  it("drops the panel once the dialog closes", async () => {
    allowed.mockReturnValue(true);
    renderAt("/shipments");
    expect(await screen.findByText("panel /shipments preview:true")).toBeInTheDocument();

    act(() => useAppDialogsStore.getState().setDialogOpen("page-access", false));

    await vi.waitFor(() => expect(screen.queryByText(/^panel /)).not.toBeInTheDocument());
  });
});

describe("PageAccessPanelSkeleton", () => {
  it("holds the panel's place while its code loads", () => {
    render(<PageAccessPanelSkeleton />);

    const skeleton = screen.getByLabelText("Loading page access");
    expect(skeleton).toHaveAttribute("aria-busy", "true");
  });
});
