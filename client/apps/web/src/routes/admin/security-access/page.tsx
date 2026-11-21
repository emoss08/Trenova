import { PageLayout } from "@/components/navigation/sidebar-layout";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy } from "react";
import { SecurityAccessSkeleton } from "./skeleton";

const SecurityAccessBody = lazy(() => import("./_components/security-access-body"));

/**
 * Security and access: how the organization's people sign in, how accounts reach it from a
 * directory, the policies a sign-in must meet and what happened, with a briefing on all of it
 * first. Each part is shown to whoever may read it.
 */
export function SecurityAccessPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Security & access"),
        description: t(
          "How your people sign in and are provisioned, the policies they meet, and what happened",
        ),
      }}
    >
      <LazyComponent fallback={<SecurityAccessSkeleton />}>
        <SecurityAccessBody />
      </LazyComponent>
    </PageLayout>
  );
}
