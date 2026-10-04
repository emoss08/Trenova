import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  getPublicConfig: vi.fn(),
  getOnboarding: vi.fn(),
}));

vi.mock("@trenova/shared/services/platform", () => ({
  platformService: { getPublicConfig: mocks.getPublicConfig },
}));

vi.mock("@/services/onboarding", () => ({
  onboardingService: { get: mocks.getOnboarding },
}));

import {
  loadOnboardingSnapshot,
  onboardingExitRedirect,
  onboardingRedirect,
  resolveOnboardingRedirect,
} from "@/lib/onboarding-gate";
import { queryClient } from "@/lib/query-client";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { SELF_HOSTED_PUBLIC_CONFIG, type PublicConfig } from "@trenova/shared/types/platform";

const cloud: Pick<PublicConfig, "platformMode"> = { platformMode: "cloud" };
const selfHosted: Pick<PublicConfig, "platformMode"> = { platformMode: "self_hosted" };
const pending = { required: true, status: "pending" as const };
const completed = { required: true, status: "completed" as const };

describe("onboardingRedirect", () => {
  it("sends a pending cloud organization to the wizard from any signed-in page", () => {
    expect(onboardingRedirect({ pathname: "/", config: cloud, state: pending })).toBe(
      "/onboarding",
    );
    expect(
      onboardingRedirect({
        pathname: "/shipment-management/shipments",
        config: cloud,
        state: pending,
      }),
    ).toBe("/onboarding");
  });

  it("never redirects the wizard to itself, with or without a trailing slash", () => {
    expect(
      onboardingRedirect({ pathname: "/onboarding", config: cloud, state: pending }),
    ).toBeNull();
    expect(
      onboardingRedirect({ pathname: "/onboarding/", config: cloud, state: pending }),
    ).toBeNull();
  });

  it("does not treat a path that merely starts with the same letters as the wizard", () => {
    expect(
      onboardingRedirect({ pathname: "/onboarding-reports", config: cloud, state: pending }),
    ).toBe("/onboarding");
  });

  it("lets a completed organization through", () => {
    expect(onboardingRedirect({ pathname: "/", config: cloud, state: completed })).toBeNull();
  });

  it("ignores a pending row the server says is not required", () => {
    expect(
      onboardingRedirect({
        pathname: "/",
        config: cloud,
        state: { required: false, status: "pending" },
      }),
    ).toBeNull();
  });

  it("never applies outside cloud mode, whatever the onboarding row says", () => {
    expect(onboardingRedirect({ pathname: "/", config: selfHosted, state: pending })).toBeNull();
    expect(onboardingRedirect({ pathname: "/", config: undefined, state: pending })).toBeNull();
  });

  it("lets the request through when the onboarding state is unknown", () => {
    expect(onboardingRedirect({ pathname: "/", config: cloud, state: null })).toBeNull();
  });
});

describe("onboardingExitRedirect", () => {
  it("keeps a pending cloud organization on the wizard", () => {
    expect(onboardingExitRedirect({ config: cloud, state: pending })).toBeNull();
  });

  it("sends everybody else home", () => {
    expect(onboardingExitRedirect({ config: cloud, state: completed })).toBe("/");
    expect(onboardingExitRedirect({ config: selfHosted, state: pending })).toBe("/");
    expect(onboardingExitRedirect({ config: cloud, state: null })).toBe("/");
  });
});

describe("loadOnboardingSnapshot", () => {
  beforeEach(() => {
    queryClient.clear();
    mocks.getPublicConfig.mockReset();
    mocks.getOnboarding.mockReset();
    useAuthStore.setState({
      user: { currentOrganizationId: "org_1" } as never,
      isAuthenticated: true,
    });
  });

  afterEach(() => {
    queryClient.clear();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("does not ask for onboarding state on a self-hosted install", async () => {
    mocks.getPublicConfig.mockResolvedValue(SELF_HOSTED_PUBLIC_CONFIG);

    const snapshot = await loadOnboardingSnapshot();

    expect(snapshot.state).toBeNull();
    expect(mocks.getOnboarding).not.toHaveBeenCalled();
  });

  it("reads the state once and answers later navigations from the cache", async () => {
    mocks.getPublicConfig.mockResolvedValue({
      ...SELF_HOSTED_PUBLIC_CONFIG,
      platformMode: "cloud",
    });
    mocks.getOnboarding.mockResolvedValue({ ...pending, operationType: null });

    expect(await resolveOnboardingRedirect("/")).toBe("/onboarding");
    expect(await resolveOnboardingRedirect("/inbox")).toBe("/onboarding");

    expect(mocks.getPublicConfig).toHaveBeenCalledTimes(1);
    expect(mocks.getOnboarding).toHaveBeenCalledTimes(1);
  });

  it("keys the state by organization, so another organization is read afresh", async () => {
    mocks.getPublicConfig.mockResolvedValue({
      ...SELF_HOSTED_PUBLIC_CONFIG,
      platformMode: "cloud",
    });
    mocks.getOnboarding.mockResolvedValueOnce(completed).mockResolvedValueOnce(pending);

    expect(await resolveOnboardingRedirect("/")).toBeNull();

    useAuthStore.setState({ user: { currentOrganizationId: "org_2" } as never });
    expect(await resolveOnboardingRedirect("/")).toBe("/onboarding");
    expect(mocks.getOnboarding).toHaveBeenCalledTimes(2);
  });

  it("fails open when the onboarding state cannot be read", async () => {
    mocks.getPublicConfig.mockResolvedValue({
      ...SELF_HOSTED_PUBLIC_CONFIG,
      platformMode: "cloud",
    });
    mocks.getOnboarding.mockRejectedValue(new Error("503"));

    expect(await resolveOnboardingRedirect("/")).toBeNull();
  });
});
