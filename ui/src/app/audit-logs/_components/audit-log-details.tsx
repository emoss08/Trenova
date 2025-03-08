import { InternalLink } from "@/components/ui/link";
import { generateDateTimeStringFromUnixTimestamp } from "@/lib/date";
import { getRoutePageInfo } from "@/lib/utils";
import { AuditEntry } from "@/types/audit-entry";
import { useMemo } from "react";
import {
  AuditEntryActionBadge,
  AuditEntryResourceBadge,
} from "./audit-column-components";

export function AuditLogDetails({ entry }: { entry: AuditEntry }) {
  // Get resource page info including whether it supports modals
  const { path: resourcePath, supportsModal } = useMemo(
    () => getRoutePageInfo(entry.resource),
    [entry.resource],
  );

  // Generate the appropriate link based on modal support
  const resourceLink = useMemo(() => {
    // Base link to the resource page
    const baseLink = resourcePath;

    // If the resource doesn't support modals, just link to the page
    if (!supportsModal) {
      return {
        to: baseLink,
        state: {},
      };
    }

    // Otherwise, build a link with modal parameters
    return {
      to: `${baseLink}?entityId=${entry.resourceId}&modal=edit`,
      state: {
        isNavigatingToModal: true,
      },
    };
  }, [resourcePath, supportsModal, entry.resourceId]);

  const items = [
    {
      title: "Resource ID",
      value: (
        <InternalLink
          to={resourceLink.to}
          state={resourceLink.state}
          className="underline cursor-pointer"
          replace
          preventScrollReset
        >
          {entry.resourceId}
        </InternalLink>
      ),
    },
    {
      title: "Action",
      value: <AuditEntryActionBadge action={entry.action} withDot={false} />,
    },
    {
      title: "Resource",
      value: (
        <AuditEntryResourceBadge resource={entry.resource} withDot={false} />
      ),
    },
    {
      title: "User",
      value: (
        <span className="underline cursor-pointer">
          {entry.user?.name || entry.user?.emailAddress}
        </span>
      ),
    },
    {
      title: "Critical",
      value: <span>{entry.critical ? "Yes" : "No"}</span>,
    },
    {
      title: "IP Address",
      value: <span>{entry.ipAddress || "-"}</span>,
    },
    {
      title: "Category",
      value: <span>{entry.category || "-"}</span>,
    },
    {
      title: "Timestamp",
      value: generateDateTimeStringFromUnixTimestamp(entry.timestamp),
    },
  ];

  return (
    <div className="flex flex-col">
      <h3 className="text-sm font-normal">Entry Details</h3>
      <p className="text-2xs text-muted-foreground">
        Detailed information about the audit log entry
      </p>
      <div className="mt-2">
        {items.map((item) => (
          <AuditLogDetailsItem key={item.title} title={item.title}>
            {item.value}
          </AuditLogDetailsItem>
        ))}
      </div>
    </div>
  );
}

function AuditLogDetailsItem({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className="p-1 sm:grid sm:grid-cols-3 sm:gap-2">
      <dt className="text-sm font-medium text-muted-foreground">{title}</dt>
      <dd className="mt-1 text-sm text-foreground sm:col-span-2 sm:mt-0">
        {children}
      </dd>
    </div>
  );
}
