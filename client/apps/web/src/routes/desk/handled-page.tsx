import { useParams } from "react-router";
import { DeskHandled } from "./_components/handled/desk-handled";

export function DeskHandledPage() {
  const { id = "" } = useParams();
  return <DeskHandled key={id} id={id} />;
}
