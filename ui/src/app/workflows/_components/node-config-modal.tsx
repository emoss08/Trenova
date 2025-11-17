import { FieldLabel } from "@/components/fields/field-components";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { WorkflowNodeType } from "@/types/workflow";
import { useEffect, useState } from "react";
import { ActionConfigForm } from "./config-forms/action-config-form";
import { ActionTypeSelector } from "./config-forms/action-type-selector";
import { ConditionConfigForm } from "./config-forms/condition-config-form";

interface NodeConfigModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  node: WorkflowNodeType | null;
  onSave: (
    nodeId: string,
    config: Record<string, any>,
    actionType?: string,
  ) => void;
}

export default function NodeConfigModal({
  open,
  onOpenChange,
  node,
  onSave,
}: NodeConfigModalProps) {
  const [selectedActionType, setSelectedActionType] = useState<
    string | undefined
  >(node?.data.actionType);

  // Update selectedActionType when node changes
  useEffect(() => {
    if (node?.data.actionType) {
      setSelectedActionType(node.data.actionType);
    }
  }, [node?.id, node?.data.actionType]);

  if (!node) return null;

  const handleSave = (config: Record<string, any>) => {
    onSave(node.id, config, selectedActionType);
    onOpenChange(false);
  };

  const handleCancel = () => {
    onOpenChange(false);
  };

  const getModalTitle = () => {
    switch (node.type) {
      case "action":
        return "Configure Action Node";
      case "condition":
        return "Configure Condition Node";
      case "delay":
        return "Configure Delay Node";
      case "loop":
        return "Configure Loop Node";
      default:
        return "Configure Node";
    }
  };

  const getModalDescription = () => {
    switch (node.type) {
      case "action":
        return "Select an action type and configure its parameters";
      case "condition":
        return "Define the condition that will determine the execution path";
      case "delay":
        return "Set how long the workflow should wait before continuing";
      case "loop":
        return "Configure the loop iteration settings";
      default:
        return "Configure this node's settings";
    }
  };

  const renderForm = () => {
    switch (node.type) {
      case "action":
        return (
          <div className="space-y-4">
            <div className=" px-4 pt-2">
              <FieldLabel label="Action Type" required />
              <ActionTypeSelector
                value={selectedActionType}
                onChange={setSelectedActionType}
              />
            </div>
            {selectedActionType && (
              <div className="border-t border-border pt-2">
                <ActionConfigForm
                  key={`${node.id}-${selectedActionType}`}
                  actionType={selectedActionType}
                  initialConfig={node.data.config || {}}
                  onSave={handleSave}
                  onCancel={handleCancel}
                />
              </div>
            )}

            {!selectedActionType && (
              <p className="text-sm text-muted-foreground">
                Select an action type to configure its settings
              </p>
            )}
          </div>
        );
      case "condition":
        return (
          <ConditionConfigForm
            key={node.id}
            initialConfig={node.data.config || {}}
            onSave={handleSave}
            onCancel={handleCancel}
          />
        );

      case "delay":
        return (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Delay configuration coming soon...
            </p>
          </div>
        );

      case "loop":
        return (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Loop configuration coming soon...
            </p>
          </div>
        );

      default:
        return (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Configuration for this node type is not yet available.
            </p>
          </div>
        );
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{getModalTitle()}</DialogTitle>
          <DialogDescription>{getModalDescription()}</DialogDescription>
        </DialogHeader>

        {renderForm()}
      </DialogContent>
    </Dialog>
  );
}
