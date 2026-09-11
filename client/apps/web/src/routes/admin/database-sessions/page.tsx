import { useT } from "@trenova/shared/i18n/use-t";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@trenova/shared/components/ui/alert-dialog";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { ShikiCodeBlock } from "@trenova/shared/components/ui/shiki-code-block";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { apiService } from "@/services/api";
import type { DatabaseSessionChain } from "@/types/database-session";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowLeftIcon,
  ChevronDownIcon,
  CodeIcon,
  RefreshCwIcon,
  ShieldCheckIcon,
} from "lucide-react";
import { useMemo } from "react";
import { toast } from "sonner";

const queryKey = ["database-session-list"];

function durationLabel(seconds: number) {
  if (seconds < 60) {
    return `${seconds}s`;
  }
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return `${minutes}m ${remainingSeconds}s`;
}

function ageSeverityClass(seconds: number) {
  if (seconds > 120) return "text-orange-600 dark:text-orange-400";
  if (seconds >= 30) return "text-foreground";
  return "text-muted-foreground";
}

function AgeBadge({ label, seconds }: { label: string; seconds: number }) {
  return (
    <span className={`text-xs ${ageSeverityClass(seconds)}`}>
      {label} {durationLabel(seconds)}
    </span>
  );
}

function SessionSide({
  pid,
  appName,
  user,
  state,
  txAge,
  queryAge,
  align = "start",
}: {
  pid: number;
  appName: string;
  user: string;
  state: string;
  txAge: number;
  queryAge: number;
  align?: "start" | "end";
}) {
  const t = useT();

  return (
    <div
      className={`flex min-w-0 flex-col gap-1 ${align === "end" ? "items-end text-right" : "items-start text-left"}`}
    >
      <Badge variant="outline" className="w-fit px-0 font-mono">
        {t("PID {0}", pid)}
      </Badge>
      <span className="text-sm font-medium">{appName || "Unknown app"}</span>
      <span className="text-muted-foreground text-xs">{user || "Unknown user"}</span>
      <span className="text-muted-foreground text-xs">{state}</span>
      <div
        className={`mt-1 flex flex-col gap-0.5 ${align === "end" ? "items-end" : "items-start"}`}
      >
        <AgeBadge label={t("Tx age")} seconds={txAge} />
        <AgeBadge label={t("Query age")} seconds={queryAge} />
      </div>
    </div>
  );
}

function QueryBlock({ label, query }: { label: string; query: string }) {
  const t = useT();

  return (
    <div className="flex min-w-0 flex-1 flex-col gap-1">
      <span className="text-muted-foreground text-[10px] tracking-wider uppercase">{label}</span>
      {query ? (
        <div className="max-h-32 overflow-auto">
          <ShikiCodeBlock code={query} lang="plsql" darkTheme="vitesse-dark" />
        </div>
      ) : (
        <div className="bg-muted/50 rounded-md p-2">
          <span className="text-muted-foreground text-xs">{t("No query text")}</span>
        </div>
      )}
    </div>
  );
}

function TerminateButton({ row }: { row: DatabaseSessionChain }) {
  const t = useT();

  const queryClient = useQueryClient();
  const mutation = useMutation({
    mutationFn: (pid: number) => apiService.databaseSessionService.terminate(pid),
    onSuccess: async (response) => {
      toast.success(response.message);
      await queryClient.invalidateQueries({ queryKey });
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : "Failed to terminate session");
    },
  });

  return (
    <AlertDialog>
      <AlertDialogTrigger
        render={
          <Button
            size="xs"
            variant="outline"
            className="text-destructive hover:bg-destructive hover:text-destructive-foreground"
          />
        }
      >
        {t("Terminate")}
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("Terminate database session?")}</AlertDialogTitle>
          <AlertDialogDescription>
            {t("This will terminate backend PID {0} to release blocked PID {1}. The in-flight transaction on the blocker will be cancelled.", row.blockingPid, row.blockedPid)}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{t("Cancel")}</AlertDialogCancel>
          <AlertDialogAction onClick={() => mutation.mutate(row.blockingPid)}>
            {t("Terminate PID {0}", row.blockingPid)}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function SessionCard({ row }: { row: DatabaseSessionChain }) {
  const t = useT();

  return (
    <Collapsible>
      <Card size="sm">
        <CardHeader className="border-b">
          <CardTitle>
            {t("PID {0} blocked by PID {1}", row.blockedPid, row.blockingPid)}
          </CardTitle>
          <CardDescription className="flex items-center gap-2">
            <Badge variant="secondary">{row.blockedWaitEventType || "Unknown"}</Badge>
            <span>{row.databaseName}</span>
          </CardDescription>
          <CardAction>
            <TerminateButton row={row} />
          </CardAction>
        </CardHeader>

        <CardContent>
          <div className="mx-auto grid w-full max-w-xl grid-cols-[1fr_auto_1fr] items-start gap-16">
            <SessionSide
              pid={row.blockedPid}
              appName={row.blockedApplicationName}
              user={row.blockedUser}
              state={row.blockedState}
              txAge={row.blockedTransactionAgeSeconds}
              queryAge={row.blockedQueryAgeSeconds}
              align="end"
            />
            <ArrowLeftIcon className="text-muted-foreground mt-1 size-4" />
            <SessionSide
              pid={row.blockingPid}
              appName={row.blockingApplicationName}
              user={row.blockingUser}
              state={row.blockingState}
              txAge={row.blockingTransactionAgeSeconds}
              queryAge={row.blockingQueryAgeSeconds}
              align="start"
            />
          </div>
        </CardContent>

        <div className="border-t px-3 py-2">
          <CollapsibleTrigger className="text-muted-foreground hover:text-foreground flex w-full cursor-pointer items-center justify-center gap-1.5 rounded-md py-1 text-xs transition-colors">
            <CodeIcon className="size-3" />
            <span>{t("Queries")}</span>
            <ChevronDownIcon className="size-3 transition-transform [[data-panel-open]_&]:rotate-180" />
          </CollapsibleTrigger>
          <CollapsibleContent>
            <div className="mt-2 grid grid-cols-2 gap-4">
              <QueryBlock label={t("Blocked query")} query={row.blockedQueryPreview} />
              <QueryBlock label={t("Blocking query")} query={row.blockingQueryPreview} />
            </div>
          </CollapsibleContent>
        </div>
      </Card>
    </Collapsible>
  );
}

