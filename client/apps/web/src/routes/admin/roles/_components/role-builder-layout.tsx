import { PageLayout } from "@/components/navigation/sidebar-layout";
import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";

type RolePageLayoutProps = {
  title: string;
  description: string;
  isSubmitting: boolean;
  submitLabel: string;
  onSubmit: () => void;
  onCancel: () => void;
  permissionCount?: number;
  children: React.ReactNode;
  banner?: React.ReactNode;
};

export function RolePageLayout({
  title,
  description,
  isSubmitting,
  submitLabel,
  onSubmit,
  onCancel,
  permissionCount,
  children,
  banner,
}: RolePageLayoutProps) {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title,
        description,
        context:
          permissionCount !== undefined && permissionCount > 0 ? (
            <Badge variant="neutral" appearance="outline">
              {t("{0, plural, one {# resource} other {# resources}} configured", permissionCount)}
            </Badge>
          ) : undefined,
        actions: (
          <>
            <Button type="button" variant="outline" size="sm" onClick={onCancel}>
              {t("Cancel")}
            </Button>
            <Button
              type="button"
              size="sm"
              onClick={onSubmit}
              isLoading={isSubmitting}
              loadingText={submitLabel}
            >
              {submitLabel}
            </Button>
          </>
        ),
      }}
    >
      <div className="mx-auto flex w-full max-w-5xl flex-col gap-6">
        {banner}
        {children}
      </div>
    </PageLayout>
  );
}
