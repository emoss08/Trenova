import { Button } from "@/components/ui/button";
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
} from "@/components/ui/drawer";
import { Textarea } from "@/components/ui/textarea";
import { createSettlementDispute, type PortalSettlementLine } from "@/lib/graphql/driver-portal";
import { cn } from "@/lib/utils";
import type { SettlementDisputeCategory } from "@trenova/graphql/generated/graphql";
import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import { disputeCategoryLabels } from "./portal-badges";

const categories = Object.keys(disputeCategoryLabels) as SettlementDisputeCategory[];

type DisputeDrawerProps = {
  settlementId: string;
  line: PortalSettlementLine | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function DisputeDrawer({ settlementId, line, open, onOpenChange }: DisputeDrawerProps) {
  const queryClient = useQueryClient();
  const [category, setCategory] = useState<SettlementDisputeCategory | null>(null);
  const [description, setDescription] = useState("");
  const [pending, setPending] = useState(false);

  const reset = () => {
    setCategory(null);
    setDescription("");
  };

  const handleSubmit = async () => {
    if (!category) {
      toast.info("Pick what kind of issue this is.");
      return;
    }
    if (description.trim().length === 0) {
      toast.info("Tell us what looks wrong so your carrier can fix it.");
      return;
    }
    setPending(true);
    try {
      await createSettlementDispute({
        settlementId,
        settlementLineId: line?.id,
        category,
        description: description.trim(),
      });
      toast.success("Sent to your carrier. You'll see updates under Money → Disputes.");
      await queryClient.invalidateQueries({ queryKey: ["dash-disputes"] });
      reset();
      onOpenChange(false);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "We couldn't send your dispute.");
    } finally {
      setPending(false);
    }
  };

  return (
    <Drawer
      open={open}
      onOpenChange={(next) => {
        if (!next) reset();
        onOpenChange(next);
      }}
    >
      <DrawerContent>
        <DrawerHeader>
          <DrawerTitle>Question about your pay?</DrawerTitle>
          <DrawerDescription>
            {line
              ? `About "${line.description}" — your carrier will review it and follow up.`
              : "Your carrier will review it and follow up on this statement."}
          </DrawerDescription>
        </DrawerHeader>

        <div className="flex flex-col gap-4 px-4">
          <div className="flex flex-wrap gap-2">
            {categories.map((value) => (
              <button
                key={value}
                type="button"
                onClick={() => setCategory(value)}
                className={cn(
                  "rounded-full border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground transition-colors",
                  category === value && "border-primary bg-primary text-primary-foreground",
                )}
              >
                {disputeCategoryLabels[value]}
              </button>
            ))}
          </div>
          <Textarea
            value={description}
            onChange={(event) => setDescription(event.target.value)}
            placeholder="Describe what looks wrong — loads, dates, amounts, anything that helps."
            rows={4}
            maxLength={4000}
          />
        </div>

        <DrawerFooter>
          <Button onClick={handleSubmit} disabled={pending} className="h-11">
            {pending ? "Sending..." : "Send to my carrier"}
          </Button>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  );
}
