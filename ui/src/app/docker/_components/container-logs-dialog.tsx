/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md
 */

import { dockerAPI } from "@/services/docker";
import { useQuery } from "@tanstack/react-query";
import {
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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ScrollArea, ScrollAreaShadow } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

import Highlight from "@/components/ui/highlight";
import { cn } from "@/lib/utils";
import {
  ArrowDown,
  Check,
  Clipboard,
  Download,
  Layers,
  RefreshCw,
  ScanText,
  Search,
} from "lucide-react";

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
  const [searchTerm, setSearchTerm] = useState("");
  const [tail, setTail] = useState<string>("100");
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [wrap, setWrap] = useState(true);
  const [follow, setFollow] = useState(true);
  const [copied, setCopied] = useState(false);

  // The scrollable element; ScrollArea forwards the ref to its viewport in shadcn
  const scrollAreaRef = useRef<HTMLDivElement>(null);
  const isAtBottomRef = useRef(true);

  const deferredSearch = useDeferredValue(searchTerm);

  const {
    data: logs,
    refetch,
    isLoading,
    isRefetching,
    error,
  } = useQuery({
    queryKey: ["docker", "container-logs", containerId, tail],
    queryFn: () => dockerAPI.getContainerLogs(containerId, tail, false),
    enabled: open,
    refetchOnWindowFocus: false,
    refetchInterval: open && autoRefresh ? 5000 : false,
  });

  useEffect(() => {
    const el = scrollAreaRef.current;
    if (!el) return;

    const onScroll = () => {
      const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 24;
      isAtBottomRef.current = nearBottom;
      if (!nearBottom) setFollow(false);
    };

    el.addEventListener("scroll", onScroll, { passive: true });
    return () => el.removeEventListener("scroll", onScroll);
  }, []);

  useEffect(() => {
    const el = scrollAreaRef.current;
    if (!el || !logs || !follow) return;
    el.scrollTop = el.scrollHeight;
  }, [logs, follow]);

  const filteredLogs = useMemo(() => {
    if (!logs) return [];
    if (!deferredSearch) return logs;
    const needle = deferredSearch.toLowerCase();
    return logs.filter((line) => line.toLowerCase().includes(needle));
  }, [logs, deferredSearch]);

  const filteredCount = filteredLogs.length;

  const handleRefresh = useCallback(() => {
    void refetch();
  }, [refetch]);

  const handleDownload = useCallback(() => {
    if (!logs?.length) return;
    const blob = new Blob([logs.join("\n")], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `container-${containerId.slice(0, 12)}-logs.txt`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }, [logs, containerId]);

  const handleCopyAll = useCallback(async () => {
    if (!logs?.length) return;
    await navigator.clipboard.writeText(logs.join("\n"));
    setCopied(true);
    const t = setTimeout(() => setCopied(false), 1200);
    return () => clearTimeout(t);
  }, [logs]);

  const handleJumpToLatest = useCallback(() => {
    const el = scrollAreaRef.current;
    if (!el) return;
    el.scrollTop = el.scrollHeight;
    setFollow(true);
  }, []);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        withClose={false}
        className={cn("max-w-4xl max-h-[85vh]", wrap && "w-auto")}
      >
        <DialogHeader>
          <div className="flex items-start justify-between gap-3">
            <div>
              <DialogTitle className="flex items-center gap-2">
                Container Logs
                <Badge variant="purple" className="font-mono">
                  {containerId.slice(0, 12)}
                </Badge>
              </DialogTitle>
              <DialogDescription>
                View, filter, and export recent logs. Tail controls how many
                lines are fetched.
              </DialogDescription>
            </div>

            <div className="flex items-center gap-2">
              <TooltipProvider delayDuration={200}>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      size="icon"
                      variant="outline"
                      onClick={handleRefresh}
                      aria-label="Refresh logs"
                    >
                      <RefreshCw
                        className={cn(
                          "size-4",
                          isRefetching ? "animate-spin" : "",
                        )}
                      />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Refresh now</TooltipContent>
                </Tooltip>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      size="icon"
                      variant="outline"
                      onClick={handleDownload}
                      aria-label="Download logs"
                    >
                      <Download className="size-4" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Download as .txt</TooltipContent>
                </Tooltip>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      size="icon"
                      variant="outline"
                      onClick={handleCopyAll}
                      aria-label="Copy logs to clipboard"
                    >
                      {copied ? (
                        <Check className="h-4 w-4" />
                      ) : (
                        <Clipboard className="size-4" />
                      )}
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>
                    {copied ? "Copied!" : "Copy all"}
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            </div>
          </div>
        </DialogHeader>

        <div className="flex flex-col p-2 gap-3">
          {/* Controls row */}
          <div className="flex flex-wrap items-center gap-3">
            <div className="relative grow">
              <Input
                icon={<Search className="size-4" />}
                aria-label="Search logs"
                placeholder="Search logs…"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Escape") setSearchTerm("");
                }}
                className="pl-8"
              />
            </div>

            <div className="flex items-center gap-2">
              <Label htmlFor="tail">Tail</Label>
              <Select value={tail} onValueChange={(v) => setTail(v)}>
                <SelectTrigger id="tail" className="w-[140px]">
                  <ScanText className="size-4 mr-2" />
                  <SelectValue placeholder="Tail lines" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="100">100 lines</SelectItem>
                  <SelectItem value="500">500 lines</SelectItem>
                  <SelectItem value="1000">1,000 lines</SelectItem>
                  <SelectItem value="5000">5,000 lines</SelectItem>
                  <SelectItem value="all">All</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <Separator orientation="vertical" className="h-6 hidden md:block" />

            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <Switch
                  id="auto-refresh"
                  checked={autoRefresh}
                  onCheckedChange={setAutoRefresh}
                />
                <Label htmlFor="auto-refresh">Auto-refresh</Label>
              </div>
              <div className="flex items-center gap-2">
                <Switch id="wrap" checked={wrap} onCheckedChange={setWrap} />
                <Label htmlFor="wrap">Wrap lines</Label>
              </div>
              <div className="flex items-center gap-2">
                <Switch
                  id="follow"
                  checked={follow}
                  onCheckedChange={(v) => {
                    setFollow(v);
                    if (v) handleJumpToLatest();
                  }}
                />
                <Label htmlFor="follow">Follow tail</Label>
              </div>
            </div>
          </div>

          {/* Log viewport */}
          <div>
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

            <ScrollArea ref={scrollAreaRef} className="h-[520px]">
              <div
                className={cn(
                  "min-h-full font-mono text-xs pb-4",
                  wrap ? "whitespace-pre-wrap break-words" : "whitespace-pre",
                  "bg-background text-muted-foreground",
                )}
                aria-live="polite"
                role="log"
              >
                {isLoading ? (
                  <div className="space-y-2">
                    <div className="h-3 w-3/4 animate-pulse rounded bg-emerald-600/30" />
                    <div className="h-3 w-2/3 animate-pulse rounded bg-emerald-600/30" />
                    <div className="h-3 w-1/2 animate-pulse rounded bg-emerald-600/30" />
                  </div>
                ) : error ? (
                  <div className="text-red-400">
                    Failed to load logs.{" "}
                    <span className="opacity-70">Try refresh.</span>
                  </div>
                ) : filteredCount > 0 ? (
                  <ul className="space-y-1">
                    {filteredLogs.map((line, idx) => (
                      <li key={idx} className="flex items-start gap-2">
                        <Layers className="mt-0.5 h-3.5 w-3.5 opacity-60" />
                        <span>
                          <Highlight text={line} highlight={deferredSearch} />
                        </span>
                      </li>
                    ))}
                  </ul>
                ) : (
                  <div>No logs found.</div>
                )}
              </div>
              <ScrollAreaShadow />
            </ScrollArea>
          </div>
        </div>

        <DialogFooter className="mt-1 flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground p-2">
          <div className="flex items-center gap-2">
            {autoRefresh && (
              <span className="opacity-80">Auto-refreshing every 5s</span>
            )}
          </div>
          <div className="opacity-70">
            Tip: Press <kbd className="rounded border px-1.5">Esc</kbd> to clear
            search.
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
