import { useT } from "@trenova/shared/i18n/use-t";
import { EmptyState } from "@/components/empty-state";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { ConstructionIcon } from "lucide-react";
import { useLocation } from "react-router";

export function PlaceholderPage() {
  const t = useT();

  const location = useLocation();

  return (
    <PageLayout
      pageHeaderProps={{
        title: formatPathToTitle(location.pathname),
        description: t("This page is under construction"),
      }}
    >
      <div className="flex flex-1 items-center justify-center">
        <EmptyState
          title={t("This page is under construction")}
          description={location.pathname}
          icons={[ConstructionIcon]}
        />
      </div>
    </PageLayout>
  );
}

function formatPathToTitle(path: string): string {
  return path
    .split("/")
    .filter(Boolean)
    .map((segment) =>
      segment
        .split("-")
        .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
        .join(" "),
    )
    .join(" → ");
}
