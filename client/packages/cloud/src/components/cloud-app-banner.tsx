import { CloudTrialBanner } from "./plan-limit/cloud-trial-banner";
import { SupportAppBanner } from "./support/support-app-banner";

/** Every banner Trenova Cloud stacks above the app header, most urgent first. */
export function CloudAppBanner() {
  return (
    <>
      <SupportAppBanner />
      <CloudTrialBanner />
    </>
  );
}
