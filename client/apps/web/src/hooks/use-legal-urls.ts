import { usePublicConfig } from "@trenova/shared/hooks/use-public-config";
import { PRIVACY_URL, TERMS_URL } from "@trenova/shared/lib/constants";

export type LegalUrls = {
  termsUrl: string;
  privacyUrl: string;
};

/**
 * The terms and privacy documents this install publishes. The server's public config
 * names them; the build's VITE_TERMS_URL / VITE_PRIVACY_URL stand in when it names
 * none. Either may be empty, and then there is no document to link to.
 */
export function useLegalUrls(): LegalUrls {
  const { config } = usePublicConfig();
  return {
    termsUrl: config.termsUrl || TERMS_URL,
    privacyUrl: config.privacyUrl || PRIVACY_URL,
  };
}
