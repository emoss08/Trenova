import { FormEditModal } from "@/components/form-edit-modal";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { TextShimmer } from "@/components/ui/text-shimmer";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  useSelectedTemplateDesignerIds,
  useTemplateDesignerTemplateListInfiniteQuery,
  useTemplateDesignerUrlActions,
} from "@/hooks/use-template-designer-state";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import { useTemplateDesignerStore } from "@/stores/template-designer-store";
import { ediTemplateSchema, type EDITemplate, type UpdateEDITemplateRequest } from "@/types/edi";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { PencilIcon } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { CreateTemplateForm } from "./create-template-form";

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
        <TemplateListItem
          key={template.id}
          template={template}
          isSelected={selectedTemplateId === template.id}
          onSelect={() => {
            if (selectedTemplateId === template.id) return;
            resetDraftState();
            patchTemplateUrlState({
              templateId: template.id,
              versionId: "",
              segmentId: "",
              elementPosition: 0,
            });
          }}
        />
      ))}
      {templatesQuery.isLoading ? (
        <div className="flex justify-center p-3">
          <TextShimmer className="font-mono text-xs" duration={1}>
            Loading templates...
          </TextShimmer>
        </div>
      ) : null}
      {templatesQuery.isError ? (
        <div className="flex flex-col items-center gap-2 p-3 text-center">
          <span className="text-sm text-destructive">Failed to load templates.</span>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => void templatesQuery.refetch()}
          >
            Retry
          </Button>
        </div>
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

function TemplateListItem({
  template,
  isSelected,
  onSelect,
}: {
  template: EDITemplate;
  isSelected: boolean;
  onSelect: () => void;
}) {
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);

  return (
    <>
      <div
        className={cn("group flex items-start rounded-md hover:bg-muted", isSelected && "bg-muted")}
      >
        <button
          type="button"
          onClick={onSelect}
          className="min-w-0 flex-1 cursor-pointer px-3 py-2 text-left"
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
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                type="button"
                variant="ghost"
                size="icon-xs"
                className="mt-2 mr-2 opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100"
                aria-label={`Edit ${template.name}`}
                onClick={() => setIsEditDialogOpen(true)}
              >
                <PencilIcon className="size-3.5" />
              </Button>
            }
          />
          <TooltipContent>Edit template</TooltipContent>
        </Tooltip>
      </div>
      <TemplateEditDialog
        template={template}
        open={isEditDialogOpen}
        onOpenChange={setIsEditDialogOpen}
      />
    </>
  );
}

function TemplateEditDialog({
  template,
  open,
  onOpenChange,
}: {
  template: EDITemplate;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();
  const form = useForm({
    resolver: zodResolver(ediTemplateSchema),
    defaultValues: template,
  });

  return (
    <FormEditModal
      open={open}
      onOpenChange={onOpenChange}
      currentRecord={template}
      form={form}
      url="/edi/templates/"
      queryKey="templates"
      title="EDI Template"
      fieldKey="name"
      className="sm:max-w-120"
      formComponent={<CreateTemplateForm mode="edit" />}
      transformValues={(values): UpdateEDITemplateRequest => {
        return {
          name: values.name.trim(),
          description: values.description?.trim() ?? "",
          status: values.status,
          version: values.version,
        };
      }}
      onSuccess={async (updatedTemplate) => {
        await queryClient.invalidateQueries({ queryKey: queries.edi.templates._def });
        await queryClient.invalidateQueries({
          queryKey: queries.edi.template(updatedTemplate.id).queryKey,
        });
      }}
    />
  );
}
