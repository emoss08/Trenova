import { SensitiveField } from "@/components/fields/sensitive-field";
import { Button } from "@/components/ui/button";
import { resetUserPassword } from "@/lib/user-api";
import { cn } from "@/lib/utils";
import { useQueryClient } from "@tanstack/react-query";
import { LockIcon } from "lucide-react";
import { useState } from "react";
import { useFormContext } from "react-hook-form";
import { toast } from "sonner";

export function EditModePassword({
  userId,
  isLocked,
}: {
  userId: string;
  isLocked?: boolean;
}) {
  const queryClient = useQueryClient();
  const [isResetting, setIsResetting] = useState(false);
  const [showNewPassword, setShowNewPassword] = useState(false);
  const { control } = useFormContext();

  const handleResetPassword = async () => {
    setIsResetting(true);
    await resetUserPassword(userId)
      .then(async () => {
        await queryClient.invalidateQueries({ queryKey: ["user", userId] });
        toast.success("Password reset email sent");
      })
      .catch(() => {
        toast.error("Failed to send password reset email");
      })
      .finally(() => {
        setIsResetting(false);
      });
  };

  return (
    <div className="space-y-4">
      {isLocked && (
        <div className="flex items-start gap-3 rounded-lg border border-destructive/30 bg-destructive/10 p-3">
          <LockIcon className="size-4 shrink-0 text-destructive" />
          <div>
            <p className="text-sm font-medium text-destructive">
              Account Locked
            </p>
            <p className="text-xs text-destructive/80">
              This account has been locked due to too many failed login
              attempts.
            </p>
          </div>
        </div>
      )}

      <div className="flex flex-col gap-3">
        <div className="flex flex-col gap-2">
          <Button
            type="button"
            onClick={handleResetPassword}
            disabled={isResetting}
          >
            {isResetting ? "Sending..." : "Send Reset Email"}
          </Button>
          <Button
            type="button"
            variant="outline"
            onClick={() => setShowNewPassword(!showNewPassword)}
          >
            {showNewPassword ? "Cancel" : "Set New Password"}
          </Button>
        </div>
      </div>

      {showNewPassword && (
        <div
          className={cn(
            "space-y-2 rounded-lg border bg-muted/30 p-3",
            "animate-in duration-200 fade-in-0 slide-in-from-top-2",
          )}
        >
          <SensitiveField
            control={control}
            name="newPassword"
            label="New Password"
            description="Enter new password"
            rules={{ required: true }}
          />
        </div>
      )}
    </div>
  );
}
