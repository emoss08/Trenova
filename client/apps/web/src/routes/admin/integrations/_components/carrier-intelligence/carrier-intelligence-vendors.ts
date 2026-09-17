import type { SpecIntegrationVendor } from "../shared/spec-integration-fields";

export type CarrierIntelIntegrationType = "CarrierOK" | "FMCSAQCMobile";

export type CarrierIntelVendor = SpecIntegrationVendor & {
  integrationType: CarrierIntelIntegrationType;
  secretKey: string;
  sandboxKeyPrefix?: string;
  supportsFallbackRole: boolean;
};

export const CARRIER_INTEL_ROLE_KEY = "role";
export const CARRIER_INTEL_FALLBACK_ROLE = "fallback";

export const carrierOKVendor: CarrierIntelVendor = {
  integrationType: "CarrierOK",
  name: "CarrierOk",
  logoLight: "/integrations/logos/carrierok-light.svg",
  logoDark: "/integrations/logos/carrierok-dark.svg",
  headline: "Vet and monitor carriers with CarrierOk",
  blurb: "Authority, insurance, safety, fraud signals and lanes from the",
  docsLabel: "CarrierOk developer API.",
  docsUrl: "https://developers.carrierok.com/docs",
  prerequisite:
    "CarrierOk bills per carrier looked up and per carrier monitored. Use a sandbox key (sk_test_) to try the connection against the fixture carriers before switching to a live key.",
  secretKey: "apiKey",
  sandboxKeyPrefix: "sk_test_",
  supportsFallbackRole: false,
};

export const fmcsaQCMobileVendor: CarrierIntelVendor = {
  integrationType: "FMCSAQCMobile",
  name: "FMCSA QCMobile",
  logoLight: "/integrations/logos/fmcsa-light.svg",
  logoDark: "/integrations/logos/fmcsa-dark.svg",
  headline: "Look up carriers with FMCSA QCMobile",
  blurb: "Free authority, insurance and safety data from the",
  docsLabel: "FMCSA QCMobile API.",
  docsUrl: "https://mobile.fmcsa.dot.gov/QCDevsite/",
  prerequisite:
    "Request a free web key from the FMCSA QCMobile developer site. FMCSA does not publish change feeds, so monitoring compares scheduled snapshots, and rules that need data FMCSA does not provide evaluate as unverifiable.",
  secretKey: "webKey",
  supportsFallbackRole: true,
};
