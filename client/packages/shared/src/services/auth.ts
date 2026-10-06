import { api, clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import { authProviderSummariesSchema } from "@trenova/shared/types/iam";
import {
  isMFAChallenge,
  mfaChallengeSchema,
  type MFAChallenge,
  type VerifyMFAChallengeRequest,
} from "@trenova/shared/types/mfa";
import type { RoleSummary } from "@trenova/shared/types/role";
import {
  loginResponseSchema,
  type LoginRequest,
  type LoginResponse,
} from "@trenova/shared/types/user";
import { API_BASE_URL } from "@trenova/shared/lib/constants";

export type LoginResult = LoginResponse | MFAChallenge;

export const authService = {
  /**
   * Signs in with a password. An account with a second factor gets a challenge back
   * instead of a session; redeem it with `verifyMFA`.
   */
  login: async (credentials: LoginRequest): Promise<LoginResult> => {
    const response = await api.post<unknown>("/auth/login", credentials);
    if (isMFAChallenge(response)) {
      return safeParse(mfaChallengeSchema, response, "Sign-in challenge");
    }

    const parsed = await safeParse(loginResponseSchema, response, "Login Response");
    setCsrfToken(parsed.csrfToken);
    return parsed;
  },

  verifyMFA: async (request: VerifyMFAChallengeRequest): Promise<LoginResponse> => {
    const response = await api.post<LoginResponse>("/auth/mfa/verify", request);
    const parsed = await safeParse(loginResponseSchema, response, "Login Response");
    setCsrfToken(parsed.csrfToken);
    return parsed;
  },

  logout: async () => {
    try {
      await api.post("/auth/logout");
    } finally {
      clearCsrfToken();
    }
  },

  listAuthorizedSessionRoles: async () => {
    return api.get<{
      roleIds: string[];
      authorizedRoleIds: string[];
      authorizedRoles: RoleSummary[];
    }>("/auth/session/roles");
  },

  activateSessionRoles: async (roleIds: string[]) => {
    return api.post<{
      activeRoleIds: string[];
      authorizedRoleIds: string[];
      activeRoles: RoleSummary[];
      authorizedRoles: RoleSummary[];
      requiresRoleActivation: boolean;
    }>("/auth/session/roles/activate", { roleIds });
  },

  // Always resolves for a well-formed address, whether or not it belongs to anyone.
  // The server answers identically in every case on purpose, so the caller must not
  // read anything into success.
  forgotPassword: async (emailAddress: string) => {
    return api.post<{ message: string }>("/auth/forgot-password", { emailAddress });
  },

  resetPassword: async (token: string, newPassword: string) => {
    return api.post<{ message: string }>("/auth/reset-password", { token, newPassword });
  },

  listProviders: async (organizationSlug: string) => {
    const response = await api.get(`/auth/providers/${organizationSlug}`);
    return safeParse(authProviderSummariesSchema, response, "AuthProviderSummaries");
  },

  getSSOStartUrl: (provider: string, slug: string, returnTo: string) => {
    const url = new URL(
      `${API_BASE_URL}/auth/sso/start/${provider}/${slug}`,
      window.location.origin,
    );
    url.searchParams.set("returnTo", returnTo);
    return url.toString();
  },
};
