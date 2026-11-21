import { SkeletonRegion } from "@/components/settings-form-skeleton";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { SecurityBriefingSkeleton } from "./_components/security-access/briefing/security-briefing-skeleton";
import { firstAllowed, useSecurityAccess } from "./_components/security-access/use-security-access";
import { securityTabValues, useSecurityTab } from "./security-access-state";

/**
 * The page's loading state: its tab row, one tab for each part this person may read, and
 * the briefing when the overview is the tab that opens. Another tab draws its own loading
 * state once its code is here.
 */
export function SecurityAccessSkeleton() {
  const t = useT();
  const access = useSecurityAccess();
  const [requestedTab] = useSecurityTab();
  const openTab = firstAllowed(requestedTab, securityTabValues, access.tabs);
  const tabs = securityTabValues.filter((value) => access.tabs[value]).length;

  return (
    <div className="flex flex-col gap-4 pb-10">
      <SkeletonRegion label={t("Loading security and access")}>
        <div className="flex w-fit items-center gap-x-0.5 py-1">
          {Array.from({ length: tabs }, (_, index) => (
            <span
              key={index}
              data-slot="tab-skeleton"
              className="flex h-9 items-center gap-1.5 px-2.5 sm:h-8"
            >
              <Skeleton className="size-4" />
              <Skeleton className="h-3.5 w-16" />
            </span>
          ))}
        </div>
      </SkeletonRegion>
      {openTab === "overview" ? <SecurityBriefingSkeleton /> : null}
    </div>
  );
}
