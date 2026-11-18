import { ArrowUpRightIcon } from "@trenova/shared/components/icons";
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
} from "@trenova/shared/components/ui/alert";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { buttonVariants } from "@trenova/shared/lib/variants/button";

/**
 * The plan is set to end: when, what happens then, and, for whoever can open the Trenova
 * account, the way to keep it. Nothing is drawn while the plan is not ending.
 */
export function PlanEndNotice({
  endsAt,
  finalBackupDays,
  manageUrl,
}: {
  endsAt: number | null | undefined;
  finalBackupDays: number;
  manageUrl: string | null;
}) {
  const t = useT();

  if (endsAt == null) {
    return null;
  }

  return (
    <Alert variant="warning">
      <AlertTitle>{t("Your plan ends on {0}.", formatUnixDateMedium(endsAt))}</AlertTitle>
      <AlertDescription>
        {t(
          "Trenova stays available until then, with everything your plan includes. After that, this workspace is taken down and a final backup is kept for {0, plural, one {# day} other {# days}}.",
          finalBackupDays,
        )}
      </AlertDescription>
      {manageUrl ? (
        <AlertAction>
          <a
            href={manageUrl}
            target="_blank"
            rel="noopener noreferrer"
            className={buttonVariants({ variant: "default", size: "sm" })}
          >
            {t("Keep plan")}
            <ArrowUpRightIcon className="size-3.5" aria-hidden />
          </a>
        </AlertAction>
      ) : null}
    </Alert>
  );
}
