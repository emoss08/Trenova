import { useBreadcrumbLabel } from "@/hooks/use-breadcrumb-label";
import { PresetEditor } from "../_components/preset-editor";

export function NewHomeLayoutPage() {
  useBreadcrumbLabel("New Home Screen");
  return <PresetEditor />;
}
