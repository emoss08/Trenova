import { onboardingStateQueryOptions } from "@/lib/queries/onboarding";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
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
  selectOptions: vi.fn(),
}));

vi.mock("@/services/onboarding", () => ({
  onboardingService: { get: mocks.get, complete: mocks.complete },
}));

// Typed lines show whole under reduced motion, which is what keeps these tests about
// the conversation rather than about the typewriter's cadence (tested on its own).
vi.mock("motion/react", async (importOriginal) => ({
  ...(await importOriginal<typeof import("motion/react")>()),
  useReducedMotion: () => true,
}));

vi.mock("@/lib/graphql/select-options", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/select-options")>()),
  fetchGraphQLSelectOptions: mocks.selectOptions,
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

// The state picker reads US states over GraphQL; the conversation only needs the id it
// sets and the option it reports.
vi.mock("@/components/autocomplete-fields", async () => {
  const { Controller } = await import("react-hook-form");
  return {
    UsStateAutocompleteField: ({
      control,
      name,
      onOptionChange,
    }: {
      control: never;
      name: string;
      onOptionChange?: (option: unknown) => void;
    }) => (
      <Controller
        control={control}
        name={name as never}
        render={({ field }) => (
          <input
            aria-label="State"
            value={(field.value as string) ?? ""}
            onChange={(event: ChangeEvent<HTMLInputElement>) => {
              field.onChange(event.target.value);
              onOptionChange?.({
                id: event.target.value,
                label: "Oklahoma",
                description: null,
                meta: { abbreviation: "OK" },
              });
            }}
          />
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

function renderOnboarding() {
  queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/onboarding"]}>
        <Routes>
          <Route path="/onboarding" element={<OnboardingPage />} />
          <Route path="/" element={<p>Home screen</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

type User = ReturnType<typeof userEvent.setup>;

async function answerName(user: User) {
  const input = await screen.findByPlaceholderText("Company name");
  await user.click(input);
  await user.keyboard("{Enter}");
}

async function answerTimezone(user: User) {
  await screen.findByRole("group", { name: "Timezone" });
  await user.keyboard("{Enter}");
}

async function answerAddress(user: User) {
  await user.click(await screen.findByRole("button", { name: "Continue" }));
}

async function skipIds(user: User) {
  await user.click(await screen.findByRole("button", { name: "Skip for now" }));
}

async function pickOperation(user: User, name: RegExp) {
  await user.click(await screen.findByRole("button", { name }));
}

async function walkToReview(user: User) {
  await answerName(user);
  await answerTimezone(user);
  await answerAddress(user);
  await skipIds(user);
  await pickOperation(user, /^Freight brokerage/);
  await pickOperation(user, /^Load sample data/);
  return screen.findByRole("button", { name: /Looks good, finish setup/ });
}

/** Steps fake time in slices, so effects scheduling the next timer commit in between. */
async function elapse(ms: number) {
  for (let spent = 0; spent < ms; spent += 200) {
    await act(async () => {
      await vi.advanceTimersByTimeAsync(200);
    });
  }
}

function bubbles() {
  return [...document.querySelectorAll(".nv-ans-b")].map((node) => node.textContent);
}

describe("OnboardingPage", { timeout: 30_000 }, () => {
  beforeEach(() => {
    mocks.get.mockReset().mockResolvedValue(pendingState);
    mocks.complete.mockReset();
    mocks.checkAuth.mockClear();
    mocks.fetchManifest.mockClear();
    mocks.selectOptions.mockReset().mockResolvedValue({
      count: 1,
      next: "",
      previous: "",
      results: [{ id: "us_tx", label: "Texas", description: null, meta: { abbreviation: "TX" } }],
    });
    vi.spyOn(Intl.DateTimeFormat.prototype, "resolvedOptions").mockReturnValue({
      timeZone: "America/Denver",
    } as Intl.ResolvedDateTimeFormatOptions);
    useAuthStore.setState({
      user: {
        id: "usr_1",
        name: "Marcus Rivera",
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
    vi.restoreAllMocks();
    vi.useRealTimers();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("greets by first name and prefills the company name from signup", async () => {
    renderOnboarding();

    expect(
      await screen.findByText(/Hi Marcus, I'm Nova\./, { selector: ".nv-prose" }),
    ).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Company name")).toHaveValue("Rivera Freight LLC");
    expect(screen.queryByRole("button", { name: "Back" })).not.toBeInTheDocument();
  });

  it("puts the browser's zone first and selects it when the organization has none", async () => {
    mocks.get.mockResolvedValue({
      ...pendingState,
      organization: { ...pendingState.organization, timezone: "" },
    });
    const user = userEvent.setup();
    renderOnboarding();

    await answerName(user);
    expect(
      await screen.findByText(/Your browser is set to/, { selector: ".nv-prose" }),
    ).toHaveTextContent("Your browser is set to Mountain time.");
    const tiles = screen.getAllByRole("button", { pressed: true });
    expect(tiles).toHaveLength(1);
    expect(tiles[0]).toHaveTextContent("Mountain time");
    const group = screen.getByRole("group", { name: "Timezone" });
    expect(group.querySelector("button")).toHaveTextContent("Mountain time");
  });

  it("holds the name turn until a name is given", async () => {
    const user = userEvent.setup();
    renderOnboarding();

    const input = await screen.findByPlaceholderText("Company name");
    await user.clear(input);
    await user.keyboard("{Enter}");

    expect(await screen.findByRole("alert")).toHaveTextContent("Your company name is required.");
    expect(screen.queryByRole("group", { name: "Timezone" })).not.toBeInTheDocument();
  });

  it("refuses a short ZIP code and a malformed SCAC", async () => {
    const user = userEvent.setup();
    renderOnboarding();

    await answerName(user);
    await answerTimezone(user);
    const zip = await screen.findByPlaceholderText("27377");
    await user.clear(zip);
    await user.type(zip, "752{Enter}");
    expect(await screen.findByRole("alert")).toHaveTextContent("ZIP code needs 5 digits.");

    await user.type(zip, "01{Enter}");
    const scac = await screen.findByPlaceholderText("RVFL");
    await user.type(scac, "r{Enter}");
    expect(await screen.findByRole("alert")).toHaveTextContent("2–4 letters.");

    const dot = screen.getByPlaceholderText("3812045");
    await user.type(dot, "38a12");
    expect(dot).toHaveValue("3812");
  });

  it("skips both IDs at once", async () => {
    const user = userEvent.setup();
    renderOnboarding();

    await answerName(user);
    await answerTimezone(user);
    await answerAddress(user);
    await user.type(await screen.findByPlaceholderText("RVFL"), "rvfl");
    await user.click(screen.getByRole("button", { name: "Skip" }));

    await screen.findByRole("group", { name: "Operation type" });
    expect(bubbles()).toContain("Skip for now");
  });

  it("picks the operation type and sample data with number keys", async () => {
    const user = userEvent.setup();
    renderOnboarding();

    await answerName(user);
    await answerTimezone(user);
    await answerAddress(user);
    await skipIds(user);
    await screen.findByRole("group", { name: "Operation type" });
    await user.keyboard("2");
    await screen.findByRole("group", { name: "Sample data" });
    await user.keyboard("2");

    await screen.findByRole("button", { name: /Looks good, finish setup/ });
    expect(bubbles()).toEqual(expect.arrayContaining(["Freight brokerage", "Start empty"]));
  });

  it("returns to the review after editing a line from it", async () => {
    const user = userEvent.setup();
    renderOnboarding();

    await walkToReview(user);
    await user.click(screen.getByRole("button", { name: /Company name/ }));
    const input = await screen.findByPlaceholderText("Company name");
    await user.clear(input);
    await user.type(input, "Rivera Logistics{Enter}");

    expect(
      await screen.findByRole("button", { name: /Looks good, finish setup/ }),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Company name/ })).toHaveTextContent(
      "Rivera Logistics",
    );
  });

  it("walks back one turn with Escape and keeps the answers", async () => {
    const user = userEvent.setup();
    renderOnboarding();

    await answerName(user);
    await answerTimezone(user);
    await screen.findByPlaceholderText("27377");
    await user.keyboard("{Escape}");

    expect(await screen.findByRole("group", { name: "Timezone" })).toBeInTheDocument();
    expect(screen.queryByPlaceholderText("27377")).not.toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /^Central time/, pressed: true }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Back" }));
    expect(await screen.findByPlaceholderText("Company name")).toHaveValue("Rivera Freight LLC");
  });

  it("sends the documented payload, narrates the build and opens the app", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    let resolveComplete: (value: unknown) => void = () => undefined;
    mocks.complete.mockReturnValue(
      new Promise((resolve) => {
        resolveComplete = resolve;
      }),
    );
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    renderOnboarding();

    await walkToReview(user);
    expect(screen.getByRole("button", { name: /Address/ })).toHaveTextContent(
      "100 Main St, Dallas, TX 75201",
    );
    await act(async () => {
      await vi.advanceTimersByTimeAsync(50);
    });
    await user.keyboard("{Control>}{Enter}{/Control}");

    await waitFor(() => expect(mocks.complete).toHaveBeenCalledTimes(1));
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

    await elapse(10_000);
    expect(screen.getByText("Making you the owner")).toBeInTheDocument();
    expect(screen.queryByText(/You're all set/)).not.toBeInTheDocument();

    await act(async () => {
      resolveComplete({ ...pendingState, status: "completed" });
    });
    await elapse(2_000);
    expect(await screen.findByText("You're all set, Marcus")).toBeInTheDocument();
    expect(screen.getByText("11 sample records")).toBeInTheDocument();
    expect(mocks.checkAuth).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Open Trenova" }));
    expect(await screen.findByText("Home screen")).toBeInTheDocument();
    expect(queryClient.getQueryData(onboardingStateQueryOptions("org_1").queryKey)).toMatchObject({
      status: "completed",
    });
    expect(mocks.checkAuth).toHaveBeenCalled();
    expect(mocks.fetchManifest).toHaveBeenCalled();
  });

  it("rewinds to the turn a server field error belongs to", async () => {
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
    renderOnboarding();

    const finish = await walkToReview(user);
    await user.click(finish);

    expect(
      await screen.findByText(/Something went wrong while I was setting things up/, {
        selector: ".nv-prose",
      }),
    ).toBeInTheDocument();
    expect(
      await screen.findByText("DOT number is already in use", undefined, { timeout: 3_000 }),
    ).toBeInTheDocument();
    expect(screen.getByPlaceholderText("3812045")).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByText("Home screen")).not.toBeInTheDocument());
  });

  it("offers to try again when the refusal names no field", async () => {
    mocks.complete.mockRejectedValueOnce(new Error("network down"));
    const user = userEvent.setup();
    renderOnboarding();

    await user.click(await walkToReview(user));
    const retry = await screen.findByRole("button", { name: "Try again" });

    mocks.complete.mockReturnValue(new Promise(() => undefined));
    fireEvent.click(retry);
    await waitFor(() => expect(mocks.complete).toHaveBeenCalledTimes(2));
  });
});
