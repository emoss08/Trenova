import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  loginResponseSchema,
  type LoginRequest,
  type LoginResponse,
} from "@/types/user";

export const authService = {
  login: async (credentials: LoginRequest) => {
    const response = await api.post<LoginResponse>("/auth/login", credentials);
    return safeParse(loginResponseSchema, response, "Login Response");
  },

  logout: async () => {
    await api.post("/auth/logout");
  },
};
