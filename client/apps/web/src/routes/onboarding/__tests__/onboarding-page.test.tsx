import { onboardingStateQueryOptions } from "@/lib/queries/onboarding";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { ChangeEvent } from "react";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  complete: vi.fn(),
  checkAuth: vi.fn(async () => true),
  fetchManifest: vi.fn(async () => ({})),
}));

vi.mock("@/services/onboarding", () => ({
  onboardingService: { get: mocks.get, complete: mocks.complete },
}));

vi.mock("@trenova/shared/hooks/use-public-config", () => ({
  usePublicConfig: () => ({
    config: {
      platformMode: "cloud",
      signupEnabled: true,
      turnstileSiteKey: "",
      termsUrl: "",
      privacyUrl: "",
      freePlan: { limits: { "shipments.total": 12, "customers.total": 8 } },
    },
    isLoading: false,
    isCloud: true,
    signupAvailable: true,
  }),
}));

// The state picker reads US states over GraphQL; the wizard only needs the id it sets.
vi.mock("@/components/autocomplete-fields", async () => {
  const { Controller } = await import("react-hook-form");
  return {
    UsStateAutocompleteField: ({
      control,
      name,
      label,
    }: {
      control: never;
      name: string;
      label: string;
    }) => (
      <Controller
        control={control}
        name={name as never}
        render={({ field }) => (
          <label>
            {label}
            <input
              value={(field.value as string) ?? ""}
              onChange={(event: ChangeEvent<HTMLInputElement>) =>
                field.onChange(event.target.value)
              }
            />
          </label>
        )}
      />
    ),
  };
});

import { OnboardingPage } from "../page";

const pendingState = {
  required: true,
  status: "pending",
  operationType: null,
  sampleDataLoaded: false,
  completedAt: null,
  organization: {
    name: "",
    timezone: "America/Chicago",
    addressLine1: "100 Main St",
    city: "Dallas",
    stateId: "us_tx",
    postalCode: "75201",
    scacCode: "",
    dotNumber: "",
  },
};

let queryClient: QueryClient;

function renderWizard() {
  queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/onboarding"]}>
        <Routes>
          <Route path="/onboarding" element={<OnboardingPage />} />
          <Route path="/" element={<p>Home screen</p>} />
          <Route path="/login" element={<p>Sign in</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("OnboardingPage", { timeout: 20_000 }, () => {
  beforeEach(() => {
    mocks.get.mockReset().mockResolvedValue(pendingState);
    mocks.complete.mockReset();
    mocks.checkAuth.mockClear();
    mocks.fetchManifest.mockClear();
    useAuthStore.setState({
      user: {
        id: "usr_1",
        currentOrganizationId: "org_1",
        memberships: [
          {
            userId: "usr_1",
            organizationId: "org_1",
            isDefault: true,
            organization: { id: "org_1", name: "Rivera Freight LLC" },
          },
        ],
      } as never,
      isAuthenticated: true,
      checkAuth: mocks.checkAuth,
    });
    usePermissionStore.setState({ fetchManifest: mocks.fetchManifest } as never);
  });

  afterEach(() => {
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("walks the steps, sends the documented payload and goes home as completed", async () => {
    mocks.complete.mockResolvedValue({ ...pendingState, status: "completed" });
    const user = userEvent.setup();
    renderWizard();

    const companyName = await screen.findByPlaceholderText("Enter company name");
    expect(companyName).toHaveValue("Rivera Freight LLC");

    await user.click(screen.getByRole("button", { name: "Continue" }));
    await user.click(await screen.findByRole("radio", { name: /freight brokerage/i }));
    await user.click(screen.getByRole("button", { name: "Continue" }));
    expect(await screen.findByText("Load sample data")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Continue" }));
    expect(await screen.findByText("Freight brokerage")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Finish setup" }));

    expect(await screen.findByText("Home screen")).toBeInTheDocument();
    expect(mocks.complete).toHaveBeenCalledWith({
      organization: {
        name: "Rivera Freight LLC",
        timezone: "America/Chicago",
        addressLine1: "100 Main St",
        city: "Dallas",
        stateId: "us_tx",
        postalCode: "75201",
      },
      operationType: "brokerage",
      loadSampleData: true,
    });
    // The protected loader reads this entry; it must say completed before the
    // navigation, or the person would be sent straight back to the wizard.
    expect(queryClient.getQueryData(onboardingStateQueryOptions("org_1").queryKey)).toMatchObject({
      status: "completed",
    });
    expect(mocks.checkAuth).toHaveBeenCalled();
    expect(mocks.fetchManifest).toHaveBeenCalled();
  });

  it("holds the company step until it is valid", async () => {
    const user = userEvent.setup();
    renderWizard();

    const companyName = await screen.findByPlaceholderText("Enter company name");
    await user.clear(companyName);
    await user.click(screen.getByRole("button", { name: "Continue" }));

    expect(await screen.findByText("Enter your company name")).toBeInTheDocument();
    expect(screen.queryByRole("radio", { name: /freight brokerage/i })).not.toBeInTheDocument();
  });

  it("returns to the step a server field error belongs to", async () => {
    const { ApiRequestError } = await import("@trenova/shared/lib/api");
    mocks.complete.mockRejectedValue(
      new ApiRequestError(422, {
        type: "https://trenova.app/problems/validation-error",
        title: "Validation Failed",
        status: 422,
        errors: [{ field: "organization.dotNumber", message: "DOT number is already in use" }],
      }),
    );
    const user = userEvent.setup();
    renderWizard();

    await screen.findByPlaceholderText("Enter company name");
    await user.click(screen.getByRole("button", { name: "Continue" }));
    await user.click(await screen.findByRole("button", { name: "Continue" }));
    await user.click(await screen.findByRole("button", { name: "Continue" }));
    await user.click(await screen.findByRole("button", { name: "Finish setup" }));

    expect(await screen.findByText("DOT number is already in use")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Enter company name")).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByText("Home screen")).not.toBeInTheDocument());
  });
});
