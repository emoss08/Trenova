import { usePermission } from "@/hooks/use-permission";
import { useQuery } from "@tanstack/react-query";
import { Phone01Icon } from "@trenova/shared/components/icons";
import { Alert, AlertAction, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useNow } from "@trenova/shared/hooks/use-now";
import { formatList, formatRelativeTime } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { isPhotoChange } from "@trenova/shared/lib/self-service";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  pendingProfileChanges,
  workerProfileChangesQuery,
} from "../profile-change/profile-change-query";

/**
 * Above the Profile form while the driver has asked from Dash to change something on it: who asked,
 * when and for which fields, and Review, which opens the request beside the form.
 */
export function ProfileChangeAlert({
  workerId,
  firstName,
  waiting,
  onReview,
}: {
  workerId: string;
  firstName: string;
  /** Requests waiting, from the overview read; nothing is read when it is zero or withheld. */
  waiting: number | null;
  onReview: (requestId: string) => void;
}) {
  const t = useT();
  const now = useNow("minute");
  const { allowed: canRead } = usePermission(Resource.ProfileChangeRequest, Operation.Read);
  const requests = useQuery({
    ...workerProfileChangesQuery(workerId),
    enabled: canRead && (waiting ?? 0) > 0,
  });
  const pending = pendingProfileChanges(requests.data);
  const first = pending[0];
  if (!canRead || !first) return null;

  const asked = formatRelativeTime(Math.min(first.submittedAt - now, 0));
  const message =
    pending.length === 1 && isPhotoChange(first.changes)
      ? t("{0} sent a new profile photo {1}.", firstName, asked)
      : pending.length === 1
        ? t(
            "{0} asked {1} for a profile change: {2}.",
            firstName,
            asked,
            formatList(first.changes.map((change) => t(change.label))),
          )
        : t(
            "{0} has {1, plural, one {# profile change} other {# profile changes}} waiting for a decision.",
            firstName,
            pending.length,
          );

  return (
    <Alert size="sm" variant="warning">
      <Phone01Icon aria-hidden />
      <AlertDescription>{message}</AlertDescription>
      <AlertAction>
        <Button type="button" size="sm" variant="outline" onClick={() => onReview(first.id)}>
          {t("Review")}
        </Button>
      </AlertAction>
    </Alert>
  );
}
