import { appPath } from "@/lib/app-path";
import { useCallback } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useAIControlNavigation } from "../../use-ai-control-navigation";
import type { NovaTarget } from "./use-nova-segments";

const WATCHTOWER_PATH = `${appPath("/desk/needs")}?lens=watchtower`;

/** Goes where a linked part of a Nova sentence leads. */
export function useNovaTargets(): (target: NovaTarget) => void {
  const navigate = useNavigate();
  const go = useAIControlNavigation();

  return useCallback(
    (target: NovaTarget) => {
      switch (target.kind) {
        case "watchtower":
          void navigate({ to: WATCHTOWER_PATH });
          return;
        case "providers":
        case "routing":
          go({ tab: "providers" });
          return;
        case "agents":
          go({ tab: "agents", agentFilter: target.filter });
          return;
        case "provider":
          go({ tab: "providers", panel: { mode: "edit", entityId: target.providerId } });
          return;
      }
    },
    [go, navigate],
  );
}
