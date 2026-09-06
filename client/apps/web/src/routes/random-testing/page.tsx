import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const RandomTestingConsole = lazy(() => import("./_components/random-testing-console"));

export function RandomTestingPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Random Testing",
        description:
          "The pools drivers are drawn from and the rounds drawn from them. Each round keeps the seed it was drawn with, so a selection can be re-checked years later.",
      }}
    >
      <div className="flex flex-col gap-4">
        <DataTableLazyComponent>
          <RandomTestingConsole />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
