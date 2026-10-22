import { useAppDialogsStore } from "@/stores/app-dialogs-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { PageAccessTrigger, usePageAccessPanel } from "../page-access-mount";

const allowed = vi.fn<(resource: string, operation: unknown) => boolean>();

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string, operation: unknown) => ({
    allowed: allowed(resource, operation),
    isLoading: false,
  }),
}));

beforeEach(() => {
  useAppDialogsStore.setState({ active: null });
});

afterEach(() => {
  cleanup();
  allowed.mockReset();
  vi.unstubAllEnvs();
});

function Probe() {
  const { shown, canPreview } = usePageAccessPanel();
  return <p>{`${shown}:${canPreview}`}</p>;
}

describe("usePageAccessPanel", () => {
  it("offers the panel to whoever may read roles, and the preview with it", () => {
    vi.stubEnv("DEV", false);
    allowed.mockReturnValue(true);
    render(<Probe />);

    expect(screen.getByText("true:true")).toBeInTheDocument();
    expect(allowed).toHaveBeenCalledWith(Resource.Role, Operation.Read);
  });

  it("hides the panel from everyone else in a production build", () => {
    vi.stubEnv("DEV", false);
    allowed.mockReturnValue(false);
    render(<Probe />);

    expect(screen.getByText("false:false")).toBeInTheDocument();
  });

  it("offers the panel without the preview on every screen of a development build", () => {
    vi.stubEnv("DEV", true);
    allowed.mockReturnValue(false);
    render(<Probe />);

    expect(screen.getByText("true:false")).toBeInTheDocument();
  });
});

describe("PageAccessTrigger", () => {
  it("opens the page access panel", async () => {
    vi.stubEnv("DEV", false);
    allowed.mockReturnValue(true);
    render(<PageAccessTrigger />);

    await userEvent.click(screen.getByRole("button", { name: "Page access" }));

    expect(useAppDialogsStore.getState().active).toBe("page-access");
  });

  it("draws nothing for a reader the panel is not offered to", () => {
    vi.stubEnv("DEV", false);
    allowed.mockReturnValue(false);
    const { container } = render(<PageAccessTrigger />);

    expect(container).toBeEmptyDOMElement();
  });
});
