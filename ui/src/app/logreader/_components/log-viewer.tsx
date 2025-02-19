import { Virtuoso } from "react-virtuoso";
import LogLine from "./log-line";

export interface ParsedLog {
  timestamp: string;
  level: string;
  service?: string;
  method?: string;
  path?: string;
  message: string;
  caller: string;
}

interface VirtualLogViewerProps {
  logs: ParsedLog[];
  lineHeight?: number;
  autoScroll?: boolean;
  searchTerm?: string;
}

export function VirtualLogViewer({
  logs,
  autoScroll = true,
  searchTerm = "",
}: VirtualLogViewerProps) {
  return (
    <Virtuoso
      style={{ height: "500px", width: "100%" }}
      data={logs}
      followOutput={autoScroll}
      itemContent={(index, log) => (
        <div className="flex items-center hover:bg-muted">
          <div className="w-12 select-none text-muted-foreground text-xs pr-4 text-right">
            {index + 1}
          </div>
          <LogLine
            timestamp={log.timestamp}
            level={log.level}
            service={log.service}
            method={log.method}
            path={log.path}
            message={log.message}
            caller={log.caller}
            searchTerm={searchTerm}
          />
        </div>
      )}
    />
  );
}
