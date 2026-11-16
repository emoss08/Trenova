import type { SupportAccessGrant, SupportAccessScope } from "@/lib/graphql/support-access";
import type { BadgeVariant } from "@trenova/shared/components/ui/badge";
import type { TranslateFn as Translate } from "@trenova/shared/i18n/runtime";

export function scopeLabel(t: Translate, scope: SupportAccessScope): string {
  return scope === "Full" ? t("Full access") : t("Read-only");
}

export function scopeDescription(t: Translate, scope: SupportAccessScope): string {
  return scope === "Full"
    ? t("Staff can do what an administrator can, except the things Trenova support can never do.")
    : t(
        "Staff can read what an administrator can, except the things Trenova support can never see, and cannot change anything.",
      );
}

export function statusLabel(t: Translate, status: SupportAccessGrant["status"]): string {
  switch (status) {
    case "Live":
      return t("Live");
    case "Revoked":
      return t("Revoked");
    default:
      return t("Ended");
  }
}

export function statusVariant(status: SupportAccessGrant["status"]): BadgeVariant {
  switch (status) {
    case "Live":
      return "success";
    case "Revoked":
      return "danger";
    default:
      return "neutral";
  }
}

export function durationLabel(t: Translate, hours: number): string {
  return t("{0, plural, one {# hour} other {# hours}}", hours);
}
