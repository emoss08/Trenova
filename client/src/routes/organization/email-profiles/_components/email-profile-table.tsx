import { DataTable } from "@/components/data-table/data-table";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { apiService } from "@/services/api";
import type { RowAction } from "@/types/data-table";
import { testEmailProfileRequestSchema, type EmailProfile } from "@/types/email";
import { Resource } from "@/types/permission";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { Row } from "@tanstack/react-table";
import { Loader2Icon, SendIcon, TrashIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { getColumns } from "./email-profile-columns";
import { emailProfileQueryKey } from "./email-profile-constants";
import { EmailProfilePanel } from "./email-profile-panel";

export default function EmailProfileTable() {
  const queryClient = useQueryClient();
  const columns = useMemo(() => getColumns(), []);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [testDialogOpen, setTestDialogOpen] = useState(false);
  const [selectedProfile, setSelectedProfile] = useState<EmailProfile | null>(null);
  const [testRecipient, setTestRecipient] = useState("");

  const deleteMutation = useMutation({
    mutationFn: async (profile: EmailProfile) => {
      if (!profile.id) {
        throw new Error("Email profile ID is required");
      }
      await apiService.emailService.deleteProfile(profile.id);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: [emailProfileQueryKey] });
      await queryClient.invalidateQueries({ queryKey: ["email", "assignments"] });
      toast.success("Email profile deleted");
      setDeleteDialogOpen(false);
      setSelectedProfile(null);
    },
    onError: (error) => {
      toast.error("Failed to delete email profile", {
        description: error instanceof Error ? error.message : "An unexpected error occurred",
      });
    },
  });

  const testSendMutation = useMutation({
    mutationFn: async ({ profile, to }: { profile: EmailProfile; to: string }) => {
      if (!profile.id) {
        throw new Error("Email profile ID is required");
      }
      return apiService.emailService.testProfile(
        profile.id,
        testEmailProfileRequestSchema.parse({
          to,
          subject: "Trenova email profile test",
          text: `This test verifies ${profile.name} can send through ${profile.provider}.`,
          html: `<p>This test verifies ${profile.name} can send through ${profile.provider}.</p>`,
        }),
      );
    },
    onSuccess: () => {
      toast.success("Test email queued");
      setTestDialogOpen(false);
      setTestRecipient("");
      setSelectedProfile(null);
    },
    onError: (error) => {
      toast.error("Failed to queue test email", {
        description: error instanceof Error ? error.message : "An unexpected error occurred",
      });
    },
  });

  const openDeleteDialog = useCallback((row: Row<EmailProfile>) => {
    setSelectedProfile(row.original);
    setDeleteDialogOpen(true);
  }, []);

  const openTestDialog = useCallback((row: Row<EmailProfile>) => {
    setSelectedProfile(row.original);
    setTestRecipient("");
    setTestDialogOpen(true);
  }, []);

  const contextMenuActions = useMemo<RowAction<EmailProfile>[]>(
    () => [
      {
        id: "send-test",
        label: "Send Test",
        icon: SendIcon,
        onClick: openTestDialog,
        disabled: (row) => row.original.status !== "Active",
      },
      {
        id: "delete",
        label: "Delete",
        icon: TrashIcon,
        variant: "destructive",
        onClick: openDeleteDialog,
      },
    ],
    [openDeleteDialog, openTestDialog],
  );

  return (
    <>
      <DataTable<EmailProfile>
        name="Email Profile"
        link="/email-profiles/"
        queryKey={emailProfileQueryKey}
        exportModelName="email-profile"
        resource={Resource.EmailProfile}
        columns={columns}
        contextMenuActions={contextMenuActions}
        TablePanel={EmailProfilePanel}
      />

      <Dialog open={testDialogOpen} onOpenChange={setTestDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Send Test Email</DialogTitle>
            <DialogDescription>
              Queue a test message from {selectedProfile?.name ?? "this email profile"}.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-2">
            <label className="text-xs font-medium text-muted-foreground" htmlFor="test-recipient">
              Recipient Email
            </label>
            <Input
              id="test-recipient"
              type="email"
              placeholder="recipient@example.com"
              value={testRecipient}
              onChange={(event) => setTestRecipient(event.target.value)}
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setTestDialogOpen(false)}>
              Cancel
            </Button>
            <Button
              type="button"
              disabled={!selectedProfile || !testRecipient}
              isLoading={testSendMutation.isPending}
              loadingText="Sending..."
              onClick={() => {
                if (selectedProfile) {
                  testSendMutation.mutate({ profile: selectedProfile, to: testRecipient });
                }
              }}
            >
              <SendIcon />
              Send Test
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia>
              <TrashIcon />
            </AlertDialogMedia>
            <AlertDialogTitle>Delete Email Profile</AlertDialogTitle>
            <AlertDialogDescription>
              Delete {selectedProfile?.name ?? "this email profile"} and remove it from any purpose
              assignment. This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={!selectedProfile || deleteMutation.isPending}
              onClick={() => {
                if (selectedProfile) {
                  deleteMutation.mutate(selectedProfile);
                }
              }}
            >
              {deleteMutation.isPending && <Loader2Icon className="mr-2 size-4 animate-spin" />}
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
