import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import type { WatchtowerAgent } from "@/lib/graphql/watchtower";
import { watchtower } from "@/lib/queries/watchtower";
import { useQuery } from "@tanstack/react-query";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { ChevronUpIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { ButtonGroup } from "@trenova/shared/components/ui/button-group";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ReactNode } from "react";

/**
 * Hand off. With an agent suggested it is a button group: the button names that
 * agent and hands the item to it and the caret opens the other agents, upward, over the
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
            <Button
              variant="outline"
              disabled={disabled}
              aria-keyshortcuts="H"
              title={t("Choose an agent to hand it to")}
            />
          }
        >
          <DeskIcon name="handoff" size={13} />
          {t("Hand off")}
          <span className="dk-w-kbd" aria-hidden>
            H
          </span>
          <ChevronUpIcon className="size-3" strokeWidth={2.4} />
        </DropdownMenuTrigger>
        {menu}
      </DropdownMenu>
    );
  }

  return (
    <ButtonGroup className="min-w-0 flex-none">
      <Button
        variant="outline"
        className="min-w-0"
        disabled={disabled}
        aria-keyshortcuts="H"
        onClick={() => onHandOff(suggested.id)}
      >
        <DeskIcon name="handoff" size={13} />
        <span className="truncate">{t("Hand off to {0}", suggested.name)}</span>
        <span className="dk-w-kbd" aria-hidden>
          H
        </span>
      </Button>
      <DropdownMenu open={open} onOpenChange={onOpenChange}>
        <DropdownMenuTrigger
          disabled={disabled}
          render={
            <Button
              variant="outline"
              size="icon"
              disabled={disabled}
              aria-label={t("Choose who to hand it to")}
            />
          }
        >
          <ChevronUpIcon className="size-3" strokeWidth={2.4} />
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
