/**
 * Trenova's own published pages: the Academy, the legal documents on the
 * marketing site and the source license. These describe Trenova itself, so
 * they are the same on every install, unlike the terms and privacy addresses
 * an install names in its public config for its own sign-in page.
 */
export const TRENOVA_LINKS = {
  academy: "https://academy.trenova.app/",
  terms: "https://trenova.app/legal/terms/",
  privacy: "https://trenova.app/legal/privacy/",
  cloudTerms: "https://trenova.app/legal/cloud-terms/",
  cloudPrivacy: "https://trenova.app/legal/cloud-privacy/",
  license: "https://github.com/emoss08/Trenova/blob/master/LICENSE",
} as const;
