import { useHomeLayoutPreset } from "@/hooks/use-home-layout";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Link, useParams } from "react-router";
import { PresetEditor } from "../_components/preset-editor";

export function EditHomeLayoutPage() {
  const { id } = useParams<{ id: string }>();
  const { data: preset, isLoading, isError } = useHomeLayoutPreset(id);

  if (isLoading) {
    return (
      <div className="flex flex-col gap-3 p-6">
        <Skeleton className="h-10 w-64" />
        <Skeleton className="h-48 rounded-lg" />
      </div>
    );
  }

  if (isError || !preset) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 py-24 text-center">
        <p className="text-sm font-medium">That home screen no longer exists</p>
        <p className="max-w-sm text-xs text-muted-foreground">
          It may have been deleted by another administrator.
        </p>
        <Link to="/admin/home-layouts" className="pt-1">
          <Button variant="outline" size="sm">
            Back to home screens
          </Button>
        </Link>
      </div>
    );
  }

  // Keyed on version so a preset saved in another tab remounts the editor with
  // the server's copy rather than leaving stale draft state on screen.
  return <PresetEditor key={`${preset.id}:${preset.version}`} preset={preset} />;
}
