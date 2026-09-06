import { trainingHealthMeta } from "../lib/training";
import { cn } from "../lib/utils";
import type { WorkerTrainingHealth } from "../types/worker-training";
import { Badge } from "./ui/badge";

type TrainingHealthBadgeProps = {
  health: WorkerTrainingHealth;
  className?: string;
};

export function TrainingHealthBadge({ health, className }: TrainingHealthBadgeProps) {
  const meta = trainingHealthMeta(health);
  return (
    <Badge variant={meta.badgeVariant} className={cn("gap-1.5 whitespace-nowrap", className)}>
      {meta.label}
    </Badge>
  );
}
