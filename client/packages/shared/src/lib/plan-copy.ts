import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { Operation, type OperationType, Resource } from "@trenova/shared/types/permission";

/**
 * Whether the person may change what the organization buys: the manifest lets them update the
 * organization. They are told to add an add-on themselves; everyone else is sent to them.
 */
export function canManagePlan(
  hasPermission: (resource: string, operation: OperationType) => boolean,
): boolean {
  return hasPermission(Resource.Organization, Operation.Update);
}

/**
 * "<Thing> is part of the <Add-on> add-on, which is not on your plan." The thing and the
 * add-on are names the server gave (the manifest's `unavailableFeatures`, a refusal's add-on),
 * never a feature key or an add-on code. Either may be empty: the sentence then says less
 * rather than naming nothing.
 */
export function planLockedSentence(t: TranslateFn, thing: string, addOnName: string): string {
  if (!addOnName) {
    return thing ? t("{0} is not on your plan.", thing) : t("This is not on your plan.");
  }
  return thing
    ? t("{0} is part of the {1} add-on, which is not on your plan.", thing, addOnName)
    : t("This is part of the {0} add-on, which is not on your plan.", addOnName);
}

/**
 * Who can add it. Someone who manages the organization adds it from their Trenova account
 * (no link: the client does not know where that account lives); anyone else asks them.
 */
export function planLockedGuidance(t: TranslateFn, canManagePlan: boolean): string {
  return canManagePlan
    ? t("Add it from your Trenova account.")
    : t("Ask your organization's administrator about adding it.");
}
