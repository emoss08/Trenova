import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Textarea } from "@/components/ui/textarea";
import { Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import { type NodeData, type WorkflowNodeType } from "./workflow-builder";

export function NodeConfigPanel({
  node,
  onUpdate,
  onDelete,
  onClose,
}: {
  node: WorkflowNodeType | null;
  onUpdate: (nodeId: string, data: Partial<NodeData>) => void;
  onDelete: () => void;
  onClose: () => void;
}) {
  const [label, setLabel] = useState("");
  const [description, setDescription] = useState("");
  const [configJson, setConfigJson] = useState("{}");

  useEffect(() => {
    if (node) {
      setLabel(node.data.label || "");
      setDescription(node.data.config?.description || "");
      setConfigJson(JSON.stringify(node.data.config || {}, null, 2));
    }
  }, [node]);

  const handleSave = () => {
    if (!node) return;

    let config = {};
    try {
      config = JSON.parse(configJson);
    } catch (e) {
      // Keep existing config if JSON is invalid
      config = node.data.config || {};
    }

    onUpdate(node.id, {
      label,
      config: {
        ...config,
        description,
      },
    });
    onClose();
  };

  return (
    <Sheet open={!!node} onOpenChange={(open) => !open && onClose()}>
      <SheetContent className="w-[400px] sm:w-[540px]">
        <SheetHeader>
          <SheetTitle>Configure Node</SheetTitle>
          <SheetDescription>
            Customize the behavior and appearance of this node
          </SheetDescription>
        </SheetHeader>

        {node && (
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="label">Label</Label>
              <Input
                id="label"
                value={label}
                onChange={(e) => setLabel(e.target.value)}
                placeholder="Node label"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="description">Description</Label>
              <Textarea
                id="description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Brief description of what this node does"
                rows={3}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="config">Configuration (JSON)</Label>
              <Textarea
                id="config"
                value={configJson}
                onChange={(e) => setConfigJson(e.target.value)}
                placeholder='{"key": "value"}'
                className="font-mono text-sm"
                rows={8}
              />
              <p className="text-xs text-muted-foreground">
                Advanced configuration in JSON format
              </p>
            </div>

            <div className="space-y-2">
              <Label>Node Type</Label>
              <div className="rounded-md bg-muted px-3 py-2 text-sm font-medium">
                {node.data.nodeType || node.type}
              </div>
            </div>

            <div className="space-y-2">
              <Label>Node ID</Label>
              <div className="font-mono text-xs text-muted-foreground">
                {node.id}
              </div>
            </div>
          </div>
        )}

        <SheetFooter className="gap-2">
          <Button
            variant="destructive"
            size="sm"
            onClick={onDelete}
            className="mr-auto"
          >
            <Trash2 className="mr-2 size-4" />
            Delete Node
          </Button>
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={handleSave}>Save Changes</Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
