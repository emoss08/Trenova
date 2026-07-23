import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Loader2Icon, ShieldIcon, UserIcon } from "lucide-react";

type UserSummaryBarProps = {
  roleCount: number;
  status: string;
  isSubmitting: boolean;
  submitLabel: string;
  onSubmit: () => void;
  onCancel: () => void;
};

export function UserSummaryBar({
  roleCount,
  status,
  isSubmitting,
  submitLabel,
  onSubmit,
  onCancel,
}: UserSummaryBarProps) {
  return (
    <div className="flex items-center justify-between px-6 py-3">
      <div className="flex items-center gap-5 text-sm">
        <div className="flex items-center gap-2">
          <UserIcon className="size-4 text-muted-foreground" />
          <span className="text-muted-foreground">Status:</span>
          <Badge
            variant={status === "Active" ? "default" : "secondary"}
            className={
              status === "Active" ? "bg-green-600 hover:bg-green-700" : ""
            }
          >
            {status || "Active"}
          </Badge>
        </div>

        <div className="h-4 w-px bg-border" />

        <div className="flex items-center gap-2">
          <ShieldIcon className="size-4 text-muted-foreground" />
          <span className="font-medium tabular-nums">{roleCount}</span>
          <span className="text-muted-foreground">
            role{roleCount !== 1 ? "s" : ""} assigned
          </span>
        </div>
      </div>

      <div className="flex items-center gap-2">
        <Button type="button" variant="outline" size="sm" onClick={onCancel}>
          Cancel
        </Button>
        <Button
          type="button"
          size="sm"
          onClick={onSubmit}
          disabled={isSubmitting}
        >
          {isSubmitting && <Loader2Icon className="mr-2 size-4 animate-spin" />}
          {submitLabel}
        </Button>
      </div>
    </div>
  );
}
