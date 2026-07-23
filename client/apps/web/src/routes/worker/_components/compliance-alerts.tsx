import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ScrollArea } from "@/components/ui/scroll-area";
import { API_BASE_URL } from "@/lib/constants";
import { cn } from "@/lib/utils";
import type { GenericLimitOffsetResponse } from "@/types/server";
import type { Worker } from "@/types/worker";
import { useQuery } from "@tanstack/react-query";
import { formatDistanceToNow } from "date-fns";
import {
  AlertTriangle,
  CheckCircle,
  CreditCard,
  EllipsisVertical,
  FileText,
  HeartPulse,
  MessageSquare,
  ShieldAlert,
} from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";

type AlertType =
  | "license_expiring"
  | "license_expired"
  | "medical_expiring"
  | "medical_expired"
  | "mvr_due"
  | "hazmat_expiring"
  | "hazmat_expired"
  | "twic_expiring"
  | "twic_expired"
  | "physical_due";

type AlertSeverity = "critical" | "warning";

interface ComplianceAlert {
  id: string;
  worker: Worker;
  type: AlertType;
  severity: AlertSeverity;
  message: string;
  dueDate: number;
  daysUntil: number;
}

const alertConfig: Record<
  AlertType,
  { icon: typeof AlertTriangle; label: string; description: string }
> = {
  license_expiring: {
    icon: CreditCard,
    label: "CDL License",
    description: "Commercial Driver's License expiring soon",
  },
  license_expired: {
    icon: CreditCard,
    label: "CDL License",
    description: "Commercial Driver's License has expired",
  },
  medical_expiring: {
    icon: HeartPulse,
    label: "Medical Card",
    description: "DOT Medical Certificate expiring soon",
  },
  medical_expired: {
    icon: HeartPulse,
    label: "Medical Card",
    description: "DOT Medical Certificate has expired",
  },
  mvr_due: {
    icon: FileText,
    label: "MVR Check",
    description: "Motor Vehicle Record check due",
  },
  hazmat_expiring: {
    icon: ShieldAlert,
    label: "Hazmat Endorsement",
    description: "Hazmat endorsement expiring soon",
  },
  hazmat_expired: {
    icon: ShieldAlert,
    label: "Hazmat Endorsement",
    description: "Hazmat endorsement has expired",
  },
  twic_expiring: {
    icon: CreditCard,
    label: "TWIC Card",
    description: "Transportation Worker ID Card expiring soon",
  },
  twic_expired: {
    icon: CreditCard,
    label: "TWIC Card",
    description: "Transportation Worker ID Card has expired",
  },
  physical_due: {
    icon: HeartPulse,
    label: "Physical Exam",
    description: "Physical examination due",
  },
};

function getInitials(firstName?: string, lastName?: string): string {
  return `${firstName?.[0] ?? ""}${lastName?.[0] ?? ""}`.toUpperCase() || "??";
}

function getSeverity(daysUntil: number): AlertSeverity {
  return daysUntil <= 7 ? "critical" : "warning";
}

function checkDateAlert(
  worker: Worker,
  dateValue: number | null | undefined,
  expiringType: AlertType,
  expiredType: AlertType,
  warningDays: number = 30,
): ComplianceAlert | null {
  if (!dateValue) return null;

  const now = Date.now();
  const daysUntil = Math.ceil((dateValue - now) / (1000 * 60 * 60 * 24));

  if (daysUntil > warningDays) return null;

  const isExpired = daysUntil <= 0;
  const type = isExpired ? expiredType : expiringType;
  const severity = getSeverity(daysUntil);

  const message = isExpired
    ? `Expired ${formatDistanceToNow(dateValue, { addSuffix: true })}`
    : `Expires ${formatDistanceToNow(dateValue, { addSuffix: true })}`;

  return {
    id: `${worker.id}-${type}`,
    worker,
    type,
    severity,
    message,
    dueDate: dateValue,
    daysUntil,
  };
}

