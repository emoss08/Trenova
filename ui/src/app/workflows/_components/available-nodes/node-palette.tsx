import { Icon } from "@/components/ui/icons";
import { ScrollArea } from "@/components/ui/scroll-area";
import { faShareNodes } from "@fortawesome/pro-regular-svg-icons";
import { useReactFlow } from "@xyflow/react";
import { Clock, GitBranch, Play, Repeat, Square, Zap } from "lucide-react";
import { nanoid } from "nanoid";

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

export default function AvailableNodes() {
  const { addNodes } = useReactFlow();
  const nodeId = nanoid();

  return (
    <div className="flex h-[80vh] w-80 flex-col">
      <div className="flex flex-col items-center justify-center gap-1 p-4">
        <div className="flex size-12 items-center justify-center rounded-full bg-primary">
          <Icon
            icon={faShareNodes}
            className="size-6 shrink-0 text-background"
          />
        </div>
        <span className="text-xl font-medium">Available Nodes</span>
        <span className="text-sm text-muted-foreground">
          Drag and drop nodes to your workflow
        </span>
      </div>

      <ScrollArea className="flex h-full flex-col rounded-b-lg bg-transparent px-4">
        <div className="flex h-full flex-col gap-2">
          {nodeTypes.map(({ type, label, icon: Icon, description }) => (
            <button
              key={type}
              className="flex w-full cursor-pointer flex-row items-center gap-2 rounded-md border border-border px-2 py-1 hover:bg-accent"
              onClick={() =>
                addNodes([
                  {
                    id: `${nodeId}-${type}`,
                    type,
                    position: {
                      x: Math.random() * 400 + 100,
                      y: Math.random() * 400 + 100,
                    },
                    data: {
                      label: type.charAt(0).toUpperCase() + type.slice(1),
                      nodeType: type,
                      config: {},
                    },
                  },
                ])
              }
            >
              <Icon className="size-4 shrink-0" />
              <div className="flex flex-col items-start leading-tight">
                <span className="text-sm">{label}</span>
                <span className="text-xs text-muted-foreground">
                  {description}
                </span>
              </div>
            </button>
          ))}
        </div>
      </ScrollArea>
    </div>
  );
}
