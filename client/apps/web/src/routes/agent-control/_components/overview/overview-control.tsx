import type { AgentControl } from "@/lib/graphql/agent-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { Ic } from "../kit/ic";

import { usePauseAgents } from "./use-pause-agents";
import { Button } from "@trenova/shared/components/ui/button";
import { HoldButton } from "@trenova/shared/components/ui/hold-button";
import { PauseCircleIcon } from "@trenova/shared/components/icons";

type OverviewControlProps = {
  control: AgentControl | undefined;
  noProvider: boolean;
  canUpdate: boolean;
  onConnectProvider: () => void;
};

/**
 * The one control beside the overview's sentence: connect a provider when there is none,
 * resume when every agent is paused, and otherwise a hold to pause them all.
 */
export function OverviewControl({
  control,
  noProvider,
  canUpdate,
  onConnectProvider,
}: OverviewControlProps) {
  const t = useT();
  const pause = usePauseAgents(control);

  if (noProvider) {
    return (
      <>
        <Button type="button" variant="default" size="lg" onClick={onConnectProvider}>
          <Ic n="plus" s={13} />
          {t("Connect a provider")}
        </Button>
        <span>{t("Takes about a minute")}</span>
      </>
    );
  }

  if (!control || !canUpdate) {
    return null;
  }

  if (control.shadowMode) {
    return (
      <>
        <Button
          type="button"
          variant="default" size="lg"
          disabled={pause.isPending}
          onClick={() => pause.mutate(false)}
        >
          <Ic n="play" s={12} />
          {t("Resume agents")}
        </Button>
        <span>{t("Every agent is paused")}</span>
      </>
    );
  }

  return (
    <>
      <HoldButton
        size="lg"
        className="min-w-57.5"
        disabled={pause.isPending}
        onDone={() => pause.mutate(true)}
      >
        <PauseCircleIcon className="text-warning-foreground size-3.5" />
        {t("Hold to pause all agents")}
      </HoldButton>
      <span>{t("They keep running in shadow")}</span>
    </>
  );
}
