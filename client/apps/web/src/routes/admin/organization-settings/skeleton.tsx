import {
  SettingsSectionsSkeleton,
  SkeletonRegion,
  type SettingsSectionSpec,
} from "@/components/settings-form-skeleton";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

const tabCount = 2;

const generalSections: readonly SettingsSectionSpec[] = [
  { title: "lg", fields: ["image"] },
  { fields: [{ kind: "input", count: 3 }] },
  {
    title: "sm",
    description: 2,
    switches: "right",
    fields: ["input", { kind: "switch", count: 2 }],
  },
  { fields: [{ kind: "input", count: 3 }] },
  {
    cols: 2,
    fields: [
      { kind: "input", full: true, count: 2 },
      { kind: "input", count: 2 },
      { kind: "input", full: true },
    ],
  },
];

/**
 * The settings tabs over the general tab's cards: branding, details, operating model, compliance
 * and the registered address.
 */
export function OrganizationSettingsSkeleton() {
  const t = useT();

  return (
    <SkeletonRegion label={t("Loading settings")} className="flex flex-col gap-4">
      <div className="flex w-fit items-center gap-x-0.5 py-1">
        {Array.from({ length: tabCount }, (_, tab) => (
          <div key={tab} className="flex h-9 items-center gap-1.5 px-2.5 sm:h-8">
            <Skeleton className="size-4" />
            <Skeleton className="h-3.5 w-14" />
          </div>
        ))}
      </div>
      <div className="flex flex-col gap-4 pb-10">
        <SettingsSectionsSkeleton sections={generalSections} />
      </div>
    </SkeletonRegion>
  );
}
