import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

const ZONE_NOTE_WIDTHS = ["w-72", "w-44", "w-52"] as const;

/**
 * The import page while its workspace's code is read. Every import opens on the upload step, so
 * this is the page's frame and title over that step: the heading and its line, and the drop zone
 * with its button, its hint and the three notes under it.
 */
export function ImportWorkspaceSkeleton() {
  const t = useT();

  return (
    <PageLayout
      fill
      className="gap-y-0 p-0"
      pageHeaderProps={{
        title: t("Import from rate confirmation"),
        description: t("Upload a rate confirmation to extract shipment details."),
        actions: <Skeleton className="size-7" />,
      }}
    >
      <div
        className="flex flex-1 flex-col items-center justify-center p-8"
        aria-busy
        aria-label={t("Loading the import")}
      >
        <div className="w-full max-w-md space-y-6" aria-hidden>
          <div className="flex flex-col items-center space-y-1">
            <div className="text-base">
              <Skeleton className="inline-block h-3.5 w-44 align-middle" />
            </div>
            <div className="text-sm">
              <Skeleton className="inline-block h-3 w-72 align-middle" />
            </div>
          </div>
          <div className="border-muted-foreground/25 flex flex-col items-center justify-center rounded-lg border-2 border-dashed p-6">
            <div className="flex flex-col items-center gap-2">
              <Skeleton className="h-8 w-24" />
              <div className="text-sm">
                <Skeleton className="inline-block h-3 w-36 align-middle" />
              </div>
            </div>
            <div className="mt-4 space-y-1 text-xs">
              {ZONE_NOTE_WIDTHS.map((width, index) => (
                <div key={index}>
                  <Skeleton className={`inline-block h-2.5 align-middle ${width}`} />
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </PageLayout>
  );
}
