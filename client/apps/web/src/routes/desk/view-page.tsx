import { useParams } from "@tanstack/react-router";
import { DeskView } from "./_components/view/desk-view";

export function DeskViewPage() {
  const { viewThreadId = "", slug = "" } = useParams({ strict: false });
  return <DeskView key={`${viewThreadId}:${slug}`} threadId={viewThreadId} slug={slug} />;
}
