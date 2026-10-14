import { useT } from "@trenova/shared/i18n/use-t";
import { FavoriteToggle, HistoryNavigation, SidebarToggle } from "@/components/header";
import { useBreadcrumbs } from "@/hooks/use-breadcrumb";
import { cn } from "@trenova/shared/lib/utils";
import { Link, useNavigation } from "react-router";

/**
 * The two things every page bar leads with: the panel's fold and history.
 * A page that draws its own bar (Home's briefing, AI control's head) puts
 * them in the same place, so they are in the same place everywhere.
 */
export function PageBarControls({ className }: { className?: string }) {
  return (
    <div className={cn("mr-1 flex shrink-0 items-center gap-0.5", className)}>
      <SidebarToggle />
      <HistoryNavigation />
    </div>
  );
}

/**
 * The trail to the page, drawn into the page's own title bar. Home is not a
 * crumb (the rail holds it), and a crumb that only repeats the title is
 * dropped, so the Shipments board reads "Shipments" once rather than three
 * times. On a phone the trail goes, and the title keeps the width.
 */
export function PageBarTrail({ title }: { title: string }) {
  const t = useT();
  const navigation = useNavigation();
  const breadcrumbs = useBreadcrumbs();
  const isLoading = navigation.state === "loading";
  const last = breadcrumbs.at(-1);
  const parents = breadcrumbs
    .slice(0, -1)
    .filter((crumb) => crumb.crumb !== title && crumb.crumb !== last?.crumb);

  if (parents.length === 0) {
    return null;
  }

  return (
    <nav
      aria-label={t("Breadcrumb")}
      className="flex min-w-0 shrink items-center gap-1.5 max-md:hidden"
    >
      {parents.map((crumb) => (
        <span key={crumb.id} className="flex min-w-0 items-center gap-1.5">
          <Link
            to={crumb.pathname}
            className={cn(
              "text-foreground-muted hover:text-foreground truncate text-sm transition-colors",
              isLoading && "opacity-50",
            )}
          >
            {crumb.crumb}
          </Link>
          <span aria-hidden className="text-foreground-subtle text-sm">
            /
          </span>
        </span>
      ))}
    </nav>
  );
}

export function PageBarLeading({ title }: { title: string }) {
  return (
    <>
      <PageBarControls />
      <PageBarTrail title={title} />
    </>
  );
}

export function PageBarTrailing() {
  return <FavoriteToggle />;
}
