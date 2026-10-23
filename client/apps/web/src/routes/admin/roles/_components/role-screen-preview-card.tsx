import type { RoleDraft } from "@/lib/graphql/page-access";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy } from "react";
import { RoleScreenPreviewSkeleton } from "./role-screen-preview-skeleton";

const RoleScreenPreview = lazy(() =>
  import("./role-screen-preview").then((module) => ({ default: module.RoleScreenPreview })),
);

/** The role editor's "Preview a screen" card, whose body is read when the editor draws it. */
export function RoleScreenPreviewCard({ draft }: { draft: RoleDraft }) {
  const t = useT();

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Preview a screen")}</CardTitle>
        <CardDescription>
          {t(
            "See what a person holding only this role would need and see on a screen, before you save.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <LazyComponent fallback={<RoleScreenPreviewSkeleton />}>
          <RoleScreenPreview draft={draft} />
        </LazyComponent>
      </CardContent>
    </Card>
  );
}
