import { handleMutationError } from "@/hooks/use-api-mutation";
import { assignDeskCase, type DeskCaseOwnership } from "@/lib/graphql/desk-case";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useEffect, useRef } from "react";

/**
 * A case opened from the Watchtower to be worked becomes the opener's: nobody
 * owns it, they chose to work it, so it is theirs without a second press.
 *
 * Only an unowned case the person may claim is taken. One that is someone
 * else's stays theirs, even for a lead who could take it over: opening a
 * case to read it is not taking it from the person working it. It is asked
 * once for each case opened this way.
 */
export function useClaimOnOpen({
  kind,
  id,
  ownership,
  enabled,
  onClaimed,
}: {
  kind: string;
  id: string;
  /** Undefined while the case is read. */
  ownership: DeskCaseOwnership | undefined;
  /** The case was opened to be worked, not only looked at. */
  enabled: boolean;
  onClaimed: () => unknown;
}) {
  const asked = useRef<string | null>(null);
  const claimed = useLatestCallback(onClaimed);
  const claimable =
    enabled && ownership !== undefined && ownership.owner == null && ownership.canClaim;

  useEffect(() => {
    const key = `${kind}:${id}`;
    if (!claimable || asked.current === key) {
      return;
    }
    asked.current = key;
    assignDeskCase(kind, id, null).then(
      () => claimed(),
      (error: unknown) => handleMutationError({ error, resourceName: "Case" }),
    );
  }, [claimable, kind, id, claimed]);
}
