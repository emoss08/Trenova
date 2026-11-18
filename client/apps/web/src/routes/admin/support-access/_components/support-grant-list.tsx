import { fetchSupportAccessGrants, type SupportAccessGrant } from "@/lib/graphql/support-access";
import { supportAccess } from "@/lib/queries/support-access";
import { useInfiniteQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { useT } from "@trenova/shared/i18n/use-t";
import { SupportGrantCard } from "./support-grant-card";
import { SupportGrantsSkeleton } from "./support-grant-skeleton";

export function SupportGrantList({
  canRevoke,
  canExtend,
  onRevoke,
  onExtend,
}: {
  canRevoke: boolean;
  canExtend: boolean;
  onRevoke: (grant: SupportAccessGrant) => void;
  onExtend: (grant: SupportAccessGrant) => void;
}) {
  const t = useT();
  const query = useInfiniteQuery({
    queryKey: supportAccess.grants().queryKey,
    queryFn: ({ pageParam, signal }) => fetchSupportAccessGrants(pageParam, { signal }),
    initialPageParam: null as string | null,
    getNextPageParam: (page) => page.nextCursor,
  });

  if (query.isLoading) {
    return <SupportGrantsSkeleton />;
  }

  const grants = query.data?.pages.flatMap((page) => page.items) ?? [];
  if (grants.length === 0) {
    return (
      <EmptySheet
        className="my-10"
        title={t("Trenova support has never had access")}
        description={t(
          "When you need help, grant one named Trenova staff member access for a set time. It ends on its own, and you can end it sooner.",
        )}
        sketch={
          <div className="flex flex-col gap-3">
            <GhostLine className="w-1/2" />
            <GhostLine className="w-2/3" />
          </div>
        }
      />
    );
  }

  return (
    <div className="flex flex-col gap-3">
      {grants.map((grant) => (
        <SupportGrantCard
          key={grant.id}
          grant={grant}
          canRevoke={canRevoke}
          canExtend={canExtend}
          onRevoke={() => onRevoke(grant)}
          onExtend={() => onExtend(grant)}
        />
      ))}
      {query.hasNextPage ? (
        <Button
          type="button"
          size="sm"
          variant="ghost"
          className="self-center"
          isLoading={query.isFetchingNextPage}
          onClick={() => query.fetchNextPage()}
        >
          {t("Show older access")}
        </Button>
      ) : null}
    </div>
  );
}
