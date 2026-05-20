import { Badge } from "@/components/ui/badge";
import { TextShimmer } from "@/components/ui/text-shimmer";
import {
  useSelectedTemplateDesignerIds,
  useTemplateDesignerTemplateListInfiniteQuery,
  useTemplateDesignerUrlActions,
} from "@/hooks/use-template-designer-state";
import { cn } from "@/lib/utils";
import { useTemplateDesignerStore } from "@/stores/template-designer-store";
import { useEffect, useMemo, useRef } from "react";

export default function TemplateList() {
  const templatesQuery = useTemplateDesignerTemplateListInfiniteQuery();
  const templates = useMemo(
    () => templatesQuery.data?.pages.flatMap((page) => page.results) ?? [],
    [templatesQuery.data?.pages],
  );
  const { selectedTemplateId } = useSelectedTemplateDesignerIds();
  const resetDraftState = useTemplateDesignerStore((state) => state.resetDraftState);
  const { patchTemplateUrlState } = useTemplateDesignerUrlActions();
  const observerTarget = useRef<HTMLDivElement>(null);
  const { fetchNextPage, hasNextPage, isFetchingNextPage } = templatesQuery;

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting && hasNextPage && !isFetchingNextPage) {
          void fetchNextPage();
        }
      },
      { threshold: 0.1 },
    );

    const currentTarget = observerTarget.current;
    if (currentTarget) observer.observe(currentTarget);

    return () => {
      if (currentTarget) observer.unobserve(currentTarget);
    };
  }, [fetchNextPage, hasNextPage, isFetchingNextPage]);

  return (
    <div className="flex flex-col gap-0.5 p-2">
      {templates.map((template) => (
        <button
          key={template.id}
          type="button"
          onClick={() => {
            if (selectedTemplateId === template.id) return;
            resetDraftState();
            patchTemplateUrlState({
              templateId: template.id,
              versionId: "",
              segmentId: "",
              elementPosition: 0,
            });
          }}
          className={cn(
            "block w-full cursor-pointer rounded-md px-3 py-2 text-left hover:bg-muted",
            selectedTemplateId === template.id && "bg-muted",
          )}
        >
          <div className="flex items-center justify-between gap-2">
            <span className="truncate text-sm font-medium">{template.name}</span>
            <Badge variant={template.status === "Active" ? "active" : "outline"}>
              {template.status}
            </Badge>
          </div>
          <div className="mt-1 text-xs text-muted-foreground">
            {template.transactionSet} {template.direction} / {template.versions.length} versions
          </div>
        </button>
      ))}
      {templatesQuery.isLoading ? (
        <div className="flex justify-center p-3">
          <TextShimmer className="font-mono text-xs" duration={1}>
            Loading templates...
          </TextShimmer>
        </div>
      ) : null}
      {templatesQuery.isError ? (
        <div className="p-3 text-sm text-destructive">Failed to load templates.</div>
      ) : null}
      {!templatesQuery.isLoading && templates.length === 0 ? (
        <div className="p-3 text-sm text-muted-foreground">No matching templates.</div>
      ) : null}
      {isFetchingNextPage ? (
        <div className="flex justify-center p-3">
          <TextShimmer className="font-mono text-xs" duration={1}>
            Loading more...
          </TextShimmer>
        </div>
      ) : null}
      <div ref={observerTarget} className="h-px" aria-hidden />
    </div>
  );
}
