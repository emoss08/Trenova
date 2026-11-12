import { ViewersStack } from "@/components/presence/viewers-stack";
import { usePresenceViewers } from "@/hooks/use-presence-viewers";
import { recordPresence } from "@/lib/presence-scopes";
import { useDeskMarginStore } from "@/stores/desk-margin-store";
import { useT } from "@trenova/shared/i18n/use-t";
import { Resource } from "@trenova/shared/types/permission";

/**
 * Who else has the case in front open, for the Desk's top bar. It sits there
 * rather than in the case, so people arriving and leaving never move what the
 * person is reading, and it is one small group however many are on the case.
 * Having the case open is what shows this person to the others.
 */
export function CaseViewers() {
  const t = useT();
  const caseId = useDeskMarginStore((state) => state.casePage?.id ?? null);
  const { viewers } = usePresenceViewers(
    caseId === null ? null : recordPresence(Resource.Watchtower, caseId),
  );

  return (
    <ViewersStack
      viewers={viewers}
      size="sm"
      max={3}
      label={(names) => t("Also on this case: {0}", names)}
    />
  );
}
