import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Icon } from "@/components/ui/icons";
import { ChangeDiffDialog } from "@/components/ui/json-diff-viewer";
import { JsonViewer } from "@/components/ui/json-viewer";
import { Separator } from "@/components/ui/separator";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  faArrowUpRightFromSquare,
  faChevronDown,
  faChevronRight,
  faEllipsis,
  faMinus,
  faPlus,
} from "@fortawesome/pro-solid-svg-icons";
import { useMemo, useState } from "react";
import { SuperJSON } from "superjson";

/**
 * Component for displaying a collapsible data section with a consistent header
 */
export function DataSection({
  title,
  description,
  children,
  defaultCollapsed = false,
}: {
  title: string;
  description: string;
  children: React.ReactNode;
  defaultCollapsed?: boolean;
}) {
  const [isOpen, setIsOpen] = useState(!defaultCollapsed);

  return (
    <Card>
      <Collapsible open={isOpen} onOpenChange={setIsOpen}>
        <div className="flex items-center">
          <CollapsibleTrigger asChild>
            <CardHeader className="pb-2 cursor-pointer flex-1">
              <div className="flex items-center">
                <Icon
                  icon={isOpen ? faChevronDown : faChevronRight}
                  className="mr-2 h-4 w-4 text-muted-foreground"
                />
                <div>
                  <CardTitle className="text-base">{title}</CardTitle>
                  <CardDescription>{description}</CardDescription>
                </div>
              </div>
            </CardHeader>
          </CollapsibleTrigger>
        </div>
        <CollapsibleContent>
          <CardContent>{children}</CardContent>
        </CollapsibleContent>
        {!isOpen && <Separator className="mb-4" />}
      </Collapsible>
    </Card>
  );
}

/**
 * Component to display changes between previous and current states
 */
export function ChangesContent({
  changes,
}: {
  changes?: Record<string, { from: any; to: any }>;
}) {
  if (!changes || Object.keys(changes).length === 0) {
    return <p className="text-muted-foreground italic">No changes recorded</p>;
  }

  return (
    <div className="space-y-4">
      {Object.entries(changes).map(([key, change]) => {
        const hasFrom = change.from !== undefined && change.from !== null;
        const hasTo = change.to !== undefined && change.to !== null;

        return (
          <Collapsible key={key} defaultOpen={true}>
            <div className="border rounded-md shadow-sm">
              <CollapsibleTrigger asChild>
                <div className="flex items-center p-3 cursor-pointer hover:bg-muted transition-colors">
                  <Icon
                    icon={faChevronDown}
                    className="mr-2 h-3 w-3 text-muted-foreground"
                  />
                  <h3 className="text-sm font-medium">{key}</h3>
                </div>
              </CollapsibleTrigger>
              <CollapsibleContent>
                <Separator />
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3 p-3">
                  <div className="p-2 bg-red-50 dark:bg-red-950/30 rounded-md border border-red-200 dark:border-red-900/50">
                    <div className="text-xs font-medium text-muted-foreground mb-1">
                      Previous Version
                    </div>
                    {hasFrom ? (
                      <JsonViewer data={change.from} />
                    ) : (
                      <p className="text-xs text-muted-foreground italic p-2">
                        null
                      </p>
                    )}
                  </div>
                  <div className="p-2 bg-green-50 dark:bg-green-950/30 rounded-md border border-green-200 dark:border-green-900/50">
                    <div className="text-xs font-medium text-muted-foreground mb-1">
                      Current Version
                    </div>
                    {hasTo ? (
                      <JsonViewer data={change.to} />
                    ) : (
                      <p className="text-xs text-muted-foreground italic p-2">
                        null
                      </p>
                    )}
                  </div>
                </div>
              </CollapsibleContent>
            </div>
          </Collapsible>
        );
      })}
    </div>
  );
}

// Constants for size thresholds
const DATA_SIZE_THRESHOLD = 10000; // ~10KB JSON size when stringified

/**
 * Display value component for changes table that supports complex values
 */
function DisplayValue({ value }: { value: any }) {
  const [isExpanded, setIsExpanded] = useState(false);

  // Handle null values
  if (value === null || value === undefined) {
    return <span className="text-muted-foreground italic">null</span>;
  }

  // Handle primitive values
  if (typeof value !== "object") {
    if (typeof value === "string") {
      return (
        <span>{value.length > 50 ? `${value.slice(0, 50)}...` : value}</span>
      );
    }
    if (typeof value === "number") {
      return <span>{value}</span>;
    }
    if (typeof value === "boolean") {
      return (
        <span className={value ? "text-green-600" : "text-red-600"}>
          {String(value)}
        </span>
      );
    }
    return <span>{String(value)}</span>;
  }

  // For arrays and objects, provide expandable view
  const isArray = Array.isArray(value);
  const count = isArray ? value.length : Object.keys(value).length;

  if (count === 0) {
    return (
      <span className="text-muted-foreground italic">
        {isArray ? "[]" : "{}"}
      </span>
    );
  }

  const summary = isArray
    ? `Array (${count} items)`
    : `Object (${count} properties)`;

  return (
    <div>
      <Button
        variant="ghost"
        size="sm"
        onClick={() => setIsExpanded(!isExpanded)}
        className="px-2 py-1 h-7 text-foreground hover:bg-muted"
      >
        <Icon icon={isExpanded ? faMinus : faPlus} className="size-3.5 mr-2" />
        <span className="text-xs font-medium">{summary}</span>
      </Button>

      {isExpanded && (
        <div className="mt-2 border-l border-border pl-4 py-2">
          <JsonViewer data={value} initiallyExpanded={false} />
        </div>
      )}
    </div>
  );
}

