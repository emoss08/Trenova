/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Zap,
  Play,
  GitBranch,
  Repeat,
  Clock,
  Square,
} from "lucide-react";

const nodeTypes = [
  { type: "trigger", label: "Trigger", icon: Zap, description: "Start point" },
  { type: "action", label: "Action", icon: Play, description: "Execute action" },
  { type: "condition", label: "Condition", icon: GitBranch, description: "Branch logic" },
  { type: "loop", label: "Loop", icon: Repeat, description: "Iterate" },
  { type: "delay", label: "Delay", icon: Clock, description: "Wait" },
  { type: "end", label: "End", icon: Square, description: "End point" },
];

export function NodePalette({ onAddNode }: { onAddNode: (type: string) => void }) {
  return (
    <Card className="w-56 shadow-lg">
      <CardHeader className="pb-3">
        <CardTitle className="text-sm">Add Node</CardTitle>
      </CardHeader>
      <CardContent className="space-y-1 pb-3">
        {nodeTypes.map(({ type, label, icon: Icon, description }) => (
          <Button
            key={type}
            variant="outline"
            size="sm"
            className="w-full justify-start"
            onClick={() => onAddNode(type)}
          >
            <Icon className="mr-2 size-4" />
            <div className="flex flex-1 flex-col items-start">
              <span className="text-sm">{label}</span>
              <span className="text-muted-foreground text-xs">{description}</span>
            </div>
          </Button>
        ))}
      </CardContent>
    </Card>
  );
}
