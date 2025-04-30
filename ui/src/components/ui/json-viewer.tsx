import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { TableSheetProps } from "@/types/data-table";
import type {
  CollapsibleNodeProps,
  JsonViewerProps,
} from "@/types/json-viewer";
import {
  faChevronDown,
  faChevronRight,
  faEllipsis,
  faMinus,
  faPlus,
} from "@fortawesome/pro-regular-svg-icons";
import React, { useMemo, useState } from "react";
import { toast } from "sonner";
import SuperJSON from "superjson";
import { BetaTag } from "./beta-tag";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "./dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "./dropdown-menu";
import { Icon } from "./icons";
import { ScrollArea } from "./scroll-area";
import { SensitiveBadge } from "./sensitive-badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "./table";

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
      <span className="text-vitess-node font-medium">
        {typeof name === "string" ? `"${name}"` : name}
      </span>
    ) : null;

  if (!isObject) {
    let valueDisplay;
    // Check if value appears to be masked/sensitive data
    const isSensitiveData = typeof value === "string" && /^\*{3,}$/.test(value);

    if (isSensitiveData) {
      valueDisplay = (
        <div className="inline-flex items-center">
          <span className="max-w-[450px] truncate text-vitess-string">
            &quot;{value}&quot;
          </span>
          <SensitiveBadge />
        </div>
      );
    } else if (typeof value === "string") {
      valueDisplay = (
        <span className="max-w-[450px] truncate text-vitess-string">
          &quot;{value}&quot;
        </span>
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
      <div className="px-3 py-1 hover:bg-muted rounded-sm flex items-center transition-colors">
        {displayName && (
          <>
            {displayName}
            <span className="text-foreground mx-1.5">:</span>
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
    <div className="rounded-sm transition-colors overflow-hidden">
      <div
        className="flex items-center px-3 py-1 cursor-pointer"
        onClick={toggleExpand}
      >
        <Icon
          icon={isExpanded ? faChevronDown : faChevronRight}
          className="size-3.5 mr-1.5 text-muted-foreground"
        />

        {displayName && (
          <>
            {displayName}
            <span className="text-foreground mx-1.5">:</span>
          </>
        )}

        <span className="text-muted-foreground text-xs font-medium">
          {isExpanded ? (isArray ? "[" : "{") : summary}
        </span>

        {!isExpanded && withComma && <span className="text-foreground">,</span>}
      </div>

      {isExpanded && (
        <ScrollArea className="ml-4 border-l border-border pl-3 py-0.5 max-h-[calc(100vh-200px)] overflow-y-auto">
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
          <div className="px-3 py-1">
            <span className="text-muted-foreground">{isArray ? "]" : "}"}</span>
            {withComma && <span className="text-foreground">,</span>}
          </div>
        </ScrollArea>
      )}
    </div>
  );
}

export function JsonViewer({
  data,
  className = "",
  initiallyExpanded = false,
}: JsonViewerProps) {
  // Handle states
  if (!data) {
    return (
      <div className="p-4 border border-dashed border-border rounded-md">
        <p className="text-xs text-muted-foreground italic">
          No data available
        </p>
      </div>
    );
  }

  // Render the collapsible JSON viewer
  return (
    <>
      <div
        className={cn(
          "relative rounded-md overflow-hidden border border-border bg-card",
          className,
        )}
      >
        <div className="absolute top-3 right-3 z-10">
          <JsonViewerActions data={data} />
        </div>
        <div className="p-3 font-mono text-sm">
          <CollapsibleNode
            name={null}
            value={data}
            isRoot={true}
            initiallyExpanded={initiallyExpanded}
          />
        </div>
      </div>
    </>
  );
}

export function ReadableJsonValue({
  value,
  path = "",
  level = 0,
}: {
  value: any;
  path?: string;
  level?: number;
}) {
  const [isExpanded, setIsExpanded] = useState(false);

  if (value === null)
    return <span className="text-muted-foreground">null</span>;
  if (value === undefined)
    return <span className="text-muted-foreground">undefined</span>;

  // Handle sensitive data detection
  const isSensitiveData = typeof value === "string" && /^\*{3,}$/.test(value);

  // Handle primitive values
  if (typeof value !== "object") {
    if (isSensitiveData) {
      return (
        <div className="flex items-center">
          <span className="max-w-[450px] truncate text-vitess-string">
            &quot;{value}&quot;
          </span>
          <SensitiveBadge />
        </div>
      );
    } else if (typeof value === "string") {
      return (
        <span className="max-w-[450px] truncate text-vitess-string">
          &quot;{value}&quot;
        </span>
      );
    } else if (typeof value === "number") {
      return <span className="text-vitess-number">{value}</span>;
    } else if (typeof value === "boolean") {
      return <span className="text-vitess-number">{value.toString()}</span>;
    }
    return <span>{String(value)}</span>;
  }

  // Handle arrays and objects
  const isArray = Array.isArray(value);
  const isEmpty = isArray
    ? value.length === 0
    : Object.keys(value).length === 0;

  if (isEmpty) {
    return (
      <span className="text-muted-foreground">{isArray ? "[]" : "{}"}</span>
    );
  }

  const count = isArray ? value.length : Object.keys(value).length;
  const itemLabel = isArray ? "items" : "properties";

  return (
    <>
      <Button
        variant="ghost"
        size="sm"
        onClick={() => setIsExpanded(!isExpanded)}
        className="px-3 py-1 h-7 text-foreground hover:bg-muted"
      >
        <Icon icon={isExpanded ? faMinus : faPlus} className="size-3.5 mr-2" />
        <span className="text-xs font-medium">
          {isExpanded ? "Collapse" : "Expand"} ({count} {itemLabel})
        </span>
      </Button>

      {isExpanded && (
        <div className="mt-3 pl-4 border-l border-border">
          <Table className="border border-border rounded-md">
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead className="w-1/3 bg-muted">Key</TableHead>
                <TableHead className="bg-muted">Value</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isArray
                ? value.map((item: any, index: number) => (
                    <TableRow
                      key={`${path}.${index}`}
                      className="hover:bg-muted"
                    >
                      <TableCell className="font-mono text-xs font-medium">
                        [{index}]
                      </TableCell>
                      <TableCell>
                        <ReadableJsonValue
                          value={item}
                          path={`${path}.${index}`}
                          level={level + 1}
                        />
                      </TableCell>
                    </TableRow>
                  ))
                : Object.entries(value).map(([key, val]) => (
                    <TableRow key={`${path}.${key}`} className="hover:bg-muted">
                      <TableCell className="font-mono text-xs font-medium">
                        {key}
                      </TableCell>
                      <TableCell>
                        <ReadableJsonValue
                          value={val}
                          path={`${path}.${key}`}
                          level={level + 1}
                        />
                      </TableCell>
                    </TableRow>
                  ))}
            </TableBody>
          </Table>
        </div>
      )}
    </>
  );
}

function JsonViewerActions({ data }: { data: any }) {
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [readableDialogOpen, setReadableDialogOpen] = useState(false);

  // Format JSON data as a string for copying
  const jsonString = useMemo(() => {
    if (!data) {
      return "";
    }

    try {
      return SuperJSON.stringify(data);
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

  if (error) {
    return <div className="text-red-500 text-sm">{error}</div>;
  }

  // If copied pop up a toast
  if (copied) {
    toast.success("JSON copied to clipboard");
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            className="size-8 hover:bg-muted rounded-md"
          >
            <Icon icon={faEllipsis} className="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent side="bottom" align="end">
          <DropdownMenuLabel>JSON Options</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuItem
              onClick={handleCopyToClipboard}
              className="flex flex-col items-start"
              title="Copy JSON"
              description="Copy formatted JSON data to clipboard"
            />
            <DropdownMenuItem
              title="View in structured view"
              description="Display data in a structured tabular format"
              onClick={() => setReadableDialogOpen(true)}
              className="flex flex-col items-start"
            />
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
      <JsonViewerDialog
        data={data}
        open={readableDialogOpen}
        onOpenChange={setReadableDialogOpen}
      />
    </>
  );
}

type JsonViewerDialogProps = {
  data: any;
} & TableSheetProps;

export function JsonViewerDialog({
  data,
  open,
  onOpenChange,
}: JsonViewerDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl">
        <DialogHeader>
          <DialogTitle>
            Data Structure View <BetaTag />
          </DialogTitle>
          <DialogDescription>
            Structured view of JSON data for easier analysis and inspection
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="p-0">
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead className="w-1/3 bg-muted font-medium">
                  Property
                </TableHead>
                <TableHead className="bg-muted font-medium">Value</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {Object.entries(data).map(([key, value]) => (
                <TableRow key={key} className="hover:bg-muted">
                  <TableCell className="font-mono text-xs font-medium border-r border-border/50">
                    {key}
                  </TableCell>
                  <TableCell>
                    <ReadableJsonValue value={value} path={key} />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}
