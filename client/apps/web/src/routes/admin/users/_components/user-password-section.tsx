import { useT } from "@trenova/shared/i18n/use-t";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { resetUserMFA, resetUserPassword } from "@/lib/user-api";
import { cn } from "@trenova/shared/lib/utils";
import { useQueryClient } from "@tanstack/react-query";
import { Lock01Icon } from "@trenova/shared/components/icons";
import { useState } from "react";
import { useFormContext } from "react-hook-form";
import { toast } from "sonner";

export function EditModePassword({ userId, isLocked }: { userId: string; isLocked?: boolean }) {
  const t = useT();

  const queryClient = useQueryClient();
  const [isResetting, setIsResetting] = useState(false);
  const [showNewPassword, setShowNewPassword] = useState(false);
  const [confirmingMFAReset, setConfirmingMFAReset] = useState(false);
  const [isResettingMFA, setIsResettingMFA] = useState(false);
  const { control } = useFormContext();

  const handleResetMFA = async () => {
    setIsResettingMFA(true);
    try {
      await resetUserMFA(userId);
      toast.success(t("Two-factor authentication reset"), {
        description: t("This user signs in with their password alone until they set it up again."),
      });
      setConfirmingMFAReset(false);
    } catch (error) {
      handleMutationError({ error, resourceName: "Two-factor reset" });
    } finally {
      setIsResettingMFA(false);
    }
  };

  const handleResetPassword = async () => {
    setIsResetting(true);
    try {
      await resetUserPassword(userId);
      await queryClient.invalidateQueries({ queryKey: ["user", userId] });
      toast.success(t("Password reset link sent"));
    } catch (error) {
      // Surfaces what the server actually said. The previous fixed string hid the fact
      // that this button was calling a route that did not exist.
      handleMutationError({ error, resourceName: "Password reset" });
    } finally {
      setIsResetting(false);
    }
  };

  return (
    <div className="space-y-4">
      {isLocked && (
        <Alert variant="destructive" size="sm">
          <Lock01Icon />
          <AlertTitle>{t("Account locked")}</AlertTitle>
          <AlertDescription>
            {t("This account has been locked due to too many failed login attempts.")}
          </AlertDescription>
        </Alert>
      )}

      <div className="flex flex-col gap-3">
        <div className="flex flex-col gap-2">
          <Button type="button" onClick={handleResetPassword} disabled={isResetting}>
            {isResetting ? t("Sending...") : t("Send reset email")}
          </Button>
          <p className="text-muted-foreground text-2xs">
            {t(
              "Emails this user a single-use link to choose their own password. Their current password keeps working until they use it, and you never see the new one.",
            )}
          </p>
          <Button
            type="button"
            variant="outline"
            onClick={() => setShowNewPassword(!showNewPassword)}
          >
            {showNewPassword ? t("Cancel") : t("Set new password")}
          </Button>
          {confirmingMFAReset ? (
            <div className="flex gap-2">
              <Button
                type="button"
                variant="destructive"
                className="flex-1"
                onClick={handleResetMFA}
                disabled={isResettingMFA}
              >
                {isResettingMFA ? t("Resetting...") : t("Confirm two-factor reset")}
              </Button>
              <Button type="button" variant="outline" onClick={() => setConfirmingMFAReset(false)}>
                {t("Cancel")}
              </Button>
            </div>
          ) : (
            <Button type="button" variant="outline" onClick={() => setConfirmingMFAReset(true)}>
              {t("Reset two-factor authentication")}
            </Button>
          )}
          <p className="text-muted-foreground text-2xs">
            {t(
              "Removes this user's authenticator app and recovery codes, for someone who lost their device. They can set it up again from their settings.",
            )}
          </p>
        </div>
      </div>

      {showNewPassword && (
        <div
          className={cn(
            "bg-muted/30 space-y-2 rounded-lg border p-3",
            "animate-in fade-in-0 slide-in-from-top-2 duration-200",
          )}
        >
          <SensitiveField
            control={control}
            name="newPassword"
            label={t("New password")}
            description={t("Enter new password")}
            rules={{ required: true }}
          />
        </div>
      )}
    </div>
  );
}
