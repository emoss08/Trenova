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
import { respondToMyAssignment, type PortalLoad } from "@/lib/graphql/driver-portal";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { CheckIcon, XIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { useDashFeatures } from "./use-dash-features";

export function AssignmentResponseCard({ load }: { load: PortalLoad }) {
  const queryClient = useQueryClient();
  const features = useDashFeatures();
  const [declineOpen, setDeclineOpen] = useState(false);
  const [reason, setReason] = useState("");

  const respond = useMutation({
    mutationFn: (accept: boolean) =>
      respondToMyAssignment({
        assignmentId: load.assignmentId,
        accept,
        reason: accept ? undefined : reason.trim(),
      }),
    onSuccess: async (_, accept) => {
      toast.success(
        accept
          ? "You're on it — dispatch can see you accepted."
          : "Dispatch has been notified so they can replan.",
      );
      setDeclineOpen(false);
      setReason("");
      await queryClient.invalidateQueries({ queryKey: ["dash-loads"] });
    },
    onError: (error: Error) => toast.error(error.message || "We couldn't send your response."),
  });

  const respondable =
    features.requireLoadAcknowledgment && (load.status === "New" || load.status === "Assigned");

  if (load.ackStatus === "Declined") {
    return (
      <div className="rounded-2xl border border-border bg-card p-4">
        <p className="text-sm font-semibold">You declined this load</p>
        <p className="mt-1 text-xs text-muted-foreground">
          Dispatch has been notified. If plans changed, call your dispatcher.
        </p>
      </div>
    );
  }

  if (load.ackStatus !== "Pending" || !respondable) {
    return null;
  }

  return (
    <div className="rounded-2xl border border-primary/30 bg-primary/5 p-4">
      <p className="text-sm font-semibold">Can you take this load?</p>
      <p className="mt-1 text-xs text-muted-foreground">
        Let dispatch know so they can plan around you.
      </p>
      <div className={features.allowLoadRefusals ? "mt-3 grid grid-cols-2 gap-2" : "mt-3"}>
        <Button
          className="h-11 w-full"
          disabled={respond.isPending}
          onClick={() => respond.mutate(true)}
        >
          <CheckIcon className="size-4" />
          Accept
        </Button>
        {features.allowLoadRefusals ? (
          <Button
            variant="outline"
            className="h-11 w-full"
            disabled={respond.isPending}
            onClick={() => setDeclineOpen(true)}
          >
            <XIcon className="size-4" />
            Decline
          </Button>
        ) : null}
      </div>

      <Drawer open={declineOpen} onOpenChange={setDeclineOpen}>
        <DrawerContent>
          <DrawerHeader>
            <DrawerTitle>Decline this load?</DrawerTitle>
            <DrawerDescription>
              Tell dispatch why so they can replan — hours, home time, equipment, anything.
            </DrawerDescription>
          </DrawerHeader>
          <div className="px-4">
            <Textarea
              value={reason}
              onChange={(event) => setReason(event.target.value)}
              placeholder="Why can't you take it?"
              rows={3}
              maxLength={1000}
            />
          </div>
          <DrawerFooter>
            <Button
              variant="destructive"
              className="h-11"
              disabled={reason.trim().length === 0 || respond.isPending}
              onClick={() => respond.mutate(false)}
            >
              {respond.isPending ? "Sending..." : "Decline load"}
            </Button>
          </DrawerFooter>
        </DrawerContent>
      </Drawer>
    </div>
  );
}
