import Highlight from "@/components/ui/highlight";
import { cn } from "@/lib/utils";
import { memo } from "react";

interface LogLineProps {
  timestamp: string;
  level: string;
  service?: string;
  method?: string;
  path?: string;
  message: string;
  caller: string;
  searchTerm?: string;
}

function LogLine({
  timestamp,
  level,
  service,
  method,
  path,
  message,
  caller,
  searchTerm,
}: LogLineProps) {
  const getLevelColor = (level: string) => {
    switch (level.toLowerCase()) {
      case "debug":
        return "text-blue-500";
      case "info":
        return "text-green-500";
      case "warn":
        return "text-yellow-500";
      case "error":
        return "text-red-500";
      default:
        return "text-muted-foreground";
    }
  };

  const methodColor = (method: string) => {
    switch (method.toLowerCase()) {
      case "get":
        return "text-blue-500";
      case "post":
        return "text-green-500";
      case "put":
        return "text-yellow-500";
      case "delete":
        return "text-red-500";
      default:
        return "text-muted-foreground";
    }
  };

  return (
    <div className="flex gap-1 font-mono text-sm">
      <span className="text-cyan-500">{timestamp}</span>
      <span className="whitespace-nowrap">
        [
        <span className={cn("uppercase", getLevelColor(level), "uppercase")}>
          {level}
        </span>
        ]
      </span>
      {service && (
        <span className="whitespace-nowrap">
          [<span className="text-purple-500">{service}</span>]
        </span>
      )}
      {method && (
        <span className="whitespace-nowrap uppercase">
          [<span className={methodColor(method)}>{method}</span>]
        </span>
      )}
      {path && (
        <span className="whitespace-nowrap">
          [<span className="text-orange-500">{path}</span>]
        </span>
      )}
      <Highlight
        text={message}
        highlight={searchTerm}
        className="text-foreground flex-1"
      />
      <span className="text-fuchsia-500 whitespace-nowrap">({caller})</span>
    </div>
  );
}

LogLine.displayName = "LogLine";

export default memo(LogLine);
