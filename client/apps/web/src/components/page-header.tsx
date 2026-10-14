import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import { InfoPopover } from "./info-popover";
import { Metadata } from "./metadata";

export type PageHeaderProps = {
  title: string;
  description: string;
  context?: React.ReactNode;
  actions?: React.ReactNode;
  /**
   * What the shell puts before the title when the title bar is also the
   * page bar: the sidebar fold, history and the trail. Pages never set it.
   */
  leading?: React.ReactNode;
  /** What the shell puts after the actions, set apart by a hairline. */
  trailing?: React.ReactNode;
  includeMetadata?: boolean;
  className?: string;
  /**
   * Keeps the title and description for assistive technology and the
   * document's metadata without drawing the bar: a room draws its own title
   * in its rail, so the bar would say it twice. The page's actions and context
   * go with the bar; a room puts them where it draws its title.
   */
  visuallyHidden?: boolean;
};

export const PAGE_HEADER_HEIGHT_CLASS = "min-h-11";

export function PageHeader({
  title,
  description,
  context,
  actions,
  leading,
  trailing,
  includeMetadata = true,
  className,
  visuallyHidden = false,
}: PageHeaderProps) {
  if (visuallyHidden) {
    return (
      <div data-slot="page-header" className="sr-only">
        <h1>{title}</h1>
        {description ? <p>{description}</p> : null}
        {includeMetadata && <Metadata title={title} description={description} />}
      </div>
    );
  }

  return (
    <div
      data-slot="page-header"
      className={cn(
        "border-border bg-background flex shrink-0 flex-wrap items-center justify-between gap-x-3 gap-y-1.5 border-b px-4 py-1.5",
        PAGE_HEADER_HEIGHT_CLASS,
        className,
      )}
    >
      <div className="flex min-w-0 items-center gap-1.5">
        {leading}
        <h1 className="truncate text-lg font-semibold">{title}</h1>
        {description ? <InfoPopover title={title}>{description}</InfoPopover> : null}
        {context ? <div className="ml-1 flex min-w-0 items-center gap-2">{context}</div> : null}
      </div>
      {actions || trailing ? (
        <div className="flex shrink-0 flex-wrap items-center gap-2">
          {actions}
          {trailing ? (
            <div
              className={cn(
                "flex items-center gap-1",
                actions && "border-border ml-1 border-l pl-3",
              )}
            >
              {trailing}
            </div>
          ) : null}
        </div>
      ) : null}
      {includeMetadata && <Metadata title={title} description={description} />}
    </div>
  );
}

export function PageHeaderSkeleton() {
  return (
    <div
      className={cn(
        "border-border flex shrink-0 items-center border-b px-4 py-1.5",
        PAGE_HEADER_HEIGHT_CLASS,
      )}
    >
      <Skeleton className="h-5 w-40" />
    </div>
  );
}
