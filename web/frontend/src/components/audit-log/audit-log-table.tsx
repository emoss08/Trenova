import JsonViewer from "@/components/json-viewer";
import { formatToUserTimezone } from "@/lib/date";
import { AuditLog, EnumAuditLogStatus } from "@/types/organization";
import { MoonIcon, SunIcon } from "@radix-ui/react-icons";
import { useState } from "react";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";
import { ScrollArea } from "../ui/scroll-area";

export function AuditLogView({ auditLog }: { auditLog: AuditLog }) {
  const [showDetails, setShowDetails] = useState(false);
  const [isDarkTheme, setIsDarkTheme] = useState(true);

  const getChangeDescription = (changes: any) => {
    const changedFields = Object.keys(changes).filter(
      (key) =>
        JSON.stringify(changes[key].from) !== JSON.stringify(changes[key].to),
    );

    if (changedFields.length === 0) return "No changes detected";
    if (changedFields.length === 1) return `Changed ${changedFields[0]}`;
    if (changedFields.length === 2)
      return `Changed ${changedFields[0]} and ${changedFields[1]}`;
    return `Changed ${changedFields.length} fields`;
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold">{auditLog.action}</h3>
          <p className="text-sm text-gray-500">
            {formatToUserTimezone(auditLog.timestamp)} by {auditLog.username}
          </p>
        </div>
        <Badge
          variant={
            auditLog.status === EnumAuditLogStatus.SUCCEEDED
              ? "active"
              : "inactive"
          }
        >
          {auditLog.status}
        </Badge>
      </div>

      <p>{getChangeDescription(auditLog.changes)}</p>

      {auditLog.description && (
        <p className="text-sm italic">{auditLog.description}</p>
      )}

      {auditLog.errorMessage && (
        <p className="text-sm text-red-500">{auditLog.errorMessage}</p>
      )}

      <Button variant="outline" onClick={() => setShowDetails(!showDetails)}>
        {showDetails ? "Hide Details" : "Show Details"}
      </Button>

      {showDetails && (
        <div>
          <div className="mb-2 flex justify-end">
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setIsDarkTheme(!isDarkTheme)}
            >
              {isDarkTheme ? (
                <SunIcon className="size-4" />
              ) : (
                <MoonIcon className="size-4" />
              )}
            </Button>
          </div>
          <ScrollArea className="h-[300px] rounded border">
            <JsonViewer json={auditLog.changes} dark={isDarkTheme} />
          </ScrollArea>
        </div>
      )}
    </div>
  );
}
