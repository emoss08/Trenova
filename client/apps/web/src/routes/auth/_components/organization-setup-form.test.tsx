import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { OrganizationSettings } from "@trenova/shared/types/organization";
import type { OrganizationSetup, User } from "@trenova/shared/types/user";
import type { ChangeEvent } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  OrganizationSetupStep,
  organizationSetupDefaults,
  organizationWithSetup,
} from "./organization-setup-form";

const mocks = vi.hoisted(() => ({
  getOrganizationSettingsGraphQL: vi.fn(),
  updateOrganizationSettingsGraphQL: vi.fn(),
  signOut: vi.fn(async () => {}),
  checkSession: vi.fn(),
  toastSuccess: vi.fn(),
}));

vi.mock("@/lib/graphql/organization", () => ({
  getOrganizationSettingsGraphQL: mocks.getOrganizationSettingsGraphQL,
  updateOrganizationSettingsGraphQL: mocks.updateOrganizationSettingsGraphQL,
}));

vi.mock("@/hooks/use-sign-out", () => ({
  useSignOut: () => mocks.signOut,
}));

vi.mock("@trenova/shared/components/ui/toast", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/components/ui/toast")>();
  return { ...actual, toast: { ...actual.toast, success: mocks.toastSuccess } };
});

// The state and time zone pickers read their options over the network; the step only
// needs the value each one sets.
vi.mock("@/components/autocomplete-fields", async () => {
  const { Controller } = await import("react-hook-form");
  return {
    UsStateAutocompleteField: ({ control, name }: { control: never; name: string }) => (
      <Controller
        control={control}
        name={name as never}
        render={({ field, fieldState }) => (
          <>
            <input
              aria-label="State"
              value={(field.value as string) ?? ""}
              onChange={(event: ChangeEvent<HTMLInputElement>) =>
                field.onChange(event.target.value)
              }
            />
            {fieldState.error?.message ? <p>{fieldState.error.message}</p> : null}
          </>
        )}
      />
    ),
  };
});

vi.mock("@/components/fields/timezone-field", async () => {
  const { Controller } = await import("react-hook-form");
  return {
    TimezoneField: ({ control, name }: { control: never; name: string }) => (
      <Controller
        control={control}
        name={name as never}
        render={({ field, fieldState }) => (
          <>
            <input
              aria-label="Timezone"
              value={(field.value as string) ?? ""}
              onChange={(event: ChangeEvent<HTMLInputElement>) =>
                field.onChange(event.target.value)
              }
            />
            {fieldState.error?.message ? <p>{fieldState.error.message}</p> : null}
          </>
        )}
      />
    ),
  };
});

// The organization a new instance's bootstrap made: its name, the default time zone and
// state, no street address and the bootstrap's stand-ins for the rest.
const bootstrapped: OrganizationSettings = {
  id: "org_01",
  version: 4,
  createdAt: 1794441464,
  updatedAt: 1794441464,
  businessUnitId: "bu_01",
  loginSlug: "acme-freight",
  bucketName: "",
  name: "Acme Freight",
  scacCode: "TBDX",
  dotNumber: "0",
  logoUrl: "",
  iconUrl: "",
  addressLine1: "",
  addressLine2: "",
  city: "Pending",
  stateId: "us_ny",
  postalCode: "00000",
  timezone: "America/New_York",
  taxId: "",
  brokerageEnabled: true,
  assetOperationsEnabled: true,
  state: null,
};

const missing = ["addressLine1", "city", "postalCode", "scacCode", "dotNumber"];

const administrator: OrganizationSetup = { complete: false, missing, canComplete: true };

function signedIn() {
  useAuthStore.setState({
    user: {
      id: "usr_01",
      currentOrganizationId: "org_01",
      businessUnitId: "bu_01",
      name: "Dana Whitfield",
      username: "dana",
      emailAddress: "dana@acme.example",
      timezone: "America/New_York",
      organizationSetup: administrator,
    } as User,
    isAuthenticated: true,
    checkSession: mocks.checkSession,
  });
}

