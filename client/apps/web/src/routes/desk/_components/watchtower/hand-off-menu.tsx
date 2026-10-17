import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import type { WatchtowerAgent } from "@/lib/graphql/watchtower";
import { watchtower } from "@/lib/queries/watchtower";
import { useQuery } from "@tanstack/react-query";
import { ChevronUpIcon } from "@trenova/shared/components/icons";
import { ButtonGroup, ButtonGroupSeparator } from "@trenova/shared/components/ui/button-group";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ReactNode } from "react";
import { QueueButton } from "../queue/queue-bar";

/**
 * Hand off. With an agent suggested it is a button group: the button hands the
 * item to that agent and the caret opens the other agents, upward, over the
 * bar. With none suggested there is nothing for a first half to do that the
 * list does not, so it is one button that opens the list.
 */
export function WatchtowerHandOff({
  suggested,
  open,
  onOpenChange,
  onHandOff,
  disabled,
}: {
  suggested: WatchtowerAgent | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onHandOff: (agentId?: string) => void;
  disabled: boolean;
}) {
  const t = useT();
  const agentsQuery = useQuery({ ...watchtower.handOffAgents(), staleTime: 60_000 });
  const others = (agentsQuery.data ?? []).filter((agent) => agent.id !== suggested?.id);

  const menu = (
    <DropdownMenuContent side="top" align="end" sideOffset={8} className="w-72.5">
      {agentsQuery.isLoading ? (
        <MenuNote>{t("Reading the agents…")}</MenuNote>
      ) : others.length === 0 ? (
        <MenuNote>
          {suggested
            ? t("No other agent can take this")
            : t("No agent can take this yet. Start one from AI Control.")}
        </MenuNote>
      ) : (
        others.map((agent) => (
          <DropdownMenuItem
            key={agent.id}
            title={agent.name}
            description={agent.description || undefined}
            startContent={<DeskAgentTile agent={agent} size="sm" />}
            onClick={() => onHandOff(agent.id)}
          />
        ))
      )}
    </DropdownMenuContent>
  );

  if (suggested === null) {
    return (
      <DropdownMenu open={open} onOpenChange={onOpenChange}>
        <DropdownMenuTrigger
          disabled={disabled}
          render={
            <QueueButton
              ink
              shortcut="H"
              disabled={disabled}
              title={t("Choose an agent to hand it to")}
            />
          }
        >
          {t("Hand off")}
          <ChevronUpIcon className="-ml-1 size-3" strokeWidth={2.6} />
        </DropdownMenuTrigger>
        {menu}
      </DropdownMenu>
    );
  }

  return (
    <ButtonGroup className="flex-none">
      <QueueButton
        ink
        shortcut="H"
        disabled={disabled}
        title={t("Hand to {0}", suggested.name)}
        onClick={() => onHandOff(suggested.id)}
      >
        {t("Hand off")}
      </QueueButton>
      <ButtonGroupSeparator className="bg-ink-foreground/25" />
      <DropdownMenu open={open} onOpenChange={onOpenChange}>
        <DropdownMenuTrigger
          disabled={disabled}
          render={
            <QueueButton
              ink
              disabled={disabled}
              className="px-2.5 @max-[900px]/mainc:pl-2.5"
              aria-label={t("Choose who to hand it to")}
            />
          }
        >
          <ChevronUpIcon className="size-3" strokeWidth={2.6} />
        </DropdownMenuTrigger>
        {menu}
      </DropdownMenu>
    </ButtonGroup>
  );
}

/**
 * A line in the menu that is not an item: the list loading, or there being
 * nobody to choose. Base UI's group label belongs to a group of items, so this
 * is plain text the menu announces as it changes.
 */
function MenuNote({ children }: { children: ReactNode }) {
  return (
    <p role="status" className="px-1.5 py-1 text-xs text-muted-foreground">
      {children}
    </p>
  );
}
