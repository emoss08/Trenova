import type { RoutePrefetch } from "@/lib/route-prefetch";
import { DeskScheduledPage } from "./_components/scheduled/desk-scheduled-page";
import { allSchedulesQuery } from "./_components/scheduled/schedules-query";

export const prefetch: RoutePrefetch = () => [allSchedulesQuery()];

export function DeskScheduledRoutePage() {
  return <DeskScheduledPage />;
}
