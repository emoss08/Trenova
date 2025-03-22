import { http } from "@/lib/http-client";
import type {
  CheckEmailResponse,
  LoginRequest,
  LoginResponse,
  ResetPasswordResponse,
} from "@/types/auth";
import { User } from "@/types/user";

export async function checkEmail(email: User["emailAddress"]) {
  return http.post<CheckEmailResponse>("/auth/check-email/", {
    emailAddress: email,
  });
}

export async function resetPassword(email: User["emailAddress"]) {
  return http.post<ResetPasswordResponse>("/auth/reset-password/", {
    emailAddress: email,
  });
}

export async function login(request: LoginRequest) {
  return http.post<LoginResponse>("/auth/login/", request);
}

export async function validateSession() {
  return http.post<{ valid: boolean }>("/auth/validate-session/");
}

export async function logout() {
  return http.post("/auth/logout/");
}

export async function getCurrentUser() {
  return http.get<User>("/users/me/");
}
