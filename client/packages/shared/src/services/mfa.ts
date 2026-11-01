import {
  BeginMyTotpEnrollmentDocument,
  ConfirmMyTotpEnrollmentDocument,
  DisableMyTotpDocument,
  MyMfaStatusDocument,
  RegenerateMyRecoveryCodesDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
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

export const mfaService = {
  async status(options?: { signal?: AbortSignal }): Promise<MFAStatus> {
    const data = await requestGraphQL<{ myMfaStatus: unknown }>({
      document: MyMfaStatusDocument,
      operationName: "MyMfaStatus",
      signal: options?.signal,
    });
    return safeParse(mfaStatusSchema, data.myMfaStatus, "Two-factor status");
  },

  async beginEnrollment(password: string): Promise<TOTPEnrollment> {
    const data = await requestGraphQL<{ beginMyTotpEnrollment: unknown }>({
      document: BeginMyTotpEnrollmentDocument,
      operationName: "BeginMyTotpEnrollment",
      variables: { password },
    });
    return safeParse(totpEnrollmentSchema, data.beginMyTotpEnrollment, "Authenticator enrollment");
  },

  async confirmEnrollment(code: string): Promise<RecoveryCodes> {
    const data = await requestGraphQL<{ confirmMyTotpEnrollment: unknown }>({
      document: ConfirmMyTotpEnrollmentDocument,
      operationName: "ConfirmMyTotpEnrollment",
      variables: { code },
    });
    return safeParse(recoveryCodesSchema, data.confirmMyTotpEnrollment, "Recovery codes");
  },

  async disable(request: DisableTOTPRequest): Promise<void> {
    await requestGraphQL({
      document: DisableMyTotpDocument,
      operationName: "DisableMyTotp",
      variables: {
        input: {
          password: request.password,
          code: request.code ?? null,
          recoveryCode: request.recoveryCode ?? null,
        },
      },
    });
  },

  async regenerateRecoveryCodes(code: string): Promise<RecoveryCodes> {
    const data = await requestGraphQL<{ regenerateMyRecoveryCodes: unknown }>({
      document: RegenerateMyRecoveryCodesDocument,
      operationName: "RegenerateMyRecoveryCodes",
      variables: { code },
    });
    return safeParse(recoveryCodesSchema, data.regenerateMyRecoveryCodes, "Recovery codes");
  },
};
