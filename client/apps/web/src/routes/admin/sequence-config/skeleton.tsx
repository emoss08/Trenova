import {
  SettingsSectionsSkeleton,
  SkeletonRegion,
  type SettingsSectionSpec,
} from "@/components/settings-form-skeleton";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

/** Items in each group of the sidebar: operations, billing, accounting, settlements, locations. */
const sidebarGroupSizes = [4, 2, 3, 2, 1] as const;

const sections: readonly SettingsSectionSpec[] = [
  { title: "sm", ruled: true, width: "full", cols: 2, fields: ["input", "input", "switch"] },
  { title: "sm", ruled: true, width: "full", cols: 2, fields: [{ kind: "switch", count: 4 }] },
  { title: "sm", ruled: true, width: "full", cols: 2, fields: [{ kind: "switch", count: 2 }] },
  { title: "sm", fields: [] },
];

function SidebarSkeleton() {
  return (
    <div className="sticky top-4 hidden w-60 shrink-0 self-start md:block">
      <div className="bg-card rounded-md border p-3">
        <div className="flex flex-col gap-5">
          {sidebarGroupSizes.map((size, group) => (
            <div key={group}>
              <div className="mb-1.5 flex h-4 items-center px-2">
                <Skeleton className="h-3 w-20" />
              </div>
              <div className="flex flex-col gap-0.5">
                {Array.from({ length: size }, (_, item) => (
                  <div key={item} className="flex h-9 items-center gap-2.5 px-2.5">
                    <Skeleton className="size-4 shrink-0" />
                    <Skeleton className="h-3.5 w-28" />
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function PreviewSkeleton() {
  return (
    <div className="rounded-lg border px-4 py-3.5">
      <div className="mb-1.5 flex items-center justify-between gap-3">
        <Skeleton className="h-3 w-20" />
        <Skeleton className="h-6 w-16" />
      </div>
      <div className="flex h-7 items-center">
        <Skeleton className="h-5 w-48" />
      </div>
      <Skeleton className="mt-1.5 h-3 w-72 max-w-full" />
    </div>
  );
}

/**
 * The sequence workspace: the sidebar of sequence types, and the first type's panel with its
 * heading, live preview and section cards.
 */
export function SequenceConfigSkeleton() {
  const t = useT();

  return (
    <SkeletonRegion label={t("Loading settings")} className="flex gap-6 pb-20">
      <SidebarSkeleton />
      <div className="flex min-w-0 flex-1 flex-col gap-6">
        <div className="flex items-start justify-between gap-4">
          <div className="flex flex-col gap-1">
            <Skeleton className="h-4 w-48" />
            <span className="flex h-5 items-center">
              <Skeleton className="h-3.5 w-96 max-w-full" />
            </span>
          </div>
          <Skeleton className="h-7 w-32 shrink-0" />
        </div>
        <PreviewSkeleton />
        <SettingsSectionsSkeleton sections={sections} />
      </div>
    </SkeletonRegion>
  );
}
