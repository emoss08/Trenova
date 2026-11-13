import type { HomeLayout } from "@/lib/graphql/home-layout";
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Home } from "../page";

const widget = {
  id: "widget_1",
  key: "favorites",
  title: null,
  w: 4,
  h: 4,
  config: {},
} as unknown as HomeLayout["widgets"][number];

const baseLayout = {
  schemaVersion: 1,
  version: 3,
  source: "BUILT_IN",
  presetId: null,
  presetName: "Operations home",
  locked: false,
  canCustomize: true,
  density: "comfortable",
  widgets: [widget],
} as HomeLayout;

let layout = baseLayout;
const updateLayout = vi.fn(() => Promise.resolve(baseLayout));
const resetLayout = vi.fn(() => Promise.resolve(baseLayout));

vi.mock("@/hooks/use-home-layout", () => ({
  useHomeLayout: () => ({ data: layout, isLoading: false }),
  useHomeWidgetCatalog: () => ({ data: undefined, isLoading: false }),
  useUpdateHomeLayout: () => ({ mutateAsync: updateLayout, isPending: false }),
  useResetHomeLayout: () => ({ mutateAsync: resetLayout, isPending: false }),
}));

vi.mock("../_components/use-home-data", () => ({
  useHomeData: () => ({
    attention: undefined,
    attentionLoading: false,
    shipmentAnalytics: {},
    shipmentAnalyticsLoading: false,
    shipmentAnalyticsReady: false,
  }),
}));

vi.mock("../_components/home-brief", () => ({
  HomeBrief: () => <h2>Good afternoon, Dana</h2>,
}));

vi.mock("../_components/home-canvas", () => ({
  HomeCanvas: ({ editing }: { editing: boolean }) => (
    <div data-testid="canvas" data-editing={String(editing)} />
  ),
}));

vi.mock("@/components/navigation/sidebar-layout", () => ({
  PageLayout: ({
    pageHeaderProps,
    children,
  }: {
    pageHeaderProps: { title: string; description: string; actions?: React.ReactNode };
    children: React.ReactNode;
  }) => (
    <>
      <header>
        <h1>{pageHeaderProps.title}</h1>
        <p>{pageHeaderProps.description}</p>
        {pageHeaderProps.actions}
      </header>
      <main>{children}</main>
    </>
  ),
}));

beforeEach(() => {
  layout = baseLayout;
  updateLayout.mockClear();
  resetLayout.mockClear();
});

afterEach(cleanup);

describe("Home", () => {
  it("names itself Home in the page bar and greets the person in the body, not the bar", () => {
    render(<Home />);

    const bar = screen.getByRole("banner");
    expect(within(bar).getByRole("heading", { level: 1 })).toHaveTextContent(/^Home$/);
    expect(bar).not.toHaveTextContent("Good afternoon");
    expect(
      within(screen.getByRole("main")).getByRole("heading", { name: "Good afternoon, Dana" }),
    ).toBeInTheDocument();
  });

  it("keeps Customize in the page bar, where it turns the canvas editable and becomes Done", () => {
    render(<Home />);

    const bar = screen.getByRole("banner");
    expect(screen.getByTestId("canvas")).toHaveAttribute("data-editing", "false");

    fireEvent.click(within(bar).getByRole("button", { name: "Customize" }));

    expect(screen.getByTestId("canvas")).toHaveAttribute("data-editing", "true");
    expect(within(bar).getByRole("button", { name: "Done" })).toBeEnabled();
  });

  it("does not save a home screen nobody changed when Done is pressed", () => {
    render(<Home />);

    fireEvent.click(screen.getByRole("button", { name: "Customize" }));
    fireEvent.click(screen.getByRole("button", { name: "Done" }));

    expect(updateLayout).not.toHaveBeenCalled();
    expect(screen.getByTestId("canvas")).toHaveAttribute("data-editing", "false");
  });

  it("leaves Customize in place but disabled when an administrator locked the home screen", () => {
    layout = { ...baseLayout, locked: true, canCustomize: false };
    render(<Home />);

    expect(screen.getByRole("button", { name: "Customize" })).toBeDisabled();
  });

  it("says what a personal layout was customized from and offers the way back", () => {
    layout = { ...baseLayout, source: "USER" };
    render(<Home />);

    expect(screen.getByText("Customized from Operations home")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Reset" }));
    expect(resetLayout).toHaveBeenCalledTimes(1);
  });
});
