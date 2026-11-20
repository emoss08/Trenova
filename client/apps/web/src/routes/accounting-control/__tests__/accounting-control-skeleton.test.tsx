import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { OrganizationCapabilities } from "@trenova/shared/types/organization-capability";
import type { User } from "@trenova/shared/types/user";
import { afterEach, describe, expect, it, vi } from "vitest";
import AccountingControlForm from "../_components/accounting-control-form";
import { AccountingControlSkeleton } from "../skeleton";
import { savedAccountingControl } from "./accounting-control-fixture";

vi.mock("@/lib/queries/accounting-control", () => ({
  accountingControl: {
    get: Object.assign(
      () => ({ queryKey: ["accounting-control"], queryFn: async () => savedAccountingControl }),
      { _def: ["accounting-control"] },
    ),
  },
}));

vi.mock("@/hooks/use-optimistic-mutation", () => ({
  useOptimisticMutation: () => ({ mutateAsync: vi.fn() }),
}));

function signIn(capabilities: OrganizationCapabilities) {
  useAuthStore.setState({
    user: {
      currentOrganizationId: "org_01",
      memberships: [
        {
          userId: "usr_01",
          organizationId: "org_01",
          isDefault: true,
          organization: { id: "org_01", name: "Carrier Co", ...capabilities },
        },
      ],
    } as unknown as User,
    isAuthenticated: true,
  });
}

type Outline = { tabs: number; sections: number; rows: number };

function skeletonOutline(search = ""): Outline {
  const { container } = render(
    <TestRouter search={search}>
      <AccountingControlSkeleton />
    </TestRouter>,
  );
  return {
    tabs: container.querySelectorAll('[data-slot="tab-skeleton"]').length,
    sections: container.querySelectorAll('[data-slot="settings-section-skeleton"]').length,
    rows: container.querySelectorAll('[data-slot="field-skeleton"]').length,
  };
}

async function formOutline(search = ""): Promise<Outline> {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const { container } = render(
    <TestRouter search={search}>
      <QueryClientProvider client={queryClient}>
        <AccountingControlForm />
      </QueryClientProvider>
    </TestRouter>,
  );
  await screen.findByText("Recognition");
  const open = '[data-slot="tabs-content"]:not([hidden])';
  return {
    tabs: container.querySelectorAll('[role="tab"]').length,
    sections: container.querySelectorAll(`${open} [data-slot="settings-section"]`).length,
    rows: container.querySelectorAll(`${open} [data-slot="setting-row"]`).length,
  };
}

describe("AccountingControlSkeleton", () => {
  afterEach(() => {
    cleanup();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  // The fallback has to be the page the organization will see, or the layout jumps when
  // the form lands: driver pay is there only for a fleet.
  it("draws every tab, and Posting's sections and rows, for a fleet", async () => {
    signIn({ brokerageEnabled: false, assetOperationsEnabled: true });

    const outline = skeletonOutline();
    expect(outline).toEqual({ tabs: 5, sections: 2, rows: 7 });
    cleanup();

    expect(await formOutline()).toEqual(outline);
  });

  it("leaves out driver pay for a brokerage, as the form does", async () => {
    signIn({ brokerageEnabled: true, assetOperationsEnabled: false });

    const outline = skeletonOutline();
    expect(outline).toEqual({ tabs: 4, sections: 2, rows: 7 });
    cleanup();

    expect(await formOutline()).toEqual(outline);
  });

  it.each([
    ["accounts", { tabs: 5, sections: 3, rows: 9 }],
    ["driverPay", { tabs: 5, sections: 2, rows: 8 }],
    ["close", { tabs: 5, sections: 2, rows: 5 }],
    ["currency", { tabs: 5, sections: 1, rows: 2 }],
  ])("draws the %s tab the address opens, as the form does", async (tab, expected) => {
    signIn({ brokerageEnabled: false, assetOperationsEnabled: true });

    const outline = skeletonOutline(`?tab=${tab}`);
    expect(outline).toEqual(expected);
    cleanup();

    expect(await formOutline(`?tab=${tab}`)).toEqual(outline);
  });

  it("is announced once as loading", () => {
    signIn({ brokerageEnabled: false, assetOperationsEnabled: true });

    render(
      <TestRouter>
        <AccountingControlSkeleton />
      </TestRouter>,
    );

    const root = screen.getByLabelText("Loading settings");
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root.firstElementChild).toHaveAttribute("aria-hidden", "true");
  });
});
