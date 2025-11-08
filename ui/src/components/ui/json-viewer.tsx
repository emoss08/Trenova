import { Button } from "@/components/ui/button";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { cn } from "@/lib/utils";
import type { TableSheetProps } from "@/types/data-table";
import type {
  CollapsibleNodeProps,
  JsonViewerProps,
} from "@/types/json-viewer";
import {
  faChevronDown,
  faChevronRight,
  faCompress,
  faCopy,
  faEllipsis,
  faExpand,
  faMinus,
  faPlus,
  faTable,
} from "@fortawesome/pro-regular-svg-icons";
import React, { useEffect, useMemo, useState } from "react";
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
import { SensitiveValue } from "./sensitive-value";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "./shadcn-table";

interface CollapsibleNodeInternalProps extends CollapsibleNodeProps {
  forceExpanded?: boolean;
  onToggle?: (path: string) => void;
  path?: string;
}

function CollapsibleNode({
  name,
  value,
  isRoot = false,
  initiallyExpanded = false,
  withComma = false,
  forceExpanded,
  onToggle,
  path = "",
}: CollapsibleNodeInternalProps) {
  const [isExpanded, setIsExpanded] = useState(initiallyExpanded || isRoot);
  const isObject = value !== null && typeof value === "object";
  const isArray = Array.isArray(value);

  // Update expanded state when forceExpanded changes
  useEffect(() => {
    if (forceExpanded !== undefined) {
      setIsExpanded(forceExpanded);
    }
  }, [forceExpanded]);

  const toggleExpand = (e: React.MouseEvent) => {
    e.stopPropagation();
    const newExpanded = !isExpanded;
    setIsExpanded(newExpanded);
    if (onToggle && path) {
      onToggle(path);
    }
  };

  const displayName =
    name !== null ? (
      <span className="text-blue-600 dark:text-blue-400 font-medium">
        {typeof name === "string" ? `"${name}"` : name}
      </span>
    ) : null;

  if (!isObject) {
    const valueDisplay = (
      <SensitiveValue
        value={value}
        className="max-w-[350px] truncate overflow-hidden text-ellipsis"
        prefix=" "
      />
    );

    return (
      <div
        className={cn(
          "px-3 py-1 rounded-sm flex items-center transition-all duration-150",
          "hover:bg-muted-foreground/10",
        )}
      >
        {displayName && (
          <>
            {displayName}
            <span className="text-muted-foreground">: </span>
          </>
        )}
        {valueDisplay}
        {withComma && <span className="text-muted-foreground">,</span>}
      </div>
    );
  }

  const childrenCount = isArray ? value.length : Object.keys(value).length;
  const summary = isArray ? `[${childrenCount}]` : `{${childrenCount}}`;

  return (
    <div className="w-full">
      <div
        className={cn(
          "flex items-center px-3 py-1 cursor-pointer rounded-sm transition-all duration-150",
          "hover:bg-muted-foreground/10",
        )}
        onClick={toggleExpand}
      >
        <Icon
          icon={isExpanded ? faChevronDown : faChevronRight}
          className="size-3.5 mr-1.5 text-muted-foreground transition-transform duration-150"
        />

        {displayName && (
          <>
            {displayName}
            <span className="text-muted-foreground">: </span>
          </>
        )}

        <span className="text-muted-foreground text-xs font-medium font-mono">
          {isExpanded ? (isArray ? "[" : "{") : summary}
        </span>

        {!isExpanded && withComma && (
          <span className="text-muted-foreground">,</span>
        )}
      </div>

      {isExpanded && (
        <div className="ml-4 border-l-2 border-border pl-3 py-0.5">
          {isArray
            ? // Handle array rendering
              value.map((item: any, index: number) => (
                <CollapsibleNode
                  key={index}
                  name={null}
                  value={item}
                  withComma={index < value.length - 1}
                  forceExpanded={forceExpanded}
                  onToggle={onToggle}
                  path={`${path}[${index}]`}
                />
              ))
            : // Handle object rendering
              Object.entries(value).map(([key, val], index, arr) => (
                <CollapsibleNode
                  key={key}
                  name={key}
                  value={val}
                  withComma={index < arr.length - 1}
                  forceExpanded={forceExpanded}
                  onToggle={onToggle}
                  path={path ? `${path}.${key}` : key}
                />
              ))}
          <div className="px-3 py-1">
            <span className="text-muted-foreground font-mono">
              {isArray ? "]" : "}"}
            </span>
            {withComma && <span className="text-muted-foreground">,</span>}
          </div>
        </div>
      )}
    </div>
  );
}

