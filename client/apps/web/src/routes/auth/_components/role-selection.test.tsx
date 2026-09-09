import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { RoleSelection } from "./role-selection";

const mocks = vi.hoisted(() => ({
  activateSessionRoles: vi.fn(),
  fetchManifest: vi.fn(),
  onActivated: vi.fn(),
}));

vi.mock("@trenova/shared/services/auth", () => ({
  authService: { activateSessionRoles: mocks.activateSessionRoles },
}));

vi.mock("@trenova/shared/stores/permission-store", () => ({
  usePermissionStore: (
    selector: (state: { fetchManifest: typeof mocks.fetchManifest }) => unknown,
  ) => selector({ fetchManifest: mocks.fetchManifest }),
}));

const roles = [
  {
    id: "rol_admin",
    name: "Organization Administrator",
    description: "Full access to every resource",
    isSystem: true,
    permissionCount: 214,
  },
  {
    id: "rol_dispatch",
    name: "Dispatch Supervisor",
    description: "Loads, lanes, driver availability",
    isSystem: false,
    permissionCount: 68,
  },
];

describe("RoleSelection", () => {
  afterEach(() => {
    cleanup();
  });

  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockClear());
    mocks.activateSessionRoles.mockResolvedValue({});
    mocks.fetchManifest.mockResolvedValue({});
    mocks.onActivated.mockResolvedValue(undefined);
  });

  // The tally splits the number away from the noun so it can animate, so the assertion
  // has to read the whole crumb rather than a single text node.
  const crumb = (text: string) =>
    screen.getByText((_content, element) => element?.textContent === text);

  it("totals the permissions of the selected roles", async () => {
    const user = userEvent.setup();
    render(<RoleSelection roles={roles} stepLabel="03 / 03" onActivated={mocks.onActivated} />);

    expect(crumb("0 permissions")).toBeInTheDocument();

    await user.click(screen.getByRole("checkbox", { name: /organization administrator/i }));
    await waitFor(() => expect(crumb("214 permissions")).toBeInTheDocument());

    await user.click(screen.getByRole("checkbox", { name: /dispatch supervisor/i }));
    await waitFor(() => expect(crumb("282 permissions")).toBeInTheDocument());
  });

  it("falls back to a selection count when the server sent no permission counts", async () => {
    const user = userEvent.setup();
    const countless = roles.map(({ permissionCount: _permissionCount, ...role }) => role);
    render(<RoleSelection roles={countless} stepLabel="03 / 03" onActivated={mocks.onActivated} />);

    expect(screen.getByText("0 of 2 selected")).toBeInTheDocument();
    expect(screen.queryByText(/NaN/)).not.toBeInTheDocument();

    await user.click(screen.getByRole("checkbox", { name: /organization administrator/i }));
    expect(screen.getByText("1 of 2 selected")).toBeInTheDocument();
  });

  it("blocks activation until at least one role is chosen", async () => {
    const user = userEvent.setup();
    render(<RoleSelection roles={roles} stepLabel="03 / 03" onActivated={mocks.onActivated} />);

    const button = screen.getByRole("button", { name: "Select at least one role" });
    expect(button).toBeDisabled();

    await user.click(screen.getByRole("checkbox", { name: /dispatch supervisor/i }));
    expect(screen.getByRole("button", { name: "Activate 1 role" })).toBeEnabled();
  });

  it("preselects the only authorized role", () => {
    render(
      <RoleSelection roles={[roles[0]]} stepLabel="02 / 02" onActivated={mocks.onActivated} />,
    );

    expect(screen.getByRole("checkbox", { name: /organization administrator/i })).toHaveAttribute(
      "aria-checked",
      "true",
    );
    expect(screen.getByRole("button", { name: "Activate 1 role" })).toBeEnabled();
  });

  it("activates the session roles and refreshes the manifest before reporting", async () => {
    const user = userEvent.setup();
    render(<RoleSelection roles={roles} stepLabel="03 / 03" onActivated={mocks.onActivated} />);

    await user.click(screen.getByRole("checkbox", { name: /dispatch supervisor/i }));
    await user.click(screen.getByRole("button", { name: "Activate 1 role" }));

    await waitFor(() => expect(mocks.activateSessionRoles).toHaveBeenCalledWith(["rol_dispatch"]));
    expect(mocks.fetchManifest).toHaveBeenCalledTimes(1);
    expect(mocks.onActivated).toHaveBeenCalledWith([
      expect.objectContaining({ id: "rol_dispatch" }),
    ]);
  });
});
