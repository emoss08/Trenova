import { FavoriteToggle, HistoryNavigation, SidebarToggle } from "@/components/header";
import { useBreadcrumbs } from "@/hooks/use-breadcrumb";
import { cn } from "@trenova/shared/lib/utils";
import { Link, useNavigation } from "react-router";

/**
 * The trail to the page, drawn into the page's own title bar: the fold,
 * history, then every crumb above the title. Home is not a crumb (the rail
 * holds it), and a crumb that only repeats the title is dropped, so the
 * Shipments board reads "Shipments" once rather than three times.
 */
export function RailPageBarLeading({ title }: { title: string }) {
  const navigation = useNavigation();
  const breadcrumbs = useBreadcrumbs();
  const isLoading = navigation.state === "loading";
  const last = breadcrumbs.at(-1);
  const parents = breadcrumbs
    .slice(0, -1)
    .filter((crumb) => crumb.crumb !== title && crumb.crumb !== last?.crumb);

  return (
    <>
      <div className="mr-1 flex shrink-0 items-center gap-0.5">
        <SidebarToggle />
        <HistoryNavigation />
      </div>
      {parents.length > 0 && (
        <nav
          aria-label="Breadcrumb"
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
      )}
    </>
  );
}

export function RailPageBarTrailing() {
  return <FavoriteToggle />;
}
