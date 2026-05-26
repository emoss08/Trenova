import { api, clearCsrfToken, setCsrfToken } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import type { RoleSummary } from "@/types/role";
import { loginResponseSchema, type LoginRequest, type LoginResponse } from "@/types/user";
import { API_BASE_URL } from "@/lib/constants";

export const authService = {
  login: async (credentials: LoginRequest) => {
    const response = await api.post<LoginResponse>("/auth/login", credentials);
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

  getSSOStartUrl: (provider: string, slug: string, returnTo: string) => {
    const url = new URL(
      `${API_BASE_URL}/auth/sso/start/${provider}/${slug}`,
      window.location.origin,
    );
    url.searchParams.set("returnTo", returnTo);
    return url.toString();
  },
};
