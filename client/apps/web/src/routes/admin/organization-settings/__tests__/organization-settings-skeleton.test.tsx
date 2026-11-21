import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { User } from "@trenova/shared/types/user";
import { domMax, LazyMotion } from "motion/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { clearPermissions } from "@/test/permissions";
import { seedPlan } from "@/test/plan";
import OrganizationSettingsForm from "../_components/organization-settings-form";
import { OrganizationSettingsSkeleton } from "../skeleton";
import { savedOrganization } from "./organization-settings-fixture";

vi.mock("@/lib/graphql/organization", () => ({
  getOrganizationSettingsGraphQL: async () => savedOrganization,
  updateOrganizationSettingsGraphQL: vi.fn(),
}));

vi.mock("@/lib/graphql/select-options", async (importOriginal) => {
  const ohio = { id: "us_oh", value: "us_oh", label: "Ohio" };
  return {
    ...(await importOriginal<typeof import("@/lib/graphql/select-options")>()),
    fetchGraphQLSelectOptions: async () => ({ count: 1, results: [ohio], next: null }),
    fetchGraphQLSelectedOption: async () => ohio,
  };
});

vi.mock("@/hooks/use-optimistic-mutation", () => ({
  useOptimisticMutation: () => ({ mutateAsync: vi.fn() }),
}));

type Outline = { tabs: number; sections: number; rows: number };

function skeletonOutline(search = ""): Outline {
  const { container } = render(
    <TestRouter search={search}>
      <OrganizationSettingsSkeleton />
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
      <LazyMotion features={domMax}>
        <QueryClientProvider client={queryClient}>
          <OrganizationSettingsForm />
        </QueryClientProvider>
      </LazyMotion>
    </TestRouter>,
  );
  await screen.findByRole("tab", { name: "Profile" });
  const open = '[data-slot="tabs-content"]:not([hidden])';
  return {
    tabs: container.querySelectorAll('[role="tab"]').length,
    sections: container.querySelectorAll(`${open} [data-slot="settings-section"]`).length,
    rows: container.querySelectorAll(`${open} [data-slot="setting-row"]`).length,
  };
}

describe("OrganizationSettingsSkeleton", () => {
  beforeEach(() => {
    useAuthStore.setState({
      user: { currentOrganizationId: "org_01", memberships: [] } as unknown as User,
      isAuthenticated: true,
    });
    seedPlan({}, []);
  });

  afterEach(() => {
    cleanup();
    clearPermissions();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  // The fallback has to be the page the organization will see, or the layout jumps when the
  // form lands.
  it.each([
    ["", { tabs: 3, sections: 2, rows: 8 }],
    ["?tab=operations", { tabs: 3, sections: 2, rows: 6 }],
  ])("draws the tab the address %s opens, as the form does", async (search, expected) => {
    const outline = skeletonOutline(search);
    expect(outline).toEqual(expected);
    cleanup();

    expect(await formOutline(search)).toEqual(outline);
  });

  // The preview is not a row; the outline draws it as one picture in its one section.
  it("draws the branding tab's one section", async () => {
    const outline = skeletonOutline("?tab=branding");
    expect(outline).toEqual({ tabs: 3, sections: 1, rows: 1 });
    expect(document.querySelector('[data-kind="preview"]')).not.toBeNull();
    expect(document.querySelector('[data-kind="image"]')).toBeNull();
    cleanup();

    const form = await formOutline("?tab=branding");
    expect({ tabs: form.tabs, sections: form.sections }).toEqual({ tabs: 3, sections: 1 });
  });

  it("is announced once as loading", () => {
    render(
      <TestRouter>
        <OrganizationSettingsSkeleton />
      </TestRouter>,
    );

    const root = screen.getByLabelText("Loading settings");
    expect(root).toHaveAttribute("aria-busy", "true");
    expect(root.firstElementChild).toHaveAttribute("aria-hidden", "true");
  });
});
