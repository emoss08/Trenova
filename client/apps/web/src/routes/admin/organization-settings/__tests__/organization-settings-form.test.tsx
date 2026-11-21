import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { OrganizationSettings } from "@trenova/shared/types/organization";
import type { User } from "@trenova/shared/types/user";
import { domMax, LazyMotion, MotionGlobalConfig } from "motion/react";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { clearPermissions } from "@/test/permissions";
import { seedPlan } from "@/test/plan";
import OrganizationSettingsForm from "../_components/organization-settings-form";
import { OrganizationSettingsSkeleton } from "../skeleton";
import { savedOrganization } from "./organization-settings-fixture";

const served = vi.hoisted(() => ({
  organization: null as unknown,
  read: null as null | (() => Promise<unknown>),
  saves: [] as unknown[],
  updates: [] as unknown[][],
  refusal: null as unknown,
  removed: null as unknown,
}));

vi.mock("@/lib/graphql/organization", () => ({
  getOrganizationSettingsGraphQL: vi.fn(async () =>
    served.read ? served.read() : served.organization,
  ),
  updateOrganizationSettingsGraphQL: vi.fn(async (...args: unknown[]) => {
    served.updates.push(args);
    return args[1];
  }),
}));

vi.mock("@/services/organization", () => ({
  organizationService: {
    getBranding: async () => ({ logoUrl: "https://files.test/logo.png", iconUrl: "" }),
    uploadBrandImage: vi.fn(),
    deleteBrandImage: vi.fn(async () => served.removed),
  },
}));

// The state picker reads its options over the network; Ohio is the one the record names.
vi.mock("@/lib/graphql/select-options", async (importOriginal) => {
  const ohio = { id: "us_oh", value: "us_oh", label: "Ohio" };
  return {
    ...(await importOriginal<typeof import("@/lib/graphql/select-options")>()),
    fetchGraphQLSelectOptions: async () => ({ count: 1, results: [ohio], next: null }),
    fetchGraphQLSelectedOption: async () => ohio,
  };
});

// The save goes to the server through the page's own `save`, and a refusal reaches the form
// as the real mutation hands it over.
vi.mock("@/hooks/use-optimistic-mutation", () => ({
  useOptimisticMutation: ({
    form,
    mutationFn,
  }: {
    form: unknown;
    mutationFn: (values: unknown) => Promise<unknown>;
  }) => ({
    mutateAsync: async (values: unknown) => {
      served.saves.push(values);
      if (!served.refusal) return mutationFn(values);
      const { handleMutationError } = await vi.importActual<
        typeof import("@/hooks/use-api-mutation")
      >("@/hooks/use-api-mutation");
      handleMutationError({ error: served.refusal, form: form as never });
      throw served.refusal;
    },
  }),
}));

function signIn(currentOrganizationId: string | null = "org_01") {
  useAuthStore.setState({
    user: { currentOrganizationId, memberships: [] } as unknown as User,
    isAuthenticated: true,
  });
}

/** The form behind the boundary the page puts around it. */
function renderPage(search = "") {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <TestRouter search={search}>
      <LazyMotion features={domMax}>
        <QueryClientProvider client={queryClient}>
          <LazyComponent fallback={<OrganizationSettingsSkeleton />}>
            <OrganizationSettingsForm />
          </LazyComponent>
        </QueryClientProvider>
      </LazyMotion>
    </TestRouter>,
  );
}

async function refusedBy(path: string, message: string) {
  const { ApiRequestError } = await import("@trenova/shared/lib/api");
  return new ApiRequestError(400, {
    type: "https://api.trenova.app/problems/validation-error",
    title: "Validation Failed",
    status: 400,
    detail: message,
    instance: "/graphql",
    code: "INVALID",
    errors: [{ path: [path], message, code: "DUPLICATE", location: "body" }],
  } as ConstructorParameters<typeof ApiRequestError>[1]);
}

function sectionNamed(name: string) {
  return screen.getByRole("heading", { name }).closest("section")!;
}

function rowLabels(section: HTMLElement) {
  return [...section.querySelectorAll('[data-slot="setting-row"]')].map((row) =>
    row.querySelector("label, [data-slot='field-label']")?.textContent?.replace("*", ""),
  );
}