function AlertCard({
  alert,
  onSendSMS,
  smsEnabled,
}: {
  alert: ComplianceAlert;
  onSendSMS?: (alert: ComplianceAlert) => void;
  smsEnabled?: boolean;
}) {
  const config = alertConfig[alert.type];
  const Icon = config.icon;
  const hasPhone = Boolean(alert.worker.phoneNumber);

  return (
    <div
      className={cn(
        "group relative rounded-lg border p-3 transition-all hover:shadow-sm",
        alert.severity === "critical"
          ? "border-red-200 bg-red-50/50 dark:border-red-900/50 dark:bg-red-950/20"
          : "border-amber-200 bg-amber-50/50 dark:border-amber-900/50 dark:bg-amber-950/20",
      )}
    >
      <div
        className={cn(
          "absolute top-0 bottom-0 left-0 w-1 rounded-l-lg",
          alert.severity === "critical" ? "bg-red-500" : "bg-amber-500",
        )}
      />
      <div className="flex items-start gap-3 pl-2">
        <Avatar className="size-10 shrink-0 border">
          <AvatarImage
            src={alert.worker.profilePicUrl ?? undefined}
            alt={`${alert.worker.firstName} ${alert.worker.lastName}`}
          />
          <AvatarFallback className="bg-muted text-xs font-medium">
            {getInitials(alert.worker.firstName, alert.worker.lastName)}
          </AvatarFallback>
        </Avatar>
        <div className="min-w-0 flex-1">
          <div className="flex items-center justify-between gap-2">
            <div className="flex min-w-0 items-center gap-2">
              <span className="truncate font-medium">
                {alert.worker.firstName} {alert.worker.lastName}
              </span>
              <Badge
                variant={alert.severity === "critical" ? "inactive" : "warning"}
                className="h-5 shrink-0 text-[10px]"
              >
                {alert.severity === "critical" ? "Critical" : "Warning"}
              </Badge>
            </div>
            <DropdownMenu>
              <DropdownMenuTrigger
                render={
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    className="size-7 opacity-0 transition-opacity group-hover:opacity-100"
                  >
                    <EllipsisVertical className="size-4" />
                  </Button>
                }
              />
              <DropdownMenuContent align="end">
                <DropdownMenuGroup>
                  <DropdownMenuLabel>Actions</DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem title="View Worker" onClick={() => {}} />
                  <DropdownMenuItem
                    title="Mark as Resolved"
                    onClick={() => {}}
                  />
                  {smsEnabled && hasPhone && (
                    <DropdownMenuItem
                      title="Send SMS Reminder"
                      onClick={() => onSendSMS?.(alert)}
                      startContent={<MessageSquare className="mr-2 size-4" />}
                    />
                  )}
                </DropdownMenuGroup>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
          <div className="mt-1.5 flex items-center gap-2">
            <div
              className={cn(
                "flex size-6 items-center justify-center rounded",
                alert.severity === "critical"
                  ? "bg-red-100 dark:bg-red-900/30"
                  : "bg-amber-100 dark:bg-amber-900/30",
              )}
            >
              <Icon
                className={cn(
                  "size-3.5",
                  alert.severity === "critical"
                    ? "text-red-600 dark:text-red-400"
                    : "text-amber-600 dark:text-amber-400",
                )}
              />
            </div>
            <div className="flex flex-col">
              <span className="text-sm font-medium">{config.label}</span>
              <span className="text-xs text-muted-foreground">
                {alert.message}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function EmptyState() {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 py-12">
      <div className="flex size-14 items-center justify-center rounded-full bg-emerald-100 dark:bg-emerald-900/30">
        <CheckCircle className="size-7 text-emerald-600 dark:text-emerald-400" />
      </div>
      <div className="text-center">
        <p className="font-medium">All Clear</p>
        <p className="text-sm text-muted-foreground">
          No compliance issues at this time
        </p>
      </div>
    </div>
  );
}

function LoadingSkeleton() {
  return (
    <div className="flex flex-col gap-3 p-3">
      {Array.from({ length: 3 }).map((_, i) => (
        <div
          key={i}
          className="flex animate-pulse items-start gap-3 rounded-lg border p-3"
        >
          <div className="size-10 rounded-full bg-muted" />
          <div className="flex-1 space-y-2">
            <div className="h-4 w-32 rounded bg-muted" />
            <div className="h-6 w-full rounded bg-muted" />
            <div className="h-3 w-24 rounded bg-muted" />
          </div>
        </div>
      ))}
    </div>
  );
}

async function fetchWorkersForCompliance(): Promise<
  GenericLimitOffsetResponse<Worker>
> {
  const url = new URL(`${API_BASE_URL}/workers/`, window.location.origin);
  url.searchParams.set("limit", "100");
  url.searchParams.set("offset", "0");

  const response = await fetch(url.href, { credentials: "include" });
  if (!response.ok) {
    throw new Error("Failed to fetch workers");
  }
  return response.json();
}

export function ComplianceAlerts() {
  const [smsEnabled] = useState(false); // Will be set from API check

  const { data: workers, isLoading } = useQuery({
    queryKey: ["worker-compliance-alerts"],
    queryFn: fetchWorkersForCompliance,
    staleTime: 30000,
  });

  const alerts = useMemo(() => {
    if (!workers?.results) return [];

    const allAlerts: ComplianceAlert[] = [];

    for (const worker of workers.results) {
      if (worker.status !== "Active") continue;

      const profile = worker.profile;
      if (!profile) continue;

      const licenseAlert = checkDateAlert(
        worker,
        profile.licenseExpiry,
        "license_expiring",
        "license_expired",
        30,
      );
      if (licenseAlert) allAlerts.push(licenseAlert);

      const medicalAlert = checkDateAlert(
        worker,
        profile.medicalCardExpiry,
        "medical_expiring",
        "medical_expired",
        30,
      );
      if (medicalAlert) allAlerts.push(medicalAlert);

      const mvrAlert = checkDateAlert(
        worker,
        profile.mvrDueDate,
        "mvr_due",
        "mvr_due",
        14,
      );
      if (mvrAlert) allAlerts.push(mvrAlert);

      if (profile.endorsement === "H" || profile.endorsement === "X") {
        const hazmatAlert = checkDateAlert(
          worker,
          profile.hazmatExpiry,
          "hazmat_expiring",
          "hazmat_expired",
          30,
        );
        if (hazmatAlert) allAlerts.push(hazmatAlert);
      }

      const twicAlert = checkDateAlert(
        worker,
        profile.twicExpiry,
        "twic_expiring",
        "twic_expired",
        30,
      );
      if (twicAlert) allAlerts.push(twicAlert);

      const physicalAlert = checkDateAlert(
        worker,
        profile.physicalDueDate,
        "physical_due",
        "physical_due",
        14,
      );
      if (physicalAlert) allAlerts.push(physicalAlert);
    }

    return allAlerts
      .sort((a, b) => {
        if (a.severity !== b.severity) {
          return a.severity === "critical" ? -1 : 1;
        }
        return a.daysUntil - b.daysUntil;
      })
      .slice(0, 20);
  }, [workers]);

  const criticalCount = alerts.filter((a) => a.severity === "critical").length;
  const warningCount = alerts.filter((a) => a.severity === "warning").length;

  const handleSendSMS = async (alert: ComplianceAlert) => {
    const config = alertConfig[alert.type];
    const message = `Hi ${alert.worker.firstName}, this is a reminder that your ${config.label} ${alert.message}. Please update your records as soon as possible.`;

    // TODO: Call SMS API endpoint
    toast.info("SMS feature coming soon", {
      description: `Would send: "${message}" to ${alert.worker.phoneNumber}`,
    });
  };

  return (
    <div className="flex h-full flex-col overflow-hidden rounded-md border border-border">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-border bg-muted/30 px-4 py-3">
        <div className="flex items-center gap-2">
          <AlertTriangle className="size-4 text-amber-500" />
          <h3 className="font-semibold">Compliance Alerts</h3>
        </div>
        <div className="flex items-center gap-2">
          {criticalCount > 0 && (
            <Badge variant="inactive" className="text-xs">
              {criticalCount} critical
            </Badge>
          )}
          {warningCount > 0 && (
            <Badge variant="outline" className="text-xs">
              {warningCount} warning
            </Badge>
          )}
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-hidden">
        {isLoading ? (
          <LoadingSkeleton />
        ) : alerts.length === 0 ? (
          <EmptyState />
        ) : (
          <ScrollArea className="h-full">
            <div className="flex flex-col gap-3 p-3">
              {alerts.map((alert) => (
                <AlertCard
                  key={alert.id}
                  alert={alert}
                  onSendSMS={handleSendSMS}
                  smsEnabled={smsEnabled}
                />
              ))}
            </div>
          </ScrollArea>
        )}
      </div>
    </div>
  );
}
