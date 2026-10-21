import { translate } from "@trenova/shared/i18n/runtime";

export function quarantinedUploadMessage(threatName?: string | null): string {
  const name = threatName?.trim();
  return name
    ? translate("Blocked: the virus scanner found {0} in this file", name)
    : translate("Blocked: the virus scanner found a virus in this file");
}
