import { useT } from "@trenova/shared/i18n/use-t";
import { useBreadcrumbLabel } from "@/hooks/use-breadcrumb-label";
import { useHomeLayoutPreset } from "@/hooks/use-home-layout";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Link, useParams } from "react-router";
import { PresetEditor } from "../_components/preset-editor";

export function EditHomeLayoutPage() {
  const t = useT();

  const { id } = useParams<{ id: string }>();
  const { data: preset, isLoading, isError } = useHomeLayoutPreset(id);
  useBreadcrumbLabel(preset?.name);

  if (isLoading) {
    return (
      <PageLayout pageHeaderProps={{ title: t("Home screens"), description: t("Loading...") }}>
        <Skeleton className="h-10 w-64" />
        <Skeleton className="h-48 rounded-lg" />
      </PageLayout>
    );
  }

  if (isError || !preset) {
    return (
      <PageLayout
        pageHeaderProps={{
          title: t("Home screens"),
          description: t("It may have been deleted by another administrator."),
        }}
      >
        <div className="border-border flex flex-col items-center justify-center gap-2 rounded-lg border border-dashed py-16 text-center">
          <p className="text-sm font-medium">{t("That home screen no longer exists")}</p>
          <p className="text-muted-foreground max-w-sm text-xs">
            {t("It may have been deleted by another administrator.")}
          </p>
          <Link to="/admin/home-layouts" className="pt-1">
            <Button variant="outline" size="sm">
              {t("Back to home screens")}
            </Button>
          </Link>
        </div>
      </PageLayout>
    );
  }

  // Keyed on version so a preset saved in another tab remounts the editor with
  // the server's copy rather than leaving stale draft state on screen.
  return <PresetEditor key={`${preset.id}:${preset.version}`} preset={preset} />;
}
