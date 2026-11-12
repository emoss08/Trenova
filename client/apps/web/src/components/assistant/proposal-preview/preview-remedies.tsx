import { recordPath } from "@/config/record-links";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import type { PreviewWarning } from "@/lib/graphql/agent-preview";
import { agentPreview } from "@/lib/queries/agent-preview";
import { useQueryClient } from "@tanstack/react-query";
import { CheckIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { inviteWorkerToPortal } from "@trenova/shared/lib/graphql/driver-portal";
import { buttonVariants } from "@trenova/shared/lib/variants/button";
import { Operation, Resource, type OperationType } from "@trenova/shared/types/permission";
import { useRef, useState } from "react";
import { Link } from "react-router";

export type PreviewRemedy = PreviewWarning["remedies"][number];

/**
 * What one kind of remedy is on the client: who may take it, the fix itself
 * when Trenova has an API for it, and the place it is made by hand.
 *
 * The server says which remedy applies to a warning and to which record
 * (pkg/remedy); everything a person reads or presses is kept here, in one
 * entry per kind. A warning with a remedy this client has no entry for is
 * shown as the warning alone, so a remedy added on the server never breaks a
 * surface that has not learned it yet.
 */
export type RemedyDefinition = {
  /** What the reader must be allowed to do to be offered the remedy at all. */
  needs: { resource: string; operation: OperationType };
  /**
   * The fix, taken where the warning is read. It answers with what happened,
   * in a sentence, which stays under the warning.
   */
  act?: {
    label: (t: TranslateFn, remedy: PreviewRemedy) => string;
    /** `again` is set when the remedy was already taken and is being taken once more. */
    run: (remedy: PreviewRemedy, t: TranslateFn, again: boolean) => Promise<string>;
    /**
     * What is said of a remedy the server reports as already taken and still
     * waiting to take effect, and what the button that takes it again says.
     * The server is where this is remembered, so it reads the same after a
     * refresh, on another device and for another person.
     */
    taken: (t: TranslateFn, remedy: PreviewRemedy) => string;
    again: (t: TranslateFn) => string;
  };
  /** Where the fix is made by hand: beside the action, and alone when there is none. */
  place?: (remedy: PreviewRemedy) => { to: string; label: (t: TranslateFn) => string } | null;
};

export const REMEDIES: Readonly<Record<string, RemedyDefinition>> = {
  invite_driver_to_portal: {
    needs: { resource: Resource.DriverPortal, operation: Operation.Create },
    act: {
      label: (t, remedy) =>
        remedy.recordLabel !== ""
          ? t("Invite {0} to the driver portal", remedy.recordLabel)
          : t("Invite them to the driver portal"),
      taken: (t, remedy) =>
        remedy.takenNote !== ""
          ? t(
              "Invitation sent to {1} on {0}. Messages reach them once they accept it.",
              formatUnixDateMedium(remedy.takenAt, { fallback: "" }),
              remedy.takenNote,
            )
          : t(
              "Invitation sent on {0}. Messages reach them once they accept it.",
              formatUnixDateMedium(remedy.takenAt, { fallback: "" }),
            ),
      again: (t) => t("Send it again"),
      run: async (remedy, t, again) => {
        const result = await inviteWorkerToPortal({
          workerId: remedy.recordId,
          replacePending: again,
        });
        return result.emailSent
          ? t(
              "Invitation emailed to {0}. Messages reach them once they accept it.",
              result.invitation.email,
            )
          : t(
              "Invitation created, but no email could be sent. Share the link from their portal tab; messages reach them once they accept it.",
            );
      },
    },
    place: (remedy) =>
      remedy.recordId === ""
        ? null
        : {
            to: recordPath("worker", remedy.recordId),
            label: (t) =>
              remedy.recordLabel !== "" ? t("Open {0}", remedy.recordLabel) : t("Open the worker"),
          },
  },
  set_up_email_profile: {
    needs: { resource: Resource.EmailProfile, operation: Operation.Create },
    place: () => ({
      to: "/organization/email-profiles",
      label: (t) => t("Set up an email profile"),
    }),
  },
};

/** The remedies of a warning this client knows how to offer. */
export function knownRemedies(warning: Pick<PreviewWarning, "remedies">): PreviewRemedy[] {
  return warning.remedies.filter((remedy) => Object.hasOwn(REMEDIES, remedy.kind));
}

/**
 * What a person can do about a warning, offered under it: the fix as a solid
 * button where there is one, and the place to make it by hand as a link. Nothing is
 * drawn for a reader who may not take any of them; the server refuses the
 * action again whatever the screen shows.
 */
export function PreviewRemedies({ warning }: { warning: Pick<PreviewWarning, "remedies"> }) {
  const remedies = knownRemedies(warning);
  if (remedies.length === 0) {
    return null;
  }

  return (
    <>
      {remedies.map((remedy) => (
        <RemedyOffer key={`${remedy.kind}:${remedy.recordId}`} remedy={remedy} />
      ))}
    </>
  );
}

function RemedyOffer({ remedy }: { remedy: PreviewRemedy }) {
  const t = useT();
  const queryClient = useQueryClient();
  const definition = REMEDIES[remedy.kind];
  const { allowed } = usePermission(
    definition?.needs.resource ?? Resource.Assistant,
    definition?.needs.operation ?? Operation.Read,
  );
  const [outcome, setOutcome] = useState("");
  const sending = useRef(false);
  const act = definition?.act;
  const alreadyTaken = remedy.takenAt != null && remedy.takenAt > 0;
  const taken = useApiMutation({
    mutationFn: async () => (act ? act.run(remedy, t, alreadyTaken) : ""),
    onSuccess: async (said) => {
      setOutcome(said);
      await queryClient.invalidateQueries({ queryKey: agentPreview._def });
    },
    onSettled: () => {
      sending.current = false;
    },
  });
  const take = () => {
    if (sending.current) {
      return;
    }
    sending.current = true;
    taken.mutate();
  };

  if (!definition || !allowed) {
    return null;
  }
  const place = definition.place?.(remedy) ?? null;

  // Taken: what is left to show is that it was, and the way to do it again.
  // The link to do it by hand goes, since it has been done.
  const said = outcome !== "" ? outcome : alreadyTaken && act ? act.taken(t, remedy) : "";
  if (said !== "") {
    return (
      <span className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1">
        <span className="flex items-start gap-1.5 text-xs text-success-foreground" role="status">
          <CheckIcon aria-hidden className="mt-0.5 size-3 shrink-0" />
          {said}
        </span>
        {act && (
          <Button size="sm" variant="link" isLoading={taken.isPending} onClick={take}>
            {act.again(t)}
          </Button>
        )}
      </span>
    );
  }

  return (
    <div className="mt-1 flex flex-wrap items-center gap-2">
      {act && (
        <Button size="sm" isLoading={taken.isPending} onClick={take}>
          {act.label(t, remedy)}
        </Button>
      )}
      {place && (
        <Link className={buttonVariants({ variant: "link", size: "sm" })} to={place.to}>
          {place.label(t)}
        </Link>
      )}
    </div>
  );
}
