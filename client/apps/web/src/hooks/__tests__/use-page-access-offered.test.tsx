import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { usePageAccessOffered } from "../use-permission";

const hasPermission = vi.fn<(resource: string, operation: number) => boolean>();

afterEach(() => {
  cleanup();
  hasPermission.mockReset();
  vi.unstubAllEnvs();
});

function Probe() {
  return <p>{String(usePageAccessOffered())}</p>;
}

function renderWith(readsRoles: boolean) {
  hasPermission.mockImplementation(
    (resource, operation) =>
      readsRoles && resource === Resource.Role && operation === Operation.Read,
  );
  usePermissionStore.setState({ hasPermission, manifest: {} } as never);
  render(<Probe />);
}

describe("usePageAccessOffered", () => {
  it("offers the panel to whoever may read roles", () => {
    vi.stubEnv("DEV", false);
    renderWith(true);

    expect(screen.getByText("true")).toBeInTheDocument();
  });

  it("hides the panel from everyone else in a production build", () => {
    vi.stubEnv("DEV", false);
    renderWith(false);

    expect(screen.getByText("false")).toBeInTheDocument();
  });

  it("offers the panel to everyone in a development build", () => {
    vi.stubEnv("DEV", true);
    renderWith(false);

    expect(screen.getByText("true")).toBeInTheDocument();
  });
});
