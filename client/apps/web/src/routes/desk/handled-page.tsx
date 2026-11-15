import { useParams } from "@tanstack/react-router";
import { DeskHandled } from "./_components/handled/desk-handled";

export function DeskHandledPage() {
  const { id = "" } = useParams({ strict: false });
  return <DeskHandled key={id} id={id} />;
}