/**
 * Enhanced changes table with better handling of complex objects
 */
export function ChangesTable({
  changes,
}: {
  changes: Record<string, { from: any; to: any }>;
}) {
  const [diffDialogOpen, setDiffDialogOpen] = useState(false);

  // Check if the changes data is too large
  const { isDataTooLarge, totalSize } = useMemo(() => {
    let totalJsonSize = 0;

    try {
      // Estimate total size by stringifying all changes
      Object.entries(changes).forEach(([, change]) => {
        if (change.from) {
          totalJsonSize += SuperJSON.stringify(change.from).length;
        }
        if (change.to) {
          totalJsonSize += SuperJSON.stringify(change.to).length;
        }
      });

      return {
        isDataTooLarge: totalJsonSize > DATA_SIZE_THRESHOLD,
        totalSize: totalJsonSize,
      };
    } catch {
      return { isDataTooLarge: true, totalSize: 0 };
    }
  }, [changes]);

  if (!changes) {
    return (
      <div className="flex flex-col gap-2 border border-dashed border-border rounded-md p-4">
        <p className="text-xs text-muted-foreground italic">
          No changes recorded
        </p>
      </div>
    );
  }

  // For large data sets, show a simplified version with just a button
  if (isDataTooLarge) {
    return (
      <div className="flex flex-col gap-2 border border-border rounded-md p-4">
        <div className="flex justify-between items-center">
          <div className="flex flex-col">
            <h3 className="text-sm font-normal">
              Changes Summary ({totalSize} bytes)
            </h3>
            <p className="text-2xs text-muted-foreground">
              {Object.keys(changes).length} fields were modified in this
              operation
            </p>
          </div>
          <ChangeActions changes={changes} />
        </div>

        <ChangeTooLargeAlert
          changes={changes}
          setDiffDialogOpen={setDiffDialogOpen}
        />

        <ChangeDiffDialog
          changes={changes}
          open={diffDialogOpen}
          onOpenChange={setDiffDialogOpen}
        />
      </div>
    );
  }

  // For smaller data sets, show the regular table
  return (
    <div className="flex flex-col gap-2 border border-border rounded-md p-3">
      <div className="flex justify-between items-center">
        <div className="flex flex-col">
          <h3 className="text-sm font-normal">Changes Summary</h3>
          <p className="text-2xs text-muted-foreground">
            Fields modified in this operation
          </p>
        </div>
        <ChangeActions changes={changes} />
      </div>
      <div className="overflow-x-auto">
        <Table>
          <TableHeader className="bg-transparent">
            <TableRow className="hover:bg-transparent">
              <TableHead className="font-medium bg-transparent">
                Field
              </TableHead>
              <TableHead className="font-medium bg-transparent">
                Previous Value
              </TableHead>
              <TableHead className="font-medium bg-transparent">
                Current Value
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {Object.entries(changes).map(([key, change]) => (
              <TableRow key={key} className="hover:bg-transparent">
                <TableCell className="font-medium whitespace-nowrap">
                  {key}
                </TableCell>
                <TableCell className="max-w-xs">
                  <DisplayValue value={change.from} />
                </TableCell>
                <TableCell className="max-w-xs">
                  <DisplayValue value={change.to} />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <ChangeDiffDialog
        changes={changes}
        open={diffDialogOpen}
        onOpenChange={setDiffDialogOpen}
      />
    </div>
  );
}

function ChangeActions({
  changes,
}: {
  changes: Record<string, { from: any; to: any }>;
}) {
  const [diffDialogOpen, setDiffDialogOpen] = useState(false);

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
        <DropdownMenuContent align="end">
          <DropdownMenuLabel>Change Options</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuItem
              title="Detailed Comparison"
              description="View side-by-side comparison of all changes"
              onClick={() => setDiffDialogOpen(true)}
            />
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
      <ChangeDiffDialog
        changes={changes}
        open={diffDialogOpen}
        onOpenChange={setDiffDialogOpen}
      />
    </>
  );
}

function ChangeTooLargeAlert({
  changes,
  setDiffDialogOpen,
}: {
  changes: Record<string, { from: any; to: any }>;
  setDiffDialogOpen: (open: boolean) => void;
}) {
  return (
    <div className="flex bg-amber-500/20 border border-amber-600/50 border-dashed p-4 rounded-md flex-col items-center">
      <p className="text-sm mb-4 text-center max-w-md text-amber-600">
        This operation includes extensive data changes that are best viewed in a
        dedicated comparison view.
      </p>
      <Button
        onClick={() => setDiffDialogOpen(true)}
        className="flex items-center gap-2 bg-amber-600 hover:bg-amber-600/80 text-amber-100"
      >
        <Icon icon={faArrowUpRightFromSquare} className="size-4" />
        <span>View Complete Changes</span>
      </Button>

      <div className="w-full mt-6 px-4">
        <div className="text-xs text-amber-600 mb-2">Modified fields:</div>
        <div className="flex flex-wrap gap-2">
          {Object.keys(changes).map((key) => (
            <div
              key={key}
              className="px-2 py-1 bg-amber-600/20 border border-amber-600/50 text-amber-600 rounded-md text-xs"
            >
              {key}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
