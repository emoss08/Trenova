import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormSection } from "@trenova/shared/components/ui/form";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { resetUserMFA, resetUserPassword } from "@/lib/user-api";
import { useState } from "react";
import { toast } from "@trenova/shared/components/ui/toast";

export function UserAccessSection({
  userId,
  emailAddress,
  isActive,
  awaitingSetup,
}: {
  userId: string;
  emailAddress: string;
  isActive: boolean;
  awaitingSetup: boolean;
}) {
  const t = useT();

  const [isSending, setIsSending] = useState(false);
  const [confirmingMFAReset, setConfirmingMFAReset] = useState(false);
  const [isResettingMFA, setIsResettingMFA] = useState(false);

  const handleSendLink = async () => {
    setIsSending(true);
    try {
      await resetUserPassword(userId);
      toast.success(awaitingSetup ? t("Invitation sent") : t("Password reset link sent"), {
        description: t("The email went to {0}.", emailAddress),
      });
    } catch (error) {
      handleMutationError({
        error,
        resourceName: awaitingSetup ? "Invitation" : "Password reset",
      });
    } finally {
      setIsSending(false);
    }
  };

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

  return (
    <FormControl>
      <FormSection
        className="border-border gap-3 rounded-lg border p-4"
        title={t("Sign-in access")}
        description={
          awaitingSetup
            ? t("This user has not set up their account yet.")
            : t("Nobody, including you, ever sees this user's password.")
        }
      >
        <div className="flex flex-col gap-2">
          <Button
            type="button"
            variant="outline"
            onClick={handleSendLink}
            disabled={isSending || !isActive}
            isLoading={isSending}
            loadingText={t("Sending...")}
          >
            {awaitingSetup ? t("Resend invitation") : t("Send password reset link")}
          </Button>
          <p className="text-muted-foreground text-xs">
            {!isActive
              ? t("Activate the account before sending it a sign-in link.")
              : awaitingSetup
                ? t(
                    "Emails {0} a new link to choose a password. Any earlier invitation stops working.",
                    emailAddress,
                  )
                : t(
                    "Emails {0} a single-use link to choose a new password. Their current password keeps working until they use it.",
                    emailAddress,
                  )}
          </p>
        </div>
        <div className="flex flex-col gap-2">
          {confirmingMFAReset ? (
            <div className="flex gap-2">
              <Button
                type="button"
                variant="destructive"
                className="flex-1"
                onClick={handleResetMFA}
                disabled={isResettingMFA}
                isLoading={isResettingMFA}
                loadingText={t("Resetting...")}
              >
                {t("Confirm two-factor reset")}
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
          <p className="text-muted-foreground text-xs">
            {t(
              "Removes this user's authenticator app and recovery codes, for someone who lost their device. They can set it up again from their settings.",
            )}
          </p>
        </div>
      </FormSection>
    </FormControl>
  );
}
