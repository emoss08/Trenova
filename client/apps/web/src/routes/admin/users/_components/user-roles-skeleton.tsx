import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableSkeleton } from "@trenova/shared/components/data-table/data-table-skeleton";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import { parseAsStringLiteral, useQueryState } from "nuqs";

export const USER_ROLES_TABS = ["users", "roles"] as const;

export type UserRolesTab = (typeof USER_ROLES_TABS)[number];

export const userRolesTabParser = parseAsStringLiteral(USER_ROLES_TABS).withDefault("users");

/**
 * The columns each tab's table draws, the selection column included, so the
 * table skeleton under the tabs has the loaded table's shape.
 */
export const USER_ROLES_COLUMN_COUNTS: Record<UserRolesTab, number> = {
  users: 6,
  roles: 4,
};

const TAB_WIDTHS: Record<UserRolesTab, string> = {
  users: "w-8",
  roles: "w-28",
};

/**
 * The users page while its tabs' code arrives: the tab strip where the real
 * one lands, and under it the table of the tab the address names. The tables
 * then load under the real strip with the same skeleton, so the page draws
 * one shape from the first paint to the first row.
 */
export function UserRolesSkeleton() {
  const t = useT();
  const [tab] = useQueryState("tab", userRolesTabParser);

  return (
    <div className="flex flex-col gap-2" aria-busy aria-label={t("Loading users and roles")}>
      <div className="contents" aria-hidden>
        <div className="flex w-fit items-center gap-x-0.5 py-1">
          {USER_ROLES_TABS.map((value) => (
            <span key={value} className="flex h-9 items-center px-2.5 sm:h-8">
              <Skeleton className={cn("h-3", TAB_WIDTHS[value])} />
            </span>
          ))}
        </div>
        <DataTableSkeleton columnCount={USER_ROLES_COLUMN_COUNTS[tab]} />
      </div>
    </div>
  );
}