export function JsonViewer({
  data,
  className = "",
  initiallyExpanded = false,
}: JsonViewerProps) {
  const [isFullyExpanded, setIsFullyExpanded] = useState(initiallyExpanded);
  const [forceExpanded, setForceExpanded] = useState<boolean | undefined>(
    undefined,
  );

  const handleToggleAll = () => {
    const newExpanded = !isFullyExpanded;
    setIsFullyExpanded(newExpanded);
    setForceExpanded(newExpanded);
    // Reset forceExpanded after a short delay to allow manual control
    setTimeout(() => setForceExpanded(undefined), 100);
  };

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
          "size-full relative rounded-md overflow-hidden",
          "border border-border",
          "bg-muted",
          "shadow-sm",
          className,
        )}
      >
        <div className="absolute top-3 right-3 z-10 flex items-center gap-2">
          <Button
            variant="ghost"
            size="icon"
            onClick={handleToggleAll}
            title={isFullyExpanded ? "Collapse all" : "Expand all"}
          >
            <Icon
              icon={isFullyExpanded ? faCompress : faExpand}
              className="size-4 text-muted-foreground"
            />
          </Button>
          <JsonViewerActions data={data} />
        </div>
        <ScrollArea className="h-full">
          <div className="p-3 font-mono text-sm">
            <CollapsibleNode
              name={null}
              value={data}
              isRoot={true}
              initiallyExpanded={isFullyExpanded}
              forceExpanded={forceExpanded}
            />
          </div>
        </ScrollArea>
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
    return <span className="text-muted-foreground italic">null</span>;
  if (value === undefined)
    return <span className="text-muted-foreground italic">undefined</span>;

  // Handle primitive values
  if (typeof value !== "object") {
    return <SensitiveValue value={value} className="max-w-[450px] truncate" />;
  }

  // Handle arrays and objects
  const isArray = Array.isArray(value);
  const isEmpty = isArray
    ? value.length === 0
    : Object.keys(value).length === 0;

  if (isEmpty) {
    return (
      <span className="text-muted-foreground font-mono">
        {isArray ? "[]" : "{}"}
      </span>
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
        className="px-3 py-1 h-7 text-foreground hover:bg-muted-foreground/10"
      >
        <Icon icon={isExpanded ? faMinus : faPlus} className="size-3.5 mr-2" />
        <span className="text-xs font-medium">
          {isExpanded ? "Collapse" : "Expand"} ({count} {itemLabel})
        </span>
      </Button>

      {isExpanded && (
        <div className="mt-3 pl-4 border-l-2 border-border">
          <Table className="border border-border rounded-md">
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead className="w-1/3 bg-muted font-medium">
                  Key
                </TableHead>
                <TableHead className="bg-muted font-medium">Value</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isArray
                ? value.map((item: any, index: number) => (
                    <TableRow
                      key={`${path}.${index}`}
                      className="hover:bg-muted-foreground/10"
                    >
                      <TableCell className="font-mono text-xs font-medium text-muted-foreground">
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
                    <TableRow
                      key={`${path}.${key}`}
                      className="hover:bg-muted-foreground/10"
                    >
                      <TableCell className="font-mono text-xs font-medium text-blue-600">
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
  const { copy, isCopied } = useCopyToClipboard();
  const [readableDialogOpen, setReadableDialogOpen] = useState(false);

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

    await copy(jsonString, { withToast: true });
  };

  if (error) {
    return <div className="text-red-500 text-sm">{error}</div>;
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="icon">
            <Icon icon={faEllipsis} className="size-4 text-muted-foreground" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent side="bottom" align="end">
          <DropdownMenuLabel>JSON Options</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuItem
              onClick={handleCopyToClipboard}
              className="flex items-center gap-2"
              startContent={
                <Icon icon={faCopy} className="size-4 text-muted-foreground" />
              }
              title={isCopied ? "Copied!" : "Copy JSON"}
              description="Copy formatted JSON to clipboard"
            />
            <DropdownMenuItem
              onClick={() => setReadableDialogOpen(true)}
              className="flex items-center gap-2"
              startContent={
                <Icon icon={faTable} className="size-4 text-muted-foreground" />
              }
              title="Structured View"
              description="Display in tabular format"
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
          <DialogTitle className="flex items-center gap-2">
            Data Structure View <BetaTag />
          </DialogTitle>
          <DialogDescription>
            Structured view of JSON data for easier analysis and inspection
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="p-0">
          <ScrollArea className="flex flex-col overflow-y-auto max-h-[calc(100vh-8.5rem)]">
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead className="w-1/6 bg-muted font-medium">
                    Property
                  </TableHead>
                  <TableHead className="bg-muted font-medium">Value</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {Object.entries(data).map(([key, value]) => (
                  <TableRow key={key} className="hover:bg-muted-foreground/10">
                    <TableCell className="font-mono text-xs font-medium text-blue-600 border-r border-border">
                      {key}
                    </TableCell>
                    <TableCell>
                      <ReadableJsonValue value={value} path={key} />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </ScrollArea>
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}
