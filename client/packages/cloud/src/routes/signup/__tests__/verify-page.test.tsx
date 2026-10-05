import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { ApiRequestError, clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import { StrictMode } from "react";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  fetchManifest: vi.fn(async () => ({})),
}));

vi.mock("@trenova/shared/hooks/use-public-config", () => ({
  usePublicConfig: () => ({
    config: {
      platformMode: "cloud",
      signupEnabled: true,
      turnstileSiteKey: "",
      termsUrl: "",
      privacyUrl: "",
      freePlan: { limits: {} },
    },
    isLoading: false,
    isCloud: true,
    signupAvailable: true,
  }),
}));

vi.mock("@/routes/auth/_components/auth-panel", () => ({ AuthPanel: () => null }));

import { cloudSignupService } from "../../../services/cloud-signup";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { SignupVerifyPage, verifyFailureOutcome } from "../verify-page";

const VERIFY_PATH = "/cloud/signups/verify";

function testUser() {
  return {
    id: "usr_1",
    version: 1,
    createdAt: 1,
    updatedAt: 1,
    businessUnitId: "bu_1",
    currentOrganizationId: "org_1",
    status: "Active",
    name: "Jordan Rivera",
    username: "jordan",
    emailAddress: "jordan@example.com",
    profilePicUrl: "",
    thumbnailUrl: "",
    timezone: "America/Chicago",
    timeFormat: "12-hour",
    isLocked: false,
    mustChangePassword: false,
  };
}

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

let verifyResponses: Array<() => Response>;
function requestUrl(input: RequestInfo | URL): string {
  if (typeof input === "string") return input;
  return input instanceof URL ? input.href : input.url;
}

const fetchMock = vi.fn(async (input: RequestInfo | URL, _init?: RequestInit) => {
  const url = requestUrl(input);
  if (!url.includes(VERIFY_PATH)) {
    throw new Error(`unexpected request ${url}`);
  }
  const next = verifyResponses.shift();
  if (!next) {
    throw new Error("no verify response queued");
  }
  return next();
});

function verifyCalls() {
  return fetchMock.mock.calls.filter(([input]) => requestUrl(input).includes(VERIFY_PATH));
}

beforeEach(() => {
  fetchMock.mockClear();
  verifyResponses = [];
  setCsrfToken("csrf-bootstrap");
  vi.stubGlobal("fetch", fetchMock);
  mocks.fetchManifest.mockClear();
  usePermissionStore.setState({ fetchManifest: mocks.fetchManifest } as never);
  useAuthStore.setState({ user: null, isAuthenticated: false });
});

afterEach(() => {
  vi.unstubAllGlobals();
  clearCsrfToken();
  useAuthStore.setState({ user: null, isAuthenticated: false });
});

function loginResponse() {
  return {
    user: testUser(),
    sessionId: "ses_1",
    expiresAt: 1,
    csrfToken: "csrf",
    activeRoleIds: [],
    authorizedRoleIds: [],
    activeRoles: [],
    authorizedRoles: [],
    requiresRoleActivation: false,
  };
}

function renderAt(url: string) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <StrictMode>
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[url]}>
          <Routes>
            <Route path="/signup/verify" element={<SignupVerifyPage />} />
            <Route path="/onboarding" element={<p>Onboarding wizard</p>} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>
    </StrictMode>,
  );
}

describe("verifyFailureOutcome", () => {
  it("puts a paused signup on the wait list", () => {
    expect(
      verifyFailureOutcome(
        new ApiRequestError(403, {
          type: "https://trenova.app/problems/plan-restricted",
          title: "Plan Restricted",
          status: 403,
          code: "PLAN_RESTRICTED",
          params: { reason: "signups_paused", capability: "", plan: "" },
        }),
      ),
    ).toEqual({ status: "signups-paused" });
  });

  it("treats a rejected, gone or unknown token as expired", () => {
    for (const status of [400, 404, 410, 422]) {
      expect(
        verifyFailureOutcome(
          new ApiRequestError(status, { type: "validation-error", title: "Bad", status }),
        ),
      ).toEqual({ status: "invalid-token" });
    }
  });

  it("keeps the wait the server asked for on a rate limit", () => {
    expect(
      verifyFailureOutcome(
        new ApiRequestError(429, { type: "rate-limit-exceeded", title: "Slow", status: 429 }, 90),
      ),
    ).toEqual({ status: "rate-limited", retryAfter: 90 });
  });

  it("calls a server or network failure retryable rather than an expired link", () => {
    expect(
      verifyFailureOutcome(
        new ApiRequestError(503, { type: "internal-error", title: "x", status: 503 }),
      ).status,
    ).toBe("failed");
    expect(verifyFailureOutcome(new TypeError("Failed to fetch")).status).toBe("failed");
  });
});

describe("cloudSignupService.verifyOnce", () => {
  it("shares one request between callers for the same token", async () => {
    verifyResponses.push(() => jsonResponse(loginResponse()));

    const [first, second] = await Promise.all([
      cloudSignupService.verifyOnce("tok-shared"),
      cloudSignupService.verifyOnce("tok-shared"),
    ]);

    expect(first).toBe(second);
    expect(verifyCalls()).toHaveLength(1);
    expect(JSON.parse(verifyCalls()[0][1]?.body as string)).toEqual({ token: "tok-shared" });
  });

  it("forgets a failed attempt so the person can retry", async () => {
    verifyResponses.push(
      () => jsonResponse({ type: "internal-error", title: "Unavailable", status: 503 }, 503),
      () => jsonResponse(loginResponse()),
    );

    await expect(cloudSignupService.verifyOnce("tok-retry")).rejects.toThrow();
    await expect(cloudSignupService.verifyOnce("tok-retry")).resolves.toMatchObject({
      sessionId: "ses_1",
    });
    expect(verifyCalls()).toHaveLength(2);
  });
});

describe("SignupVerifyPage", () => {
  it("spends the token once under StrictMode, signs the owner in and opens the wizard", async () => {
    verifyResponses.push(() => jsonResponse(loginResponse()));

    renderAt("/signup/verify?token=tok-strict");

    expect(await screen.findByText("Onboarding wizard")).toBeInTheDocument();
    expect(verifyCalls()).toHaveLength(1);
    expect(useAuthStore.getState().user?.id).toBe("usr_1");
    expect(useAuthStore.getState().isAuthenticated).toBe(true);
    expect(mocks.fetchManifest).toHaveBeenCalled();
  });

  it("offers a new link when the token has expired", async () => {
    verifyResponses.push(() =>
      jsonResponse({ type: "resource-not-found", title: "Gone", status: 410 }, 410),
    );

    renderAt("/signup/verify?token=tok-expired");

    expect(await screen.findByText("This link has expired")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Send a new link" })).toBeInTheDocument();
    expect(useAuthStore.getState().user).toBeNull();
  });

  it("tells a paused signup it is on the wait list", async () => {
    verifyResponses.push(() =>
      jsonResponse(
        {
          type: "plan-restricted",
          title: "Plan Restricted",
          status: 403,
          code: "PLAN_RESTRICTED",
          params: { reason: "signups_paused", capability: "", plan: "" },
        },
        403,
      ),
    );

    renderAt("/signup/verify?token=tok-paused");

    expect(await screen.findByText("You're on the wait list")).toBeInTheDocument();
  });

  it("does not call the server without a token", async () => {
    renderAt("/signup/verify");

    expect(await screen.findByText("This link is incomplete")).toBeInTheDocument();
    await waitFor(() => expect(verifyCalls()).toHaveLength(0));
  });
});
