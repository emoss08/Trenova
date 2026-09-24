import { queries } from "@/lib/queries";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { useParams } from "react-router";
import { DeskLayout } from "./_components/desk-layout";

export const prefetch: RoutePrefetch = () => [
  queries.assistant.threads(),
  queries.assistant.myAgents(),
];

/** The Desk's frame; the page inside it is the route's outlet. */
export function DeskPage() {
  const { threadId } = useParams<{ threadId?: string }>();

  return <DeskLayout activeThreadId={threadId ?? null} />;
}
