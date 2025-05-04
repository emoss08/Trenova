import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { cn } from "@/lib/utils";
import type { DocumentCategory } from "@/types/document";
import { faArrowRight } from "@fortawesome/pro-solid-svg-icons";
import { useMemo } from "react";

export function BillingReadinessBadge({
  documentCategories,
}: {
  documentCategories: DocumentCategory[];
}) {
  const billingReadiness = useMemo(() => {
    const requiredCategories = documentCategories;
    const completedRequired = documentCategories.filter((cat) => cat.complete);

    return {
      total: requiredCategories.length,
      completed: completedRequired.length,
      ready:
        requiredCategories.length > 0 &&
        requiredCategories.length === completedRequired.length,
    };
  }, [documentCategories]);

  return (
    billingReadiness.total > 0 && (
      <div className="mt-3 p-2 rounded-md bg-background border border-border">
        <div className="flex items-center justify-between mb-1">
          <span className="text-sm font-medium">Billing Ready</span>
          {billingReadiness.ready ? (
            <Button
              title="Transfer to Billing"
              aria-label="Transfer to Billing"
              variant="outline"
              size="xs"
            >
              Transfer to Billing
              <Icon icon={faArrowRight} className="size-4" />
            </Button>
          ) : (
            <Badge withDot={false} variant="outline">
              {billingReadiness.completed}/{billingReadiness.total}
            </Badge>
          )}
        </div>
        <div className="w-full h-1 bg-muted rounded-full overflow-hidden">
          <div
            className={cn(
              "h-full rounded-full",
              billingReadiness.ready ? "bg-green-600" : "bg-primary",
            )}
            style={{
              width: `${
                billingReadiness.total > 0
                  ? (billingReadiness.completed / billingReadiness.total) * 100
                  : 0
              }%`,
            }}
          />
        </div>
      </div>
    )
  );
}
