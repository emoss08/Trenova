import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";

function MailboxCardSkeletonBody() {
  return (
    <div className="bg-card border-border flex flex-col rounded-lg border" aria-hidden>
      <div className="flex items-start gap-3 p-4">
        <Skeleton className="size-9 shrink-0" />
        <div className="flex flex-1 flex-col gap-1.5 pt-0.5">
          <div className="flex items-center gap-2">
            <Skeleton className="h-4 w-28" />
            <Skeleton className="h-5 w-16" />
          </div>
          <Skeleton className="h-3 w-48" />
          <Skeleton className="h-3 w-56" />
        </div>
      </div>
      <div className="border-border-subtle flex items-center justify-between gap-3 border-t px-4 py-2.5">
        <Skeleton className="h-3 w-44" />
        <Skeleton className="h-6 w-24" />
      </div>
    </div>
  );
}

export function MailboxCardsSkeleton() {
  const t = useT();

  return (
    <div className="grid gap-3 lg:grid-cols-2" aria-busy aria-label={t("Loading mailboxes")}>
      <MailboxCardSkeletonBody />
      <MailboxCardSkeletonBody />
    </div>
  );
}