beforeAll(() => {
  MotionGlobalConfig.skipAnimations = true;
});
afterAll(() => {
  MotionGlobalConfig.skipAnimations = false;
});

beforeEach(() => {
  served.organization = savedOrganization;
  seedPlan({}, []);
});

afterEach(() => {
  served.read = null;
  served.saves = [];
  served.updates = [];
  served.refusal = null;
  served.removed = null;
  cleanup();
  clearPermissions();
  vi.restoreAllMocks();
  useAuthStore.setState({ user: null, isAuthenticated: false });
});

describe("loading", () => {
  // While the organization loads, the page shows its own outline rather than a line of text
  // or an empty form that fills in under the cursor.
  it("shows the settings skeleton until the organization arrives", async () => {
    signIn();
    served.read = () => new Promise(() => {});

    renderPage();

    const skeleton = await screen.findByLabelText("Loading settings");
    expect(skeleton).toHaveAttribute("aria-busy", "true");
    expect(screen.queryByRole("heading", { name: "Registered address" })).toBeNull();
  });

  // A failed read used to leave an empty form that would save blanks over the organization.
  it("draws the error state in place of the form when the organization cannot be read", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {});
    signIn();
    served.read = () => Promise.reject(new Error("upstream unavailable"));

    renderPage();

    expect(await screen.findByRole("button", { name: "Try again" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Save changes" })).toBeNull();
  });

  it("says so, without asking for an organization, when the session has none", async () => {
    signIn(null);
    const { getOrganizationSettingsGraphQL } = await import("@/lib/graphql/organization");

    renderPage();

    expect(
      await screen.findByText("No active organization found for this session."),
    ).toBeInTheDocument();
    expect(getOrganizationSettingsGraphQL).not.toHaveBeenCalled();
  });
});

describe("tabs", () => {
  it("offers the profile, the branding and the operations, and no security tab", async () => {
    signIn();

    renderPage();
    await screen.findByRole("heading", { name: "Registered address" });

    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual([
      "Profile",
      "Branding",
      "Operations",
    ]);
  });
});

describe("profile", () => {
  it("draws the name, timezone and sign-in address, then the registered address, one to a row", async () => {
    signIn();

    renderPage();
    await screen.findByDisplayValue("Acme Freight");

    expect(rowLabels(sectionNamed("Profile"))).toEqual(["Name", "Timezone", "Sign-in address"]);
    expect(rowLabels(sectionNamed("Registered address"))).toEqual([
      "Address line 1",
      "Suite or unit",
      "City",
      "State",
      "ZIP code",
    ]);
    expect(screen.getByDisplayValue("1 Main St")).toBeInTheDocument();
    expect(screen.getByText("Your people sign in at /login/acme.")).toBeInTheDocument();
  });
});

describe("operations", () => {
  it("draws the operating model and the regulatory numbers", async () => {
    signIn();

    renderPage("?tab=operations");
    await screen.findByDisplayValue("ACMF");

    expect(rowLabels(sectionNamed("Operating model"))).toEqual([
      "Preset",
      "Brokerage features",
      "Asset operations",
    ]);
    expect(rowLabels(sectionNamed("Regulatory"))).toEqual(["SCAC code", "DOT number", "Tax ID"]);
  });

  it("sets both switches from a preset", async () => {
    signIn();
    const user = userEvent.setup();

    renderPage("?tab=operations");
    await screen.findByDisplayValue("ACMF");
    await user.click(screen.getByRole("radio", { name: "Asset" }));

    expect(screen.getByRole("switch", { name: /Brokerage features/ })).not.toBeChecked();
    expect(screen.getByRole("switch", { name: /Asset operations/ })).toBeChecked();
  });

  // Without brokerage on the plan the server keeps the stored value whatever is sent, so
  // neither the switch nor a preset that would turn it on is offered.
  it("offers only asset operations when the plan leaves out brokerage", async () => {
    signIn();
    seedPlan({}, [
      {
        feature: "tms.carrier_brokerage",
        featureName: "Carrier brokerage",
        addOn: "BROKER",
        addOnName: "Brokerage",
      },
    ]);

    renderPage("?tab=operations");
    await screen.findByDisplayValue("ACMF");

    expect(rowLabels(sectionNamed("Operating model"))).toEqual(["Asset operations"]);
  });
});

describe("branding", () => {
  it("draws the live preview, whose logo and icon are the controls that change them", async () => {
    signIn();

    renderPage("?tab=branding");
    await screen.findByRole("heading", { name: "Logo and icon" });

    expect(screen.getAllByRole("button", { name: "Upload icon" }).length).toBeGreaterThan(0);
    await userEvent.setup().click(screen.getByRole("radio", { name: "Invoice" }));
    expect(screen.getByRole("button", { name: "Upload logo" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Save changes" })).toBeNull();
  });

  // Removing a picture saves at once and moves the organization's version on. A name typed
  // and not yet saved stays, and the next save carries the new version, so it is not
  // refused as out of date.
  it("keeps unsaved edits and takes the new version when a picture is removed", async () => {
    served.organization = { ...savedOrganization, logoUrl: "org_01/logo.png" };
    served.removed = { ...savedOrganization, logoUrl: "", version: 4 };
    signIn();
    const user = userEvent.setup();

    renderPage();
    const name = await screen.findByDisplayValue("Acme Freight");
    await user.clear(name);
    await user.type(name, "Acme Logistics");

    await user.click(screen.getByRole("tab", { name: "Branding" }));
    await user.click(await screen.findByRole("radio", { name: "Invoice" }));
    await user.click(await screen.findByRole("button", { name: "Change logo" }));
    await user.click(await screen.findByRole("menuitem", { name: "Remove" }));
    expect(await screen.findByRole("button", { name: "Upload logo" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(served.updates).toHaveLength(1));
    expect(served.updates[0]![1]).toMatchObject({
      name: "Acme Logistics",
      version: 4,
      logoUrl: "",
    });
  });
});

describe("saving", () => {
  it("sends the organization with every tab's changes, once", async () => {
    signIn();
    const user = userEvent.setup();

    renderPage();
    const name = await screen.findByDisplayValue("Acme Freight");
    await user.clear(name);
    await user.type(name, "Acme Logistics");
    await user.click(screen.getByRole("tab", { name: "Operations" }));
    const taxId = await screen.findByDisplayValue("12-3456789");
    await user.clear(taxId);
    await user.type(taxId, "98-7654321");
    await user.click(screen.getByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(served.updates).toHaveLength(1));
    const expected: OrganizationSettings = {
      ...savedOrganization,
      name: "Acme Logistics",
      taxId: "98-7654321",
    };
    expect(served.updates[0]).toEqual(["org_01", expected]);
  });

  it("opens and marks Operations when the server refuses its DOT number", async () => {
    served.refusal = await refusedBy(
      "dotNumber",
      "Organization with this DOT number already exists in this business unit",
    );
    signIn();
    const user = userEvent.setup();

    renderPage();
    const name = await screen.findByDisplayValue("Acme Freight");
    await user.type(name, " Co");
    await user.click(screen.getByRole("button", { name: "Save changes" }));

    const operations = await screen.findByRole("tab", { name: /Operations/, selected: true });
    expect(within(operations).getByText("Has fields to fix")).toBeInTheDocument();
    expect(
      screen.getAllByText("Organization with this DOT number already exists in this business unit")
        .length,
    ).toBeGreaterThan(0);
  });

  // The server refuses an organization that neither brokers nor runs assets; the form says
  // so first, on the tab that holds the switches, without sending anything.
  it("refuses to save an organization with neither operating capability", async () => {
    signIn();
    const user = userEvent.setup();

    renderPage("?tab=operations");
    await screen.findByDisplayValue("ACMF");
    await user.click(screen.getByRole("switch", { name: /Brokerage features/ }));
    await user.click(screen.getByRole("switch", { name: /Asset operations/ }));
    await user.click(screen.getByRole("tab", { name: "Profile" }));
    await user.click(screen.getByRole("button", { name: "Save changes" }));

    expect(
      await screen.findByRole("tab", { name: /Operations/, selected: true }),
    ).toBeInTheDocument();
    expect(
      screen.getAllByText(
        "Turn on brokerage or asset operations. An organization with neither cannot move freight.",
      ).length,
    ).toBeGreaterThan(0);
    expect(served.saves).toEqual([]);
  });
});
