import { queries } from "@/lib/queries";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { threadListQuery } from "@/lib/thread-list";
import { useParams } from "react-router";
import { DeskLayout } from "./_components/desk-layout";
import "@/components/desk-chat/desk-chat.css";
import "./_styles/desk-v2.css";
import "./_styles/desk-agent.css";
import "./_styles/desk-case.css";
import "./_styles/desk-case-checklists.css";

export const prefetch: RoutePrefetch = () => [
  threadListQuery(),
  queries.assistant.myAgents(),
];

/** The Desk's frame; the page inside it is the route's outlet. */
export function DeskPage() {
  const { threadId } = useParams<{ threadId?: string }>();

  return <DeskLayout activeThreadId={threadId ?? null} />;
}
