import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { usePermission } from "@/hooks/use-permission";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import type { RuleSet } from "@/types/document-parsing-rule";
import { Operation, Resource } from "@/types/permission";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CircleCheckIcon, FileTextIcon, PlusIcon, SearchIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";

type RuleSetListProps = {
  selectedId: string | null;
  onSelect: (id: string) => void;
};

export function RuleSetList({ selectedId, onSelect }: RuleSetListProps) {
  const [search, setSearch] = useState("");
  const [createOpen, setCreateOpen] = useState(false);

  const { allowed: canCreate } = usePermission(Resource.DocumentParsingRule, Operation.Create);

  const { data: ruleSets, isLoading } = useQuery({
    ...queries.documentParsingRule.list(),
  });

  const filtered = useMemo(() => {
    if (!ruleSets) return [];
    if (!search) return ruleSets;
    const lower = search.toLowerCase();
    return ruleSets.filter(
      (rs) =>
        rs.name.toLowerCase().includes(lower) ||
        rs.documentKind.toLowerCase().includes(lower) ||
        rs.description?.toLowerCase().includes(lower),
    );
  }, [ruleSets, search]);

  const totalCount = ruleSets?.length ?? 0;
  const filteredCount = filtered.length;
  const isFiltered = search.length > 0;
  const isEmpty = totalCount === 0 && !isLoading;

  return (
    <div className="flex h-full flex-col">
      <div className="space-y-2 border-b p-3">
        <Input
          leftElement={<SearchIcon className="size-4 text-muted-foreground" />}
          placeholder="Search rule sets..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="pl-8"
        />
        <div className="flex items-center justify-between">
          <span className="text-2xs text-muted-foreground">
            {isFiltered ? `${filteredCount} of ${totalCount}` : `${totalCount}`} rule set
            {totalCount !== 1 ? "s" : ""}
          </span>
          {canCreate && (
            <Button
              variant="outline"
              size="xs"
              className="gap-1"
              onClick={() => setCreateOpen(true)}
            >
              <PlusIcon className="size-3" />
              New
            </Button>
          )}
        </div>
      </div>

      <ScrollArea className="flex-1">
        {isLoading ? (
          <RuleSetListSkeleton />
        ) : isEmpty ? (
          <RuleSetEmptyState onCreateClick={canCreate ? () => setCreateOpen(true) : undefined} />
        ) : filteredCount === 0 ? (
          <RuleSetNoResults search={search} onClear={() => setSearch("")} />
        ) : (
          <div className="space-y-1 p-2">
            {filtered.map((rs) => (
              <RuleSetCard
                key={rs.id}
                ruleSet={rs}
                isSelected={rs.id === selectedId}
                onSelect={() => onSelect(rs.id!)}
              />
            ))}
          </div>
        )}
      </ScrollArea>

      <CreateRuleSetDialog open={createOpen} onOpenChange={setCreateOpen} onCreated={onSelect} />
    </div>
  );
}

function RuleSetCard({
  ruleSet,
  isSelected,
  onSelect,
}: {
  ruleSet: RuleSet;
  isSelected: boolean;
  onSelect: () => void;
}) {
  const isPublished = Boolean(ruleSet.publishedVersionId);

  return (
    <button
      type="button"
      onClick={onSelect}
      className={cn(
        "w-full rounded-md border p-3 text-left transition-colors",
        isSelected
          ? "border-primary bg-primary/5 ring-1 ring-primary"
          : "border-transparent hover:bg-muted/50",
      )}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <p className="truncate text-sm font-medium">{ruleSet.name}</p>
            {isPublished && (
              <Tooltip>
                <TooltipTrigger
                  render={
                    <span className="inline-flex shrink-0">
                      <CircleCheckIcon className="size-3.5 text-green-500" />
                    </span>
                  }
                />
                <TooltipContent side="right">Published</TooltipContent>
              </Tooltip>
            )}
          </div>
          {ruleSet.description && (
            <p className="mt-0.5 line-clamp-1 text-xs text-muted-foreground">
              {ruleSet.description}
            </p>
          )}
          <div className="mt-1.5 flex items-center gap-2">
            <Badge variant="info" className="text-2xs">
              {ruleSet.documentKind}
            </Badge>
            <span className="text-2xs text-muted-foreground">Priority {ruleSet.priority}</span>
          </div>
        </div>
      </div>
    </button>
  );
}

function RuleSetListSkeleton() {
  return (
    <div className="space-y-1 p-2">
      {Array.from({ length: 5 }).map((_, i) => (
        <div key={i} className="rounded-md border border-transparent p-3">
          <div className="flex items-start justify-between gap-2">
            <div className="min-w-0 flex-1 space-y-2">
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-3 w-full" />
              <div className="flex gap-2">
                <Skeleton className="h-5 w-24 rounded-md" />
                <Skeleton className="h-3 w-16" />
              </div>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

function RuleSetEmptyState({ onCreateClick }: { onCreateClick?: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 p-6 text-center">
      <div className="flex size-10 items-center justify-center rounded-full bg-muted">
        <FileTextIcon className="size-5 text-muted-foreground" />
      </div>
      <div className="space-y-1">
        <p className="text-sm font-medium">No rule sets yet</p>
        <p className="text-xs text-muted-foreground">
          Create a rule set to define how documents are parsed.
        </p>
      </div>
      {onCreateClick && (
        <Button variant="outline" size="sm" className="mt-1 gap-1" onClick={onCreateClick}>
          <PlusIcon className="size-3.5" />
          Create Rule Set
        </Button>
      )}
    </div>
  );
}

function RuleSetNoResults({ search, onClear }: { search: string; onClear: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 p-6 text-center">
      <p className="text-sm text-muted-foreground">No results for &quot;{search}&quot;</p>
      <Button variant="ghost" size="xs" onClick={onClear}>
        Clear search
      </Button>
    </div>
  );
}

function CreateRuleSetDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: (id: string) => void;
}) {
  const [name, setName] = useState("");
  const queryClient = useQueryClient();

  const { mutateAsync, isPending } = useMutation({
    mutationFn: async () =>
      apiService.documentParsingRuleService.create({
        name,
        description: "",
        documentKind: "RateConfirmation",
        priority: 100,
      }),
    onSuccess: (data) => {
      void queryClient.invalidateQueries({
        queryKey: queries.documentParsingRule.list._def,
      });
      onOpenChange(false);
      setName("");
      if (data.id) onCreated(data.id);
      toast.success("Rule set created");
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "Failed to create rule set");
    },
  });

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if (!name.trim()) return;
      await mutateAsync();
    },
    [name, mutateAsync],
  );

  function handleClose() {
    if (isPending) return;
    onOpenChange(false);
    setName("");
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent>
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Create Rule Set</DialogTitle>
            <DialogDescription>
              A rule set defines how a specific type of document is parsed. Each rule set contains
              versions with match criteria, section definitions, and field extraction rules.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="create-rule-set-name">Name</Label>
              <Input
                id="create-rule-set-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. CH Robinson Rate Confirmation"
                autoFocus
              />
              <p className="text-2xs text-muted-foreground">
                Use a descriptive name that identifies the provider or document format.
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose} disabled={isPending}>
              Cancel
            </Button>
            <Button type="submit" disabled={!name.trim() || isPending}>
              {isPending ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
