import { usePublicConfig } from "@trenova/shared/hooks/use-public-config";
import { PRIVACY_URL, TERMS_URL } from "@trenova/shared/lib/constants";

export type LegalUrls = {
  termsUrl: string;
  privacyUrl: string;
};

/**
 * The server's public config names the documents a cloud signup agrees to; the build's
 * VITE_TERMS_URL / VITE_PRIVACY_URL (or their defaults) stand in when it names none.
 */
export function useLegalUrls(): LegalUrls {
  const { config } = usePublicConfig();
  return {
    termsUrl: config.termsUrl || TERMS_URL,
    privacyUrl: config.privacyUrl || PRIVACY_URL,
  };
}
