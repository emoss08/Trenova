import type { AgentControl } from "@/lib/graphql/agent-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { Ic } from "../kit/ic";
import { Hold } from "../kit/layout";
import { usePauseAgents } from "./use-pause-agents";

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
        <button type="button" className="btn ink lg" onClick={onConnectProvider}>
          <Ic n="plus" s={13} />
          {t("Connect a provider")}
        </button>
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
        <button
          type="button"
          className="btn ink lg"
          disabled={pause.isPending}
          onClick={() => pause.mutate(false)}
        >
          <Ic n="play" s={12} />
          {t("Resume agents")}
        </button>
        <span>{t("Every agent is paused")}</span>
      </>
    );
  }

  return (
    <>
      <Hold
        label={t("Hold to pause all agents")}
        disabled={pause.isPending}
        onDone={() => pause.mutate(true)}
      />
      <span>{t("They keep running in shadow")}</span>
    </>
  );
}
