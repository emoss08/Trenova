import {
  extensionCommandsNamed,
  type SlashCommand,
} from "@/components/assistant/composer-commands";
import { assistant } from "@/lib/queries/assistant";
import type { CommandAvailability } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";

/**
 * How long the answer holds before a composer that mounts asks again. Turning
 * an extension on or off invalidates it at once (extension-detail.tsx); this
 * covers a change made on another device.
 */
const COMMANDS_STALE_MS = 60_000;

const NO_COMMANDS: readonly SlashCommand[] = [];

function selectCommands(availability: CommandAvailability): readonly SlashCommand[] {
  return extensionCommandsNamed(availability.commands);
}

/**
 * The extension commands (/browser, /search) offered for an agent: only those
 * whose extension is on, whose tools the agent holds or every agent is given,
 * and which the person may use. The server decides, so every one stays hidden
 * until it has answered, and when it cannot.
 */
export function useDeskExtensionCommands(
  agentId: string | null | undefined,
): readonly SlashCommand[] {
  const { data } = useQuery({
    ...assistant.commands(agentId ?? ""),
    enabled: agentId !== null && agentId !== undefined && agentId !== "",
    staleTime: COMMANDS_STALE_MS,
    refetchOnWindowFocus: true,
    retry: false,
    select: selectCommands,
  });

  return data ?? NO_COMMANDS;
}
