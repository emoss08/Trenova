import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsList, TabsPanel, TabsTab } from "@/components/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { usePermission } from "@/hooks/use-permission";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { Operation, Resource } from "@/types/permission";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  GitBranchIcon,
  PlayIcon,
  Settings2Icon,
  TestTube2Icon,
  TrashIcon,
  TriangleAlertIcon,
} from "lucide-react";
import { lazy, Suspense } from "react";
import { toast } from "sonner";

const MetadataTab = lazy(() => import("./tabs/metadata-tab"));
const VersionsTab = lazy(() => import("./tabs/versions-tab"));
const FixturesTab = lazy(() => import("./tabs/fixtures-tab"));
const SimulationTab = lazy(() => import("./tabs/simulation-tab"));

type RuleSetDetailProps = {
  ruleSetId: string;
  onDeleted: () => void;
};

function RuleSetDetailSkeleton() {
  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b px-4 py-3">
        <div className="flex items-center gap-3">
          <Skeleton className="h-6 w-40" />
          <Skeleton className="h-5 w-24 rounded-full" />
          <Skeleton className="h-5 w-20 rounded-full" />
        </div>
        <Skeleton className="size-8 rounded-md" />
      </div>
      <div className="mx-4 mt-2 flex gap-2">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-8 w-24 rounded-md" />
        ))}
      </div>
      <div className="space-y-3 p-4">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-3/4" />
        <Skeleton className="h-32 w-full" />
      </div>
    </div>
  );
}

function TabPanelSkeleton() {
  return (
    <div className="space-y-3">
      <Skeleton className="h-10 w-full" />
      <Skeleton className="h-10 w-full" />
      <Skeleton className="h-10 w-3/4" />
      <Skeleton className="h-32 w-full" />
    </div>
  );
}

function RuleSetDetailError({ message }: { message: string }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 p-8 text-center">
      <TriangleAlertIcon className="size-8 text-muted-foreground" />
      <p className="text-sm text-muted-foreground">{message}</p>
    </div>
  );
}

export function RuleSetDetail({ ruleSetId, onDeleted }: RuleSetDetailProps) {
  const queryClient = useQueryClient();
  const { allowed: canDelete } = usePermission(Resource.DocumentParsingRule, Operation.Delete);

  const {
    data: ruleSet,
    isLoading,
    isError,
    error,
  } = useQuery({
    ...queries.documentParsingRule.detail(ruleSetId),
  });

  const deleteMutation = useMutation({
    mutationFn: () => apiService.documentParsingRuleService.delete(ruleSetId),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: queries.documentParsingRule.list._def,
      });
      toast.success("Rule set deleted");
      onDeleted();
    },
    onError: () => {
      toast.error("Failed to delete rule set");
    },
  });

  if (isLoading) {
    return <RuleSetDetailSkeleton />;
  }

  if (isError || !ruleSet) {
    return (
      <RuleSetDetailError
        message={error?.message ?? "Failed to load rule set. It may have been deleted."}
      />
    );
  }

  return (
    <div key={ruleSetId} className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b px-4 py-3">
        <div className="flex items-center gap-3">
          <h2 className="text-lg font-semibold">{ruleSet.name}</h2>
          <Badge variant="info">{ruleSet.documentKind}</Badge>
          {ruleSet.publishedVersionId ? (
            <Badge variant="active">Published</Badge>
          ) : (
            <Badge variant="secondary">No Published Version</Badge>
          )}
        </div>
        {canDelete && (
          <AlertDialog>
            <Tooltip>
              <TooltipTrigger
                render={
                  <AlertDialogTrigger
                    render={
                      <Button variant="ghost" size="icon" className="text-destructive">
                        <TrashIcon className="size-4" />
                      </Button>
                    }
                  />
                }
              />
              <TooltipContent>Delete rule set</TooltipContent>
            </Tooltip>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogMedia className="bg-destructive/10">
                  <TrashIcon className="size-5 text-destructive" />
                </AlertDialogMedia>
                <AlertDialogTitle>Delete Rule Set</AlertDialogTitle>
                <AlertDialogDescription>
                  This will permanently delete &quot;{ruleSet.name}&quot; including all versions,
                  fixtures, and simulation results. Any shipments currently using this rule set will
                  fall back to default parsing. This action cannot be undone.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction
                  onClick={() => deleteMutation.mutate()}
                  disabled={deleteMutation.isPending}
                  className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                >
                  {deleteMutation.isPending ? "Deleting..." : "Delete"}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        )}
      </div>

      <Tabs defaultValue="metadata" className="flex flex-1 flex-col">
        <TabsList variant="underline" className="mx-4 mt-2 w-fit">
          <TabsTab value="metadata">
            <Settings2Icon className="mr-1.5 size-3.5" />
            Metadata
          </TabsTab>
          <TabsTab value="versions">
            <GitBranchIcon className="mr-1.5 size-3.5" />
            Versions
          </TabsTab>
          <TabsTab value="fixtures">
            <TestTube2Icon className="mr-1.5 size-3.5" />
            Fixtures
          </TabsTab>
          <TabsTab value="simulation">
            <PlayIcon className="mr-1.5 size-3.5" />
            Simulation
          </TabsTab>
        </TabsList>

        <div className="flex-1 overflow-auto px-4">
          <TabsPanel value="metadata">
            <Suspense fallback={<TabPanelSkeleton />}>
              <MetadataTab ruleSetId={ruleSetId} />
            </Suspense>
          </TabsPanel>
          <TabsPanel value="versions">
            <Suspense fallback={<TabPanelSkeleton />}>
              <VersionsTab ruleSetId={ruleSetId} />
            </Suspense>
          </TabsPanel>
          <TabsPanel value="fixtures">
            <Suspense fallback={<TabPanelSkeleton />}>
              <FixturesTab ruleSetId={ruleSetId} />
            </Suspense>
          </TabsPanel>
          <TabsPanel value="simulation">
            <Suspense fallback={<TabPanelSkeleton />}>
              <SimulationTab ruleSetId={ruleSetId} />
            </Suspense>
          </TabsPanel>
        </div>
      </Tabs>
    </div>
  );
}
