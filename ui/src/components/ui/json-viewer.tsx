import { useTheme } from "@/components/theme-provider";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import {
  faCheck,
  faChevronDown,
  faChevronRight,
  faCopy,
} from "@fortawesome/pro-regular-svg-icons";
import React, { useMemo, useState } from "react";
import { Icon } from "./icons";

interface ShikiJsonViewerProps {
  data: any;
  className?: string;
  initiallyExpanded?: boolean;
}

interface CollapsibleNodeProps {
  name: string | number | null;
  value: any;
  isRoot?: boolean;
  initiallyExpanded?: boolean;
  withComma?: boolean;
}

function CollapsibleNode({
  name,
  value,
  isRoot = false,
  initiallyExpanded = false,
  withComma = false,
}: CollapsibleNodeProps) {
  const [isExpanded, setIsExpanded] = useState(initiallyExpanded || isRoot);
  const isObject = value !== null && typeof value === "object";
  const isArray = Array.isArray(value);

  const toggleExpand = (e: React.MouseEvent) => {
    e.stopPropagation();
    setIsExpanded(!isExpanded);
  };

  const displayName =
    name !== null ? (
      <span className="text-vitess-node">
        {typeof name === "string" ? `"${name}"` : name}
      </span>
    ) : null;

  if (!isObject) {
    let valueDisplay;
    if (typeof value === "string") {
      valueDisplay = (
        <span className="text-vitess-string">&quot;{value}&quot;</span>
      );
    } else if (typeof value === "number") {
      valueDisplay = <span className="text-vitess-number">{value}</span>;
    } else if (typeof value === "boolean") {
      valueDisplay = (
        <span className="text-vitess-number">{String(value)}</span>
      );
    } else if (value === null) {
      valueDisplay = <span className="text-vitess-number">null</span>;
    } else {
      valueDisplay = <span>{String(value)}</span>;
    }

    return (
      <div className="px-2 py-0.5 hover:bg-muted rounded flex">
        {displayName && (
          <>
            {displayName}
            <span className="text-foreground mx-0.5">:</span>
          </>
        )}
        {valueDisplay}
        {withComma && <span className="text-foreground">,</span>}
      </div>
    );
  }

  const childrenCount = isArray ? value.length : Object.keys(value).length;
  const summary = isArray ? `[${childrenCount}]` : `{${childrenCount}}`;

  return (
    <div className="hover:bg-muted/30 rounded">
      <div
        className="flex items-center px-2 py-0.5 cursor-pointer"
        onClick={toggleExpand}
      >
        <Icon
          icon={isExpanded ? faChevronDown : faChevronRight}
          className="size-3 mr-1 text-muted-foreground"
        />

        {displayName && (
          <>
            {displayName}
            <span className="text-foreground mx-0.5">:</span>
          </>
        )}

        <span className="text-muted-foreground text-xs">
          {isExpanded ? (isArray ? "[" : "{") : summary}
        </span>

        {!isExpanded && withComma && <span className="text-foreground">,</span>}
      </div>

      {isExpanded && (
        <div className="ml-4 border-l border-border pl-2">
          {isArray
            ? // Handle array rendering
              value.map((item: any, index: number) => (
                <CollapsibleNode
                  key={index}
                  name={null}
                  value={item}
                  withComma={index < value.length - 1}
                />
              ))
            : // Handle object rendering
              Object.entries(value).map(([key, val], index, arr) => (
                <CollapsibleNode
                  key={key}
                  name={key}
                  value={val}
                  withComma={index < arr.length - 1}
                />
              ))}
          <div className="px-2 py-0.5">
            <span className="text-muted-foreground">{isArray ? "]" : "}"}</span>
            {withComma && <span className="text-foreground">,</span>}
          </div>
        </div>
      )}
    </div>
  );
}

export function ShikiJsonViewer({
  data,
  className = "",
  initiallyExpanded = false,
}: ShikiJsonViewerProps) {
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  // Format JSON data as a string for copying
  const jsonString = useMemo(() => {
    if (!data) {
      return "";
    }

    try {
      return JSON.stringify(data, null, 2);
    } catch (err) {
      console.error("Error stringifying JSON:", err);
      setError("Failed to format JSON data");
      return String(data);
    }
  }, [data]);

  // Handle Copy to Clipboard
  const handleCopyToClipboard = async () => {
    if (!jsonString) {
      return;
    }

    try {
      await navigator.clipboard.writeText(jsonString);
      setCopied(true);

      // Reset copied state after 2 seconds
      setTimeout(() => {
        setCopied(false);
      }, 2000);
    } catch (err) {
      console.error("Failed to copy to clipboard:", err);
    }
  };

  // Handle states
  if (!data) {
    return (
      <div className="text-muted-foreground italic">No data available</div>
    );
  }

  if (error) {
    return <div className="text-red-500 text-sm">{error}</div>;
  }

  // Render the collapsible JSON viewer
  return (
    <div
      className={cn(
        "relative rounded-md overflow-hidden border border-border bg-card",
        className,
      )}
    >
      <div className="absolute top-2 right-2 z-10">
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="size-8"
                onClick={handleCopyToClipboard}
              >
                {copied ? (
                  <Icon icon={faCheck} className="size-4 text-green-500" />
                ) : (
                  <Icon icon={faCopy} className="size-4" />
                )}
                <span className="sr-only">Copy JSON</span>
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              <p>{copied ? "Copied!" : "Copy JSON"}</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>
      <div className="p-2 font-mono text-sm overflow-auto">
        <CollapsibleNode
          name={null}
          value={data}
          isRoot={true}
          initiallyExpanded={initiallyExpanded}
        />
      </div>
    </div>
  );
}

/**
 * Component for rendering a diff between two JSON objects using ShikiJsonViewer
 */
export function ShikiJsonDiffViewer({
  oldData,
  newData,
  title = { old: "Previous", new: "Current" },
}: {
  oldData: any;
  newData: any;
  title?: { old: string; new: string };
}) {
  const { theme } = useTheme();

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
      <div
        className={`p-2 rounded-md ${theme === "dark" ? "bg-red-950/30" : "bg-red-50"}`}
      >
        <div className="text-xs font-medium text-muted-foreground mb-1">
          {title.old}
        </div>
        <ShikiJsonViewer data={oldData} />
      </div>
      <div
        className={`p-2 rounded-md ${theme === "dark" ? "bg-green-950/30" : "bg-green-50"}`}
      >
        <div className="text-xs font-medium text-muted-foreground mb-1">
          {title.new}
        </div>
        <ShikiJsonViewer data={newData} />
      </div>
    </div>
  );
}
