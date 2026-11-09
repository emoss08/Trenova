import { useParams } from "react-router";
import { DeskRecordCase } from "./_components/record-case/desk-record-case";

export function DeskCasePage() {
  const { kind = "", id = "" } = useParams();
  return <DeskRecordCase key={`${kind}:${id}`} kind={kind} id={id} />;
}
