import type { WorkerChecklistItemRow } from "@/lib/graphql/worker-checklist";
import {
  AUTO_SATISFIED_ITEM_KINDS,
  type ChecklistItemKind,
  type ChecklistOwner,
} from "@trenova/shared/types/worker-checklist";
import {
  ClipboardCheckIcon,
  File06Icon,
  type IconComponent,
  IdCardIcon,
  PackageIcon,
  Phone01Icon,
} from "@trenova/shared/components/icons";

export const CHECKLIST_ITEM_KIND_ICONS: Record<ChecklistItemKind, IconComponent> = {
  Document: File06Icon,
  Credential: IdCardIcon,
  Task: ClipboardCheckIcon,
  Equipment: PackageIcon,
  PortalAccess: Phone01Icon,
};

export function isAutoSatisfied(kind: string): boolean {
  return AUTO_SATISFIED_ITEM_KINDS.has(kind as ChecklistItemKind);
}

export type OwnerGroup = { owner: ChecklistOwner; items: WorkerChecklistItemRow[] };

/** Groups items by owner in order of first appearance, keeping item order inside. */
export function groupByOwner(items: readonly WorkerChecklistItemRow[]): OwnerGroup[] {
  const groups: OwnerGroup[] = [];
  const sorted = [...items].sort((a, b) => a.sortOrder - b.sortOrder);
  for (const item of sorted) {
    const owner = item.owner as ChecklistOwner;
    const group = groups.find((entry) => entry.owner === owner);
    if (group) {
      group.items.push(item);
    } else {
      groups.push({ owner, items: [item] });
    }
  }
  return groups;
}
