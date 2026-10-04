import { api, setCsrfToken } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import { loginResponseSchema, type LoginResponse } from "@trenova/shared/types/user";
import {
  signupAcceptedSchema,
  type SignupRequest,
  type SignupResendRequest,
} from "@/types/cloud-signup";

const verificationsInFlight = new Map<string, Promise<LoginResponse>>();

export const cloudSignupService = {
  async signup(request: SignupRequest): Promise<void> {
    const response = await api.post<unknown>("/cloud/signups", request);
    signupAcceptedSchema.parse(response);
  },

  async resend(request: SignupResendRequest): Promise<void> {
    const response = await api.post<unknown>("/cloud/signups/resend", request);
    signupAcceptedSchema.parse(response);
  },

  /**
   * Exchanges the emailed token for a session. The response is the login response,
   * so the CSRF token is installed exactly as `authService.login` installs it.
   */
  async verify(token: string): Promise<LoginResponse> {
    const response = await api.post<LoginResponse>("/cloud/signups/verify", { token });
    const parsed = await safeParse(loginResponseSchema, response, "Signup verification");
    setCsrfToken(parsed.csrfToken);
    return parsed;
  },

  /**
   * The token is single use, and React's development double-invoke would spend it on
   * the first call and fail the second. Every caller for one token shares a single
   * request; a failed one is forgotten so the person can retry it.
   */
  verifyOnce(token: string): Promise<LoginResponse> {
    const existing = verificationsInFlight.get(token);
    if (existing) {
      return existing;
    }

    const request = cloudSignupService.verify(token);
    verificationsInFlight.set(token, request);
    request.catch(() => {
      verificationsInFlight.delete(token);
    });
    return request;
  },
};
