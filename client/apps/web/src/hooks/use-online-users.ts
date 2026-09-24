import {
  REALTIME_USERS_SCOPE,
  realtimeClient,
  type RealtimePresenceMember,
} from "@trenova/shared/services/realtime";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useEffect, useMemo, useState } from "react";

/**
 * Who in the tenant has the app open. Membership is kept by the realtime
 * stream itself: every open tab is one member, and a user is online while any
 * of their tabs is.
 */
export function useOnlineUsers() {
  const user = useAuthStore((state) => state.user);
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const [members, setMembers] = useState<RealtimePresenceMember[]>([]);
  const hasTenantContext = Boolean(
    isAuthenticated && user?.currentOrganizationId && user.businessUnitId,
  );

  useEffect(() => {
    if (!hasTenantContext) {
      return;
    }

    const unsubscribe = realtimeClient.subscribePresence(REALTIME_USERS_SCOPE, setMembers);
    return () => {
      unsubscribe();
      // Drop members from the previous tenant so a switch starts clean before
      // the next stream's snapshot repopulates.
      setMembers([]);
    };
  }, [hasTenantContext, user?.businessUnitId, user?.currentOrganizationId]);

  const onlineUserIDs = useMemo(
    () => (hasTenantContext ? new Set(members.map((member) => member.userId)) : new Set<string>()),
    [members, hasTenantContext],
  );

  return { onlineUserIDs };
}
