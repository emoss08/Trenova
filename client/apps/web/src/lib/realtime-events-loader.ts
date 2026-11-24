/**
 * loadRealtimeEvents fetches the module that reads realtime events. It is its own module so
 * the connection's retry can be exercised with a load that fails.
 */
export const loadRealtimeEvents = () => import("@/lib/realtime-events");
