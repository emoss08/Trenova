import { BadgeAttrProps } from "@/components/status-badge";
import { Badge } from "@/components/ui/badge";
import { ModelTypes, OperationTypes } from "@/types/ai-logs";

export function ModelBadge({ model }: { model: ModelTypes }) {
  const modelAttributes: Record<ModelTypes, BadgeAttrProps> = {
    [ModelTypes.GPT5_NANO]: {
      variant: "indigo",
      text: "GPT-5 Nano",
    },
    [ModelTypes.GPT5_NANO_2025_08_07]: {
      variant: "purple",
      text: "GPT-5 Nano 2025-08-07",
    },
    [ModelTypes.OMNI_MODERATION_LATEST]: {
      variant: "orange",
      text: "Omni Moderation Latest",
    },
  };

  return (
    <Badge variant={modelAttributes[model].variant}>
      {modelAttributes[model].text}
    </Badge>
  );
}

export function OperationBadge({ operation }: { operation: OperationTypes }) {
  const operationAttributes: Record<OperationTypes, BadgeAttrProps> = {
    [OperationTypes.CLASSIFY_LOCATION]: {
      variant: "active",
      text: "Classify Location",
    },
  };

  return (
    <Badge variant={operationAttributes[operation].variant}>
      {operationAttributes[operation].text}
    </Badge>
  );
}
