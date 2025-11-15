import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Clock, GitBranch, Play, Repeat, Square, Zap } from "lucide-react";

const nodeTypes = [
  { type: "trigger", label: "Trigger", icon: Zap, description: "Start point" },
  {
    type: "action",
    label: "Action",
    icon: Play,
    description: "Execute action",
  },
  {
    type: "condition",
    label: "Condition",
    icon: GitBranch,
    description: "Branch logic",
  },
  { type: "loop", label: "Loop", icon: Repeat, description: "Iterate" },
  { type: "delay", label: "Delay", icon: Clock, description: "Wait" },
  { type: "end", label: "End", icon: Square, description: "End point" },
];

export function NodePalette({
  onAddNode,
}: {
  onAddNode: (type: string) => void;
}) {
  return (
    <Card className="w-56 shadow-lg">
      <CardHeader className="pb-3">
        <CardTitle className="text-sm">Add Node</CardTitle>
      </CardHeader>
      <CardContent className="space-y-1 pb-3">
        {nodeTypes.map(({ type, label, icon: Icon, description }) => (
          <button
            key={type}
            className="flex w-full cursor-pointer flex-row items-center gap-1 rounded-md border border-border p-1 hover:bg-accent"
            onClick={() => onAddNode(type)}
          >
            <Icon className="size-4 shrink-0" />
            <div className="flex flex-col items-start">
              <span className="text-sm">{label}</span>
              <span className="text-xs text-muted-foreground">
                {description}
              </span>
            </div>
          </button>
        ))}
      </CardContent>
    </Card>
  );
}
