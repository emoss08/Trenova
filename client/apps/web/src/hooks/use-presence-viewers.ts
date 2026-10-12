import { realtimeClient, type RealtimePresenceMember } from "@trenova/shared/services/realtime";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useEffect, useMemo, useState } from "react";

export type PresenceViewer = {
  userId: string;
  name: string;
};

/** A presence scope and the endpoint that authorizes joining it. */
export type PresenceTarget = {
  scope: string;
  joinPath: string;
};

/**
 * Everyone else in a presence scope, one entry per person however many tabs they
 * have open. Joining is what makes the current user visible to them; the server ties
 * the membership to this tab's stream, so a closed tab leaves on its own. A null
 * target joins nothing.
 */
export function usePresenceViewers(target: PresenceTarget | null) {
  const userId = useAuthStore((state) => state.user?.id);
  const [members, setMembers] = useState<RealtimePresenceMember[]>([]);
  const scope = target?.scope;
  const joinPath = target?.joinPath;

  useEffect(() => {
    if (!scope || !joinPath || !userId) return;

    const unsubscribe = realtimeClient.subscribePresence(scope, setMembers);
    const leave = realtimeClient.joinScope(scope, joinPath);

    return () => {
      unsubscribe();
      leave();
      setMembers([]);
    };
  }, [scope, joinPath, userId]);

  const viewers = useMemo<PresenceViewer[]>(() => {
    const names = new Map<string, string>();
    for (const member of members) {
      if (member.userId === userId) continue;
      if (!names.get(member.userId)) names.set(member.userId, member.name);
    }
    return Array.from(names, ([viewerId, name]) => ({
      userId: viewerId,
      name: name || "Someone",
    }));
  }, [members, userId]);

  return { viewers };
}
