/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
 */

import { useQuery } from "@tanstack/react-query";
import {
  lazy,
  useCallback,
  useDeferredValue,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

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
import { cn } from "@/lib/utils";
import { api } from "@/services/api";
import { useContainerLogStore } from "@/stores/docker-store";
import { ArrowDown } from "lucide-react";
import { LazyLoader } from "../error-boundary";
import { Kbd } from "../ui/kbd";
import { Skeleton } from "../ui/skeleton";
import { ContainerLogActions } from "./container-log-actions";
import { ContainerLogControls } from "./container-log-controls";

const ContainerLogContent = lazy(() => import("./container-log-content"));

interface ContainerLogsDialogProps {
  containerId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ContainerLogsDialog({
  containerId,
  open,
  onOpenChange,
}: ContainerLogsDialogProps) {
  const [searchTerm] = useContainerLogStore.use("searchTerm");
  const tail = useContainerLogStore.get("tail");
  const autoRefresh = useContainerLogStore.get("autoRefresh");
  const wrap = useContainerLogStore.get("wrap");
  const [follow, setFollow] = useContainerLogStore.use("follow");

  const scrollAreaRef = useRef<HTMLDivElement>(null);
  const isAtBottomRef = useRef(true);
  const [isAtBottom, setIsAtBottom] = useState(true);

  const deferredSearch = useDeferredValue(searchTerm);
  const needle = deferredSearch.trim();
  const lowerNeedle = needle.toLowerCase();

  const refreshInterval =
    open && autoRefresh && follow && isAtBottom && !needle ? 5000 : false;

  const {
    data: logs,
    refetch,
    isLoading,
    isRefetching,
    error,
  } = useQuery({
    queryKey: ["docker", "container-logs", containerId, tail],
    queryFn: () => api.docker.getContainerLogs(containerId, tail, false),
    enabled: open,
    refetchOnWindowFocus: false,
    refetchInterval: refreshInterval,
  });

  // Track scroll position & auto-follow
  useEffect(() => {
    const el = scrollAreaRef.current;
    if (!el) return;

    const onScroll = () => {
      const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 24;
      isAtBottomRef.current = nearBottom;
      setIsAtBottom(nearBottom);
      if (!nearBottom) setFollow(false);
    };

    el.addEventListener("scroll", onScroll, { passive: true });
    return () => el.removeEventListener("scroll", onScroll);
  }, [setFollow]);

  // Auto-scroll to bottom on new logs if following
  useEffect(() => {
    const el = scrollAreaRef.current;
    if (!el || !logs || !follow) return;
    el.scrollTop = el.scrollHeight;
  }, [logs, follow]);

  // Fast filtering (simple, case-insensitive)
  const filteredLogs = useMemo(() => {
    if (!logs) return [] as string[];
    if (!needle) return logs;
    return logs.filter((line) => line.toLowerCase().includes(lowerNeedle));
  }, [logs, needle, lowerNeedle]);

  const filteredCount = filteredLogs.length;

  // Mini derived metrics for overview & chart
  const handleJumpToLatest = useCallback(() => {
    const el = scrollAreaRef.current;
    if (!el) return;
    el.scrollTop = el.scrollHeight;
    setFollow(true);
  }, [setFollow]);

  const livePill = autoRefresh && follow && isAtBottom && !needle;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        withClose={false}
        className={cn(
          "sm:max-w-[1000px] max-h-[85vh] w-full",
          wrap && "sm:max-w-[1000px]",
        )}
      >
        <DialogHeader>
          <div className="flex items-start justify-between gap-3">
            <div>
              <DialogTitle className="flex items-center gap-2">
                Container Logs
                <Badge variant="purple" className="font-mono">
                  {containerId.slice(0, 12)}
                </Badge>
                {livePill && (
                  <span className="inline-flex items-center gap-1 text-xs text-emerald-600 dark:text-emerald-400">
                    <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
                    live
                  </span>
                )}
              </DialogTitle>
              <DialogDescription>
                View, filter, and export recent logs. Tail controls how many
                lines are fetched.
              </DialogDescription>
            </div>

            {/* Actions */}
            <ContainerLogActions
              refetch={refetch}
              isRefetching={isRefetching}
              logs={logs}
              filteredLogs={filteredLogs}
              needle={needle}
              containerId={containerId}
            />
          </div>
        </DialogHeader>

        <ContainerLogControls
          logs={logs}
          filteredCount={filteredCount}
          handleJumpToLatest={handleJumpToLatest}
        />
        <div className="relative">
          {!follow && (
            <Button
              variant="secondary"
              size="sm"
              className="absolute right-3 -top-2 z-10 shadow"
              onClick={handleJumpToLatest}
            >
              <ArrowDown className="mr-2 size-4" />
              Jump to latest
            </Button>
          )}
          <LazyLoader fallback={<ContainerLogContentSkeleton />}>
            <ContainerLogContent
              filteredCount={filteredCount}
              needle={needle}
              filteredLogs={filteredLogs}
              scrollAreaRef={scrollAreaRef}
              isLoading={isLoading}
              error={error}
            />
          </LazyLoader>
        </div>
        <DialogFooter className="mt-1 flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground p-2">
          <div className="flex items-center gap-2">
            {autoRefresh && (
              <span className="opacity-80">Auto-refreshing every 5s</span>
            )}
            {!isAtBottom && (
              <span className="opacity-80">(Paused while scrolled up)</span>
            )}
            {needle && (
              <span className="opacity-80">
                (Refresh paused while filtering)
              </span>
            )}
          </div>
          <div className="opacity-70">
            Tip: Press <Kbd>Esc</Kbd> to clear search.
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ContainerLogContentSkeleton() {
  return (
    <div className="flex flex-col gap-1 p-2 h-[520px] overflow-y-auto">
      {Array.from({ length: 100 }).map((_, index) => (
        <Skeleton key={index} className="h-4 w-full shrink-0" />
      ))}
    </div>
  );
}
