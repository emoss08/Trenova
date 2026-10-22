import { useAppDialogsStore } from "@/stores/app-dialogs-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { act, cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { PageAccessMount, PageAccessTrigger, usePageAccessShown } from "../page-access-mount";

const allowed = vi.fn<(resource: string, operation: unknown) => boolean>();
const sheet = vi.hoisted(() => {
  const state = { loads: 0, release: (() => undefined) as () => void, ready: Promise.resolve() };
  return {
    state,
    reset() {
      state.loads = 0;
      state.ready = new Promise<void>((resolve) => {
        state.release = resolve;
      });
    },
  };
});

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string, operation: unknown) => ({
    allowed: allowed(resource, operation),
    isLoading: false,
  }),
}));

vi.mock("../page-access-sheet", async () => {
  sheet.state.loads += 1;
  await sheet.state.ready;
  return { PageAccessSheet: () => <p>page access sheet</p> };
});

beforeEach(() => {
  sheet.reset();
  useAppDialogsStore.setState({ active: null });
});

afterEach(() => {
  cleanup();
  allowed.mockReset();
  vi.unstubAllEnvs();
});

/** The mount and the dialog store as a first visit reads them: the sheet not yet read. */
async function firstVisit() {
  vi.resetModules();
  const [mount, store] = await Promise.all([
    import("../page-access-mount"),
    import("@/stores/app-dialogs-store"),
  ]);
  return { ...mount, dialogs: store.useAppDialogsStore };
}

function Probe() {
  return <p>{String(usePageAccessShown())}</p>;
}

describe("usePageAccessShown", () => {
  it("offers the panel to whoever may read roles", () => {
    vi.stubEnv("DEV", false);
    allowed.mockReturnValue(true);
    render(<Probe />);

    expect(screen.getByText("true")).toBeInTheDocument();
    expect(allowed).toHaveBeenCalledWith(Resource.Role, Operation.Read);
  });

  it("hides the panel from everyone else in a production build", () => {
    vi.stubEnv("DEV", false);
    allowed.mockReturnValue(false);
    render(<Probe />);

    expect(screen.getByText("false")).toBeInTheDocument();
  });

  it("offers the panel on every screen of a development build", () => {
    vi.stubEnv("DEV", true);
    allowed.mockReturnValue(false);
    render(<Probe />);

    expect(screen.getByText("true")).toBeInTheDocument();
  });
});

describe("PageAccessTrigger", () => {
  it("reads the panel's code when pointed at and opens the panel when clicked", async () => {
    vi.stubEnv("DEV", false);
    allowed.mockReturnValue(true);
    const { PageAccessTrigger: Trigger, dialogs } = await firstVisit();
    render(<Trigger />);
    const button = screen.getByRole("button", { name: "Page access" });

    await userEvent.hover(button);
    expect(sheet.state.loads).toBe(1);

    await userEvent.click(button);
    expect(dialogs.getState().active).toBe("page-access");
    expect(sheet.state.loads).toBe(1);
  });

  it("draws nothing for a reader the panel is not offered to", () => {
    vi.stubEnv("DEV", false);
    allowed.mockReturnValue(false);
    const { container } = render(<PageAccessTrigger />);

    expect(container).toBeEmptyDOMElement();
  });
});

describe("PageAccessMount", () => {
  it("reads the sheet only when the panel first opens, drawing its skeleton meanwhile", async () => {
    vi.stubEnv("DEV", false);
    allowed.mockReturnValue(true);
    const { PageAccessMount: Mount, dialogs } = await firstVisit();
    const { container } = render(<Mount />);

    expect(container).toBeEmptyDOMElement();
    expect(sheet.state.loads).toBe(0);

    act(() => dialogs.getState().openDialog("page-access"));

    expect(await screen.findAllByLabelText("Loading page access")).not.toHaveLength(0);
    expect(sheet.state.loads).toBe(1);

    await act(async () => sheet.state.release());

    expect(await screen.findByText("page access sheet")).toBeInTheDocument();
    expect(screen.queryAllByLabelText("Loading page access")).toHaveLength(0);
  });

  it("mounts nothing for a reader the panel is not offered to, even when asked to open", () => {
    vi.stubEnv("DEV", false);
    allowed.mockReturnValue(false);
    useAppDialogsStore.setState({ active: "page-access" });
    const { container } = render(<PageAccessMount />);

    expect(container).toBeEmptyDOMElement();
  });
});