function renderStep(setup: OrganizationSetup = administrator) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <OrganizationSetupStep setup={setup} />
    </QueryClientProvider>,
  );
}

async function fillMissing(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText(/^address line 1/i), "100 Main St");
  await user.type(screen.getByLabelText(/^city/i), "Chicago");
  await user.type(screen.getByLabelText(/^zip code/i), "60601");
  await user.type(screen.getByLabelText(/^scac code/i), "acme");
  await user.type(screen.getByLabelText(/^dot number/i), "1234567");
}

describe("organizationSetupDefaults", () => {
  it("leaves every field the server named as missing empty and keeps the rest", () => {
    expect(organizationSetupDefaults(bootstrapped, missing)).toEqual({
      name: "Acme Freight",
      timezone: "America/New_York",
      addressLine1: "",
      addressLine2: "",
      city: "",
      stateId: "us_ny",
      postalCode: "",
      scacCode: "",
      dotNumber: "",
    });
  });
});

describe("organizationWithSetup", () => {
  it("sends the stored organization with the answers, the SCAC in capitals", () => {
    const sent = organizationWithSetup(bootstrapped, {
      name: "Acme Freight",
      timezone: "America/Chicago",
      addressLine1: "100 Main St",
      addressLine2: "Suite 4",
      city: "Chicago",
      stateId: "us_il",
      postalCode: "60601",
      scacCode: "acme",
      dotNumber: "1234567",
    });

    expect(sent).toEqual({
      ...bootstrapped,
      timezone: "America/Chicago",
      addressLine1: "100 Main St",
      addressLine2: "Suite 4",
      city: "Chicago",
      stateId: "us_il",
      postalCode: "60601",
      scacCode: "ACME",
      dotNumber: "1234567",
    });
  });
});

