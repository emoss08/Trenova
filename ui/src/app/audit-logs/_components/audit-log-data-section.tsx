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
import { Icon } from "@/components/ui/icons";
import { ShikiJsonViewer } from "@/components/ui/json-viewer";
import { Separator } from "@/components/ui/separator";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { convertValueToDisplay } from "@/lib/utils";
import {
  faChevronDown,
  faChevronRight,
} from "@fortawesome/pro-solid-svg-icons";
import { useState } from "react";

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
            <div className="border rounded-md">
              <CollapsibleTrigger asChild>
                <div className="flex items-center p-3 cursor-pointer hover:bg-accent">
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
                  <div className="p-2 bg-red-50 dark:bg-red-950/30 rounded-md">
                    <div className="text-xs font-medium text-muted-foreground mb-1">
                      Previous
                    </div>
                    {hasFrom ? (
                      <ShikiJsonViewer data={change.from} />
                    ) : (
                      <p className="text-xs text-muted-foreground italic">
                        null
                      </p>
                    )}
                  </div>
                  <div className="p-2 bg-green-50 dark:bg-green-950/30 rounded-md">
                    <div className="text-xs font-medium text-muted-foreground mb-1">
                      Current
                    </div>
                    {hasTo ? (
                      <ShikiJsonViewer data={change.to} />
                    ) : (
                      <p className="text-xs text-muted-foreground italic">
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

export function ChangesTable({
  changes,
}: {
  changes: Record<string, { from: any; to: any }>;
}) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Field</TableHead>
          <TableHead>Previous</TableHead>
          <TableHead>Current</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {Object.entries(changes).map(([key, change]) => {
          const hasFrom = change.from !== undefined && change.from !== null;
          const hasTo = change.to !== undefined && change.to !== null;

          return (
            <TableRow key={key}>
              <TableCell>{key}</TableCell>
              <TableCell>
                {hasFrom ? convertValueToDisplay(change.from) : "null"}
              </TableCell>
              <TableCell>
                {hasTo ? convertValueToDisplay(change.to) : "null"}
              </TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}
