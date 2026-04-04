import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  loginResponseSchema,
  type LoginRequest,
  type LoginResponse,
} from "@/types/user";
import { API_BASE_URL } from "@/lib/constants";

export const authService = {
  login: async (credentials: LoginRequest) => {
    const response = await api.post<LoginResponse>("/auth/login", credentials);
    return safeParse(loginResponseSchema, response, "Login Response");
  },

  logout: async () => {
    await api.post("/auth/logout");
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
