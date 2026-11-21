import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { TestRouter } from "@trenova/shared/testing/test-router";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import CaptureConsole from "../_components/capture-console";

const mocks = vi.hoisted(() => ({ granted: new Set<string>() }));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string, operation: number) => ({
    allowed: mocks.granted.has(`${resource}:${operation}`),
    isLoading: false,
  }),
}));

// Each tab's own content is tested on its own; the console decides where each one lives.
vi.mock("../_components/capture-settings-form", () => ({
  default: () => (
    <form aria-label="Capture settings">
      <input aria-label="Unsaved setting" />
    </form>
  ),
}));

vi.mock("../_components/profiles-panel", () => ({
  ProfilesPanel: () => <section aria-label="Scan profiles list" />,
}));

vi.mock("../_components/fleet-panel", () => ({
  FleetPanel: () => <section aria-label="Computers list" />,
}));

function grant(resource: Resource, operation: Operation = Operation.Read) {
  mocks.granted.add(`${resource}:${operation}`);
}

function grantAll() {
  grant(Resource.DocumentControl);
  grant(Resource.CaptureProfile);
  grant(Resource.CaptureDevice);
}

function renderConsole(search = "") {
  return render(
    <TestRouter search={search}>
      <CaptureConsole />
    </TestRouter>,
  );
}

describe("CaptureConsole", () => {
  beforeEach(() => {
    mocks.granted.clear();
  });

  afterEach(() => {
    cleanup();
  });

  it("offers settings, scan profiles and computers, in that order, opening settings", async () => {
    grantAll();
    renderConsole();

    expect((await screen.findAllByRole("tab")).map((tab) => tab.textContent)).toEqual([
      "Settings",
      "Scan profiles",
      "Computers",
    ]);
    expect(screen.getByRole("tab", { name: "Settings", selected: true })).toBeInTheDocument();
    expect(await screen.findByRole("form", { name: "Capture settings" })).toBeInTheDocument();
  });

  it("opens the tab the address names", async () => {
    grantAll();
    renderConsole("?tab=computers");

    expect(await screen.findByRole("tab", { name: "Computers", selected: true })).toBeVisible();
    expect(screen.getByRole("region", { name: "Computers list" })).toBeInTheDocument();
  });

  it("keeps scan profiles and computers out of the settings form", async () => {
    grantAll();
    const user = userEvent.setup();
    renderConsole();

    await user.click(await screen.findByRole("tab", { name: "Scan profiles" }));
    const profiles = await screen.findByRole("region", { name: "Scan profiles list" });
    expect(profiles.closest("form")).toBeNull();

    await user.click(screen.getByRole("tab", { name: "Computers" }));
    const computers = await screen.findByRole("region", { name: "Computers list" });
    expect(computers.closest("form")).toBeNull();
  });

  it("keeps settings not yet saved while another tab is open", async () => {
    grantAll();
    const user = userEvent.setup();
    renderConsole();

    await user.type(await screen.findByRole("textbox", { name: "Unsaved setting" }), "45");
    await user.click(screen.getByRole("tab", { name: "Scan profiles" }));
    await user.click(screen.getByRole("tab", { name: "Settings" }));

    expect(await screen.findByRole("textbox", { name: "Unsaved setting" })).toHaveValue("45");
  });

  it("shows only the tabs this person may read, opening the first", async () => {
    grant(Resource.CaptureProfile);
    grant(Resource.CaptureDevice);
    renderConsole("?tab=settings");

    expect((await screen.findAllByRole("tab")).map((tab) => tab.textContent)).toEqual([
      "Scan profiles",
      "Computers",
    ]);
    expect(screen.getByRole("tab", { name: "Scan profiles", selected: true })).toBeVisible();
    expect(screen.queryByRole("form", { name: "Capture settings" })).toBeNull();
  });

  it("draws no tab row when only one part may be read", async () => {
    grant(Resource.CaptureProfile);
    renderConsole();

    expect(await screen.findByRole("region", { name: "Scan profiles list" })).toBeInTheDocument();
    expect(screen.queryByRole("tab")).toBeNull();
  });
});
