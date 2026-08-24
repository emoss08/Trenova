import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { OrganizationCapabilities } from "@trenova/shared/types/organization-capability";
import type { User } from "@trenova/shared/types/user";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ConsoleCenterPane } from "../console-center-pane";
import { ConsoleToolbar } from "../console-toolbar";
import { resolveCenterMode } from "../url-state";

vi.mock("../dispatch-timeline", () => ({
  DispatchTimeline: () => <div data-testid="dispatch-timeline" />,
}));
vi.mock("../coverage-kanban", () => ({
  CoverageKanban: () => <div data-testid="coverage-kanban" />,
}));

const hybrid: OrganizationCapabilities = {
  brokerageEnabled: true,
  assetOperationsEnabled: true,
};
const brokerageOnly: OrganizationCapabilities = {
  brokerageEnabled: true,
  assetOperationsEnabled: false,
};

function signIn(capabilities: OrganizationCapabilities) {
  useAuthStore.setState({
    user: {
      currentOrganizationId: "org_01",
      memberships: [
        {
          userId: "usr_01",
          organizationId: "org_01",
          isDefault: true,
          organization: { id: "org_01", name: "Brokerage Co", ...capabilities },
        },
      ],
    } as unknown as User,
    isAuthenticated: true,
  });
}

function renderPane(searchParams: string) {
  return render(
    <NuqsTestingAdapter searchParams={searchParams}>
      <ConsoleCenterPane
        moves={[]}
        drivers={[]}
        isLoading={false}
        range={{ start: 0, end: 86_400 }}
        zoom={1}
        selectedMoveId={null}
        selectedWorkerId={null}
        onSelectMove={vi.fn()}
        onSelectDriver={vi.fn()}
      />
    </NuqsTestingAdapter>,
  );
}

function renderToolbar() {
  return render(
    <NuqsTestingAdapter searchParams="">
      <ConsoleToolbar
        canUndo={false}
        isAssigning={false}
        isPlanning={false}
        onUndo={vi.fn()}
        onPlan={vi.fn()}
      />
    </NuqsTestingAdapter>,
  );
}

describe("resolveCenterMode", () => {
  it("keeps the driver timeline for an organization that employs drivers", () => {
    expect(resolveCenterMode("timeline", hybrid)).toBe("timeline");
    expect(resolveCenterMode("board", hybrid)).toBe("board");
  });

  it("falls back to the coverage board for a brokerage", () => {
    expect(resolveCenterMode("timeline", brokerageOnly)).toBe("board");
    expect(resolveCenterMode("board", brokerageOnly)).toBe("board");
  });
});

describe("dispatch console center view", () => {
  afterEach(() => {
    cleanup();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("mounts the driver timeline for an organization that employs drivers", async () => {
    signIn(hybrid);

    renderPane("?mode=timeline");

    expect(await screen.findByTestId("dispatch-timeline")).toBeInTheDocument();
  });

  it("leaves the driver timeline unmounted for a brokerage", async () => {
    signIn(brokerageOnly);

    renderPane("?mode=timeline");

    expect(await screen.findByTestId("coverage-kanban")).toBeInTheDocument();
    expect(screen.queryByTestId("dispatch-timeline")).toBeNull();
  });
});

describe("dispatch console centre view toggle", () => {
  afterEach(() => {
    cleanup();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("offers the choice of centre views to an organization that employs drivers", () => {
    signIn(hybrid);

    renderToolbar();

    expect(screen.getByRole("group", { name: "Center view" })).toBeInTheDocument();
  });

  it("withdraws the toggle from a brokerage rather than offering one option", () => {
    signIn(brokerageOnly);

    renderToolbar();

    expect(screen.queryByRole("group", { name: "Center view" })).toBeNull();
  });
});

describe("dispatch console auto-assign", () => {
  afterEach(() => {
    cleanup();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("offers auto-assign to an organization that employs drivers", () => {
    signIn(hybrid);

    renderToolbar();

    expect(screen.getByRole("button", { name: /auto-assign/i })).toBeInTheDocument();
  });

  it("withholds auto-assign from a brokerage, whose proposals could never execute", () => {
    signIn(brokerageOnly);

    renderToolbar();

    expect(screen.queryByRole("button", { name: /auto-assign/i })).toBeNull();
  });

  it("leaves a brokerage no control at all that reaches the plan mutation", async () => {
    signIn(brokerageOnly);
    const onPlan = vi.fn();

    render(
      <NuqsTestingAdapter searchParams="">
        <ConsoleToolbar
          canUndo
          isAssigning={false}
          isPlanning={false}
          onUndo={vi.fn()}
          onPlan={onPlan}
        />
      </NuqsTestingAdapter>,
    );

    for (const control of screen.getAllByRole("button")) {
      await userEvent.click(control);
    }

    expect(onPlan).not.toHaveBeenCalled();
  });
});
