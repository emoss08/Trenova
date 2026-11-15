import { useParams } from "@tanstack/react-router";
import { DeskRecordCase } from "./_components/record-case/desk-record-case";

export function DeskCasePage() {
  const { kind = "", id = "" } = useParams({ strict: false });
  return <DeskRecordCase key={`${kind}:${id}`} kind={kind} id={id} />;
}
