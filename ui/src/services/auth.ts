import { http } from "@/lib/http-client";
import type { UserSchema } from "@/lib/schemas/user-schema";
import type {
  CheckEmailResponse,
  LoginRequest,
  LoginResponse,
  ResetPasswordResponse,
} from "@/types/auth";

export class AuthAPI {
  async checkEmail(email: UserSchema["emailAddress"]) {
    return http.post<CheckEmailResponse>("/auth/check-email/", {
      emailAddress: email,
    });
  }
  async resetPassword(email: UserSchema["emailAddress"]) {
    return http.post<ResetPasswordResponse>("/auth/reset-password/", {
      emailAddress: email,
    });
  }

  async login(request: LoginRequest) {
    return http.post<LoginResponse>("/auth/login/", request);
  }

  async validateSession() {
    return http.post<{ valid: boolean }>("/auth/validate-session/");
  }

  async logout() {
    return http.post("/auth/logout/");
  }

  async getCurrentUser() {
    return http.get<UserSchema>("/users/me/");
  }
}