describe("OrganizationSetupStep", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    signedIn();
    mocks.getOrganizationSettingsGraphQL.mockResolvedValue(bootstrapped);
    mocks.checkSession.mockResolvedValue({ status: "signed_in" });
  });

  afterEach(() => {
    cleanup();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("tells someone who may not update the organization who can, and lets them sign out", async () => {
    const user = userEvent.setup();
    renderStep({ complete: false, missing, canComplete: false });

    expect(screen.getByText("Your organization is not set up yet")).toBeInTheDocument();
    expect(screen.queryByLabelText(/^address line 1/i)).not.toBeInTheDocument();
    expect(mocks.getOrganizationSettingsGraphQL).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: /sign out/i }));
    expect(mocks.signOut).toHaveBeenCalledTimes(1);
  });

  it("shows the form's shape while the organization loads", () => {
    mocks.getOrganizationSettingsGraphQL.mockReturnValue(new Promise(() => {}));
    const { container } = renderStep();

    expect(container.querySelector('[aria-busy="true"]')).not.toBeNull();
    expect(screen.queryByRole("button", { name: /save and continue/i })).not.toBeInTheDocument();
  });

  it("says the organization could not be loaded and reads it again on request", async () => {
    const user = userEvent.setup();
    mocks.getOrganizationSettingsGraphQL.mockRejectedValueOnce(new Error("offline"));
    renderStep();

    expect(await screen.findByText("Your organization could not be loaded")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /try again/i }));

    expect(await screen.findByLabelText(/^address line 1/i)).toBeInTheDocument();
    expect(mocks.getOrganizationSettingsGraphQL).toHaveBeenCalledTimes(2);
  });

  it("opens with the stand-ins the install wrote left empty", async () => {
    renderStep();

    expect(await screen.findByLabelText(/^company name/i)).toHaveValue("Acme Freight");
    expect(screen.getByLabelText(/^city/i)).toHaveValue("");
    expect(screen.getByLabelText(/^zip code/i)).toHaveValue("");
    expect(screen.getByLabelText(/^scac code/i)).toHaveValue("");
    expect(screen.getByLabelText(/^dot number/i)).toHaveValue("");
    expect(screen.getByLabelText("State")).toHaveValue("us_ny");
  });

  it("asks for every required detail before saving", async () => {
    const user = userEvent.setup();
    renderStep();

    await user.click(await screen.findByRole("button", { name: /save and continue/i }));

    expect(await screen.findByText("Enter a street address")).toBeInTheDocument();
    expect(screen.getByText("Enter a city")).toBeInTheDocument();
    expect(screen.getByText("Enter a ZIP code")).toBeInTheDocument();
    expect(screen.getByText("A SCAC code is four letters")).toBeInTheDocument();
    expect(screen.getByText("A DOT number is 1 to 8 digits")).toBeInTheDocument();
    expect(mocks.updateOrganizationSettingsGraphQL).not.toHaveBeenCalled();
  });

  it("shows what the server refused on the field it names", async () => {
    const user = userEvent.setup();
    mocks.updateOrganizationSettingsGraphQL.mockRejectedValueOnce(
      new GraphQLRequestError({
        kind: "graphql",
        message: "validation failed",
        status: 200,
        graphQLErrors: [
          {
            message: "validation failed",
            type: "https://trenova.app/problems/validation-error",
            extensions: { type: "https://trenova.app/problems/validation-error" },
            errors: [
              {
                path: ["dotNumber"],
                message: "Organization with this DOT number already exists in this business unit",
                code: "DUPLICATE",
              },
            ],
          },
        ],
      }),
    );
    renderStep();
    await screen.findByLabelText(/^address line 1/i);

    await fillMissing(user);
    await user.click(screen.getByRole("button", { name: /save and continue/i }));

    expect(
      await screen.findByText(
        "Organization with this DOT number already exists in this business unit",
      ),
    ).toBeInTheDocument();
    expect(mocks.checkSession).not.toHaveBeenCalled();
  });

  it("saves through the organization update and checks the session again", async () => {
    const user = userEvent.setup();
    mocks.updateOrganizationSettingsGraphQL.mockImplementation(
      async (_id: string, values: OrganizationSettings) => ({ ...values, version: 5 }),
    );
    renderStep();
    await screen.findByLabelText(/^address line 1/i);

    await fillMissing(user);
    await user.click(screen.getByRole("button", { name: /save and continue/i }));

    await waitFor(() => expect(mocks.checkSession).toHaveBeenCalledTimes(1));
    expect(mocks.updateOrganizationSettingsGraphQL).toHaveBeenCalledWith("org_01", {
      ...bootstrapped,
      addressLine1: "100 Main St",
      city: "Chicago",
      postalCode: "60601",
      scacCode: "ACME",
      dotNumber: "1234567",
    });
    expect(mocks.toastSuccess).toHaveBeenCalledWith("Your organization is set up");
  });

  it("offers to continue without saving again when the session cannot be checked", async () => {
    const user = userEvent.setup();
    mocks.updateOrganizationSettingsGraphQL.mockImplementation(
      async (_id: string, values: OrganizationSettings) => ({ ...values, version: 5 }),
    );
    mocks.checkSession
      .mockResolvedValueOnce({ status: "unreachable", error: new TypeError("Failed to fetch") })
      .mockResolvedValueOnce({ status: "signed_in" });
    renderStep();
    await screen.findByLabelText(/^address line 1/i);

    await fillMissing(user);
    await user.click(screen.getByRole("button", { name: /save and continue/i }));

    expect(
      await screen.findByText(
        "Your details are saved, but Trenova could not reach the server to continue. Check your connection and try again.",
      ),
    ).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: /^continue/i }));

    await waitFor(() => expect(mocks.checkSession).toHaveBeenCalledTimes(2));
    expect(mocks.updateOrganizationSettingsGraphQL).toHaveBeenCalledTimes(1);
  });
});