function SkeletonCard() {
  return (
    <Card size="sm">
      <CardHeader className="border-b">
        <Skeleton className="h-4 w-48" />
        <Skeleton className="h-3 w-32" />
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-[1fr_auto_1fr] items-center gap-4">
          <div className="flex flex-col gap-2">
            <Skeleton className="h-5 w-16" />
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-3 w-20" />
          </div>
          <Skeleton className="size-4 rounded-full" />
          <div className="flex flex-col gap-2">
            <Skeleton className="h-5 w-16" />
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-3 w-20" />
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function EmptyState() {
  const t = useT();

  return (
    <div className="flex flex-col items-center justify-center py-20">
      <div className="bg-muted flex size-14 items-center justify-center rounded-full">
        <ShieldCheckIcon className="text-muted-foreground size-7" />
      </div>
      <h3 className="mt-4 text-sm font-medium">{t("No blocked sessions")}</h3>
      <p className="text-muted-foreground mt-1 text-xs">{t("All database sessions are running clean")}</p>
    </div>
  );
}

function StatusBar({
  count,
  isFetching,
  onRefresh,
}: {
  count: number;
  isFetching: boolean;
  onRefresh: () => void;
}) {
  const t = useT();

  return (
    <div className="flex items-center justify-between">
      <span className="text-muted-foreground text-sm">
        {t("{0} blocked {1}", count, count === 1 ? "session" : "sessions")}
      </span>
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-1.5">
          <span className="relative flex size-1.5">
            <span className="absolute inline-flex size-full animate-ping rounded-full bg-emerald-400 opacity-75" />
            <span className="relative inline-flex size-1.5 rounded-full bg-emerald-500" />
          </span>
          <span className="text-muted-foreground text-xs">{t("Live")}</span>
        </div>
        <Tooltip>
          <TooltipTrigger
            render={
              <Button size="icon-xs" variant="outline" onClick={onRefresh} disabled={isFetching} />
            }
          >
            <RefreshCwIcon className={`size-3 ${isFetching ? "animate-spin" : ""}`} />
          </TooltipTrigger>
          <TooltipContent>{t("Refresh")}</TooltipContent>
        </Tooltip>
      </div>
    </div>
  );
}

export function DatabaseSessionsPage() {
  const t = useT();

  const query = useQuery({
    queryKey,
    queryFn: () => apiService.databaseSessionService.listBlocked(),
    refetchInterval: 15000,
  });

  const rows = useMemo(() => query.data?.items ?? [], [query.data?.items]);

  return (
    <AdminPageLayout>
      <PageHeader
        title={t("Database Sessions")}
        description={t("Inspect lock contention and manually terminate blocking database sessions")}
      />
      <div className="p-4">
        <StatusBar
          count={rows.length}
          isFetching={query.isFetching}
          onRefresh={() => query.refetch()}
        />
        {query.isLoading ? (
          <div className="flex flex-col gap-3">
            <SkeletonCard />
            <SkeletonCard />
          </div>
        ) : rows.length === 0 ? (
          <EmptyState />
        ) : (
          <div className="flex flex-col gap-3">
            {rows.map((row) => (
              <SessionCard key={`${row.blockedPid}-${row.blockingPid}`} row={row} />
            ))}
          </div>
        )}
      </div>
    </AdminPageLayout>
  );
}
