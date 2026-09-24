import { DataTableLink } from "@/components/data-table/_components/data-table-components";
import { agentSubjectPath } from "@/lib/agent-subjects";

type AgentSubjectCellProps = {
  subjectType: string;
  subjectId: string;
};

export function AgentSubjectCell({ subjectType, subjectId }: AgentSubjectCellProps) {
  const href = agentSubjectPath(subjectType, subjectId);

  return (
    <span className="flex flex-col leading-tight">
      <span>{subjectType}</span>
      {href === null ? (
        <span className="text-muted-foreground font-mono text-xs">{subjectId}</span>
      ) : (
        <DataTableLink text={subjectId} href={href} className="font-mono text-xs" />
      )}
    </span>
  );
}
