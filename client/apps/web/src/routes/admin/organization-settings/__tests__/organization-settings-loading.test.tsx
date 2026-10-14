import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { OrganizationSettings } from "@trenova/shared/types/organization";
import type { User } from "@trenova/shared/types/user";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import OrganizationSettingsForm from "../_components/organization-settings-form";
import { OrganizationSettingsSkeleton } from "../skeleton";

const graphql = vi.hoisted(() => ({
  getOrganizationSettingsGraphQL: vi.fn(),
  updateOrganizationSettingsGraphQL: vi.fn(),
}));

vi.mock("@/lib/graphql/organization", () => graphql);

vi.mock("@/hooks/use-optimistic-mutation", () => ({
  useOptimisticMutation: () => ({ mutateAsync: vi.fn() }),
}));

const organization: OrganizationSettings = {
  id: "org_01",
  version: 3,
  createdAt: 1_700_000_000,
  updatedAt: 1_700_000_000,
  bucketName: "org-01",
  businessUnitId: "bu_01",
  loginSlug: "acme",
  name: "Acme Freight",
  scacCode: "ACMF",
  dotNumber: "1234567",
  logoUrl: "",
  addressLine1: "1 Main St",
  addressLine2: "",
  city: "Columbus",
  stateId: "",
  postalCode: "43004",
  timezone: "America/New_York",
  taxId: "",
  brokerageEnabled: true,
  assetOperationsEnabled: true,
  state: null,
};

function signIn(currentOrganizationId: string | undefined) {
  useAuthStore.setState({
    user: { currentOrganizationId, memberships: [] } as unknown as User,
    isAuthenticated: true,
  });
}

/** The form behind the boundary the page puts around it. */
function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <NuqsTestingAdapter>
        <LazyComponent fallback={<OrganizationSettingsSkeleton />}>
          <OrganizationSettingsForm />
        </LazyComponent>
      </NuqsTestingAdapter>
    </QueryClientProvider>,
  );
}

describe("organization settings loading", () => {
  beforeEach(() => {
    vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
    graphql.getOrganizationSettingsGraphQL.mockReset();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  // While the organization loads, the page shows its own skeleton rather than a line of text
  // or an empty form that fills in under the cursor.
  it("shows the settings skeleton until the organization arrives", async () => {
    signIn("org_01");
    graphql.getOrganizationSettingsGraphQL.mockReturnValue(new Promise(() => {}));

    renderPage();

    const skeleton = await screen.findByLabelText("Loading settings");
    expect(skeleton).toHaveAttribute("aria-busy", "true");
    expect(skeleton.querySelectorAll('[data-slot="card"]')).toHaveLength(5);
    expect(screen.queryByText("Organization details")).toBeNull();
  });

  it("opens on the organization's own values once it arrives", async () => {
    signIn("org_01");
    graphql.getOrganizationSettingsGraphQL.mockResolvedValue(organization);

    renderPage();

    expect(await screen.findByText("Organization details")).toBeInTheDocument();
    expect(screen.getByDisplayValue("Acme Freight")).toBeInTheDocument();
    expect(screen.getByDisplayValue("ACMF")).toBeInTheDocument();
    expect(screen.queryByLabelText("Loading settings")).toBeNull();
    expect(graphql.getOrganizationSettingsGraphQL).toHaveBeenCalledWith(
      "org_01",
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
  });

  // A failed read used to leave an empty form that would save blanks over the organization.
  it("draws the error state in place of the form when the organization cannot be read", async () => {
    signIn("org_01");
    graphql.getOrganizationSettingsGraphQL.mockRejectedValue(new Error("upstream unavailable"));

    renderPage();

    expect(await screen.findByRole("button", { name: "Try again" })).toBeInTheDocument();
    expect(screen.queryByText("Organization details")).toBeNull();
    expect(screen.queryByRole("button", { name: "Save changes" })).toBeNull();
  });

  it("says so, without asking for an organization, when the session has none", () => {
    signIn(undefined);

    renderPage();

    expect(screen.getByText("No active organization found for this session.")).toBeInTheDocument();
    expect(graphql.getOrganizationSettingsGraphQL).not.toHaveBeenCalled();
  });
});
