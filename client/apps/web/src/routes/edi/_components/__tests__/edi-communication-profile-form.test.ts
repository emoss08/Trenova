import { describe, expect, it } from "vitest";
import {
  getProfileFormDefaults,
  toCommunicationProfileRequest,
} from "../panel/edi-communication-profile-form";

describe("AS2 communication profile form", () => {
  it("sends the separate encryption certificate and decryption key", () => {
    const values = getProfileFormDefaults(null);
    values.method = "AS2";
    values.config.localEncryptionCertificate = "-----BEGIN CERTIFICATE-----";
    values.secrets.decryptionPrivateKey = "-----BEGIN PRIVATE KEY-----";

    const request = toCommunicationProfileRequest(values, null);

    expect(request.config).toMatchObject({
      localEncryptionCertificate: "-----BEGIN CERTIFICATE-----",
    });
    expect(request.secrets).toMatchObject({
      decryptionPrivateKey: "-----BEGIN PRIVATE KEY-----",
    });
  });

  it("leaves a blank decryption key out, so the stored one is kept", () => {
    const values = getProfileFormDefaults(null);
    values.method = "AS2";

    expect(toCommunicationProfileRequest(values, null).secrets).not.toHaveProperty(
      "decryptionPrivateKey",
    );
  });
});
