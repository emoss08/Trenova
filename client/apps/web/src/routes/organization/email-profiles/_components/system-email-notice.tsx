import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { useT } from "@trenova/shared/i18n/use-t";

export function SystemEmailNotice({ fromAddress }: { fromAddress: string }) {
  const t = useT();

  return (
    <Alert size="sm" variant="info">
      <AlertDescription>
        <p className="font-medium">{t("System email is sent by Trenova on your behalf")}</p>
        <p>
          {t(
            "Invitations, password resets and notifications go out from {0} until a profile is assigned to General. Email to anyone outside your organization needs your own profile.",
            fromAddress,
          )}
        </p>
      </AlertDescription>
    </Alert>
  );
}
