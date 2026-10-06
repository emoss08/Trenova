import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  mfaStatusSchema,
  recoveryCodesSchema,
  totpEnrollmentSchema,
  type DisableTOTPRequest,
  type MFAStatus,
  type RecoveryCodes,
  type TOTPEnrollment,
} from "@trenova/shared/types/mfa";

const BASE = "/users/me/mfa";

export const mfaService = {
  async status(): Promise<MFAStatus> {
    const response = await api.get<MFAStatus>(`${BASE}/`);
    return safeParse(mfaStatusSchema, response, "Two-factor status");
  },

  async beginEnrollment(password: string): Promise<TOTPEnrollment> {
    const response = await api.post<TOTPEnrollment>(`${BASE}/totp/enroll/`, { password });
    return safeParse(totpEnrollmentSchema, response, "Authenticator enrollment");
  },

  async confirmEnrollment(code: string): Promise<RecoveryCodes> {
    const response = await api.post<RecoveryCodes>(`${BASE}/totp/confirm/`, { code });
    return safeParse(recoveryCodesSchema, response, "Recovery codes");
  },

  async disable(request: DisableTOTPRequest): Promise<void> {
    await api.post(`${BASE}/totp/disable/`, request);
  },

  async regenerateRecoveryCodes(code: string): Promise<RecoveryCodes> {
    const response = await api.post<RecoveryCodes>(`${BASE}/recovery-codes/`, { code });
    return safeParse(recoveryCodesSchema, response, "Recovery codes");
  },

  async resetForUser(userId: string): Promise<void> {
    await api.delete(`/users/${userId}/mfa/`);
  },
};
