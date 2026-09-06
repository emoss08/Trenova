import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { archiveWorkerCredential, type WorkerCredentialRow } from "@/lib/graphql/worker-credential";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import {
  credentialArchiveSchema,
  type CredentialArchiveValues,
} from "@trenova/shared/types/worker-credential";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { useCredentialInvalidation } from "./use-credential-invalidation";

type CredentialArchiveDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  credential: Pick<WorkerCredentialRow, "id" | "version" | "credentialType"> | null;
};

export function CredentialArchiveDialog({
  open,
  onOpenChange,
  workerId,
  credential,
}: CredentialArchiveDialogProps) {
  const invalidate = useCredentialInvalidation(workerId);
  const form = useForm<CredentialArchiveValues>({
    resolver: zodResolver(credentialArchiveSchema) as Resolver<CredentialArchiveValues>,
    defaultValues: { reason: null },
  });
  const { control, handleSubmit, reset } = form;

  const { mutateAsync, isPending } = useApiMutation<
    WorkerCredentialRow,
    CredentialArchiveValues,
    unknown,
    CredentialArchiveValues
  >({
    form,
    resourceName: "Credential",
    mutationFn: async (values) => {
      if (!credential) throw new Error("No credential selected");
      return archiveWorkerCredential({
        id: credential.id,
        version: credential.version,
        reason: values.reason ?? undefined,
      });
    },
    onSuccess: () => {
      toast.success("Credential archived", {
        description: "It stays in the worker's history and no longer counts toward compliance.",
      });
      void invalidate();
      reset({ reason: null });
      onOpenChange(false);
    },
  });

  const name = credential?.credentialType?.name ?? "credential";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Archive {name}</DialogTitle>
          <DialogDescription>
            Archiving removes this credential from the active file. Use Renew instead when the
            worker has a newer card.
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            onSubmit={(event) => {
              event.preventDefault();
              event.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(event);
            }}
          >
            <FormGroup cols={1}>
              <FormControl cols="full">
                <TextareaField
                  control={control}
                  name="reason"
                  label="Reason"
                  placeholder="Optional — why this credential is being retired"
                  maxLength={255}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter className="mt-4">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" variant="destructive" disabled={isPending || !credential}>
                {isPending ? "Archiving..." : "Archive"}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
