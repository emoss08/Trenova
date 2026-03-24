import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import { useSamsaraSyncStore } from "@/stores/samsara-sync";
import type { WorkerSyncLogLevel } from "@/types/samsara";
import { CopyIcon, TerminalIcon, Trash2Icon } from "lucide-react";
import { toast } from "sonner";

function getLogLevelStyles(level: WorkerSyncLogLevel): string {
  switch (level) {
    case "success":
      return "text-emerald-500";
    case "warn":
      return "text-amber-500";
    case "error":
      return "text-red-500";
    case "debug":
      return "text-purple-500";
    case "info":
      return "text-blue-500";
    default:
      return "text-foreground";
  }
}
function formatTerminalTimestamp(unixTimestamp: number): string {
  return new Date(unixTimestamp).toLocaleTimeString([], {
    hour12: false,
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

export default function RunConsole({ isWorkflowRunning }: { isWorkflowRunning: boolean }) {
  const [logLines, setLogLines] = useSamsaraSyncStore.use("logLines");

  const handleCopyLogs = async () => {
    try {
      await navigator.clipboard.writeText(
        logLines
          .map(
            (line) =>
              `[${formatTerminalTimestamp(line.ts)}] [${line.level.toUpperCase()}] [${line.source}] ${line.message}`,
          )
          .join("\n"),
      );
      toast.success("Console logs copied");
    } catch {
      toast.error("Failed to copy logs");
    }
  };

  return (
    <div className="overflow-hidden rounded-lg border border-border bg-background">
      <div className="flex items-center justify-between border-b border-border bg-sidebar px-3 py-2">
        <div className="inline-flex items-center gap-2 text-xs font-medium text-foreground">
          <TerminalIcon className="size-3.5" />
          Run Console
          {isWorkflowRunning && (
            <Badge variant="active" className="ml-2">
              Live
            </Badge>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Button
            size="sm"
            variant="outline"
            onClick={handleCopyLogs}
            disabled={logLines.length === 0}
          >
            <CopyIcon className="size-3.5" />
            Copy
          </Button>
          <Button
            size="sm"
            variant="destructive"
            onClick={() => setLogLines([])}
            disabled={logLines.length === 0}
          >
            <Trash2Icon className="size-3.5" />
            Clear
          </Button>
        </div>
      </div>
      <ScrollArea className="h-64 px-3 py-2 font-mono text-xs leading-5">
        {logLines.length === 0 ? (
          <p className="text-foreground/50">
            No log lines yet. Start a sync to stream workflow updates.
          </p>
        ) : (
          <div className="space-y-0.5">
            {logLines.map((line) => (
              <div key={line.id} className="grid grid-cols-[62px_56px_68px_1fr] items-start gap-2">
                <span className="text-foreground/50">{formatTerminalTimestamp(line.ts)}</span>
                <span className={cn("font-semibold uppercase", getLogLevelStyles(line.level))}>
                  {line.level}
                </span>
                <span className="text-foreground">{line.source}</span>
                <span className="text-foreground">{line.message}</span>
              </div>
            ))}
          </div>
        )}
      </ScrollArea>
    </div>
  );
}
