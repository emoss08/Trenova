import { useParams } from "react-router";
import { DeskView } from "./_components/view/desk-view";

export function DeskViewPage() {
  const { viewThreadId = "", slug = "" } = useParams<{ viewThreadId: string; slug: string }>();
  return <DeskView key={`${viewThreadId}:${slug}`} threadId={viewThreadId} slug={slug} />;
}
