import { UserAutocompleteField } from "@/components/ui/autocomplete-fields";
import { Button, FormSaveButton } from "@/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { usePopoutWindow } from "@/hooks/popout-window/use-popout-window";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { http } from "@/lib/http-client";
import {
  shareConfigurationSchema,
  type ShareConfigurationSchema,
  type TableConfigurationSchema,
} from "@/lib/schemas/table-configuration-schema";
import type { TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useCallback } from "react";
import { FormProvider, useForm, useFormContext } from "react-hook-form";
import { toast } from "sonner";

interface TableConfigurationShareModalProps extends TableSheetProps {
  configId: TableConfigurationSchema["id"];
}

export function TableConfigurationShareModal(
  props: TableConfigurationShareModalProps,
) {
  const { isPopout, closePopout } = usePopoutWindow();
  const { configId } = props;

  const form = useForm({
    resolver: zodResolver(shareConfigurationSchema),
    defaultValues: {
      shareWithId: "",
      shareType: "User",
      configurationId: configId,
    },
  });

  const { reset } = form;

  const { mutateAsync } = useApiMutation({
    mutationFn: async (data: ShareConfigurationSchema) => {
      return await http.post("/table-configurations/share", data);
    },
    onSuccess: () => {
      toast.success("Configuration shared");
      closePopout();

      reset();
    },
  });

  const onSubmit = useCallback(
    async (data: ShareConfigurationSchema) => {
      await mutateAsync(data);
    },
    [mutateAsync],
  );

  return (
    <Dialog {...props}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Share Configuration</DialogTitle>
          <DialogDescription className="text-sm text-muted-foreground">
            Share the configuration with another user
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form onSubmit={form.handleSubmit(onSubmit)}>
            <DialogBody>
              <TableConfigurationShareForm />
            </DialogBody>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => props.onOpenChange(false)}
              >
                Cancel
              </Button>
              <FormSaveButton
                isPopout={isPopout}
                isSubmitting={form.formState.isSubmitting}
                title="Share Configuration"
              />
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

function TableConfigurationShareForm() {
  const { control } = useFormContext<ShareConfigurationSchema>();

  return (
    <FormGroup cols={1}>
      <FormControl>
        <UserAutocompleteField
          control={control}
          name="shareWithId"
          label="Share With"
          placeholder="Share with user"
          description="The user to share the configuration with"
        />
      </FormControl>
    </FormGroup>
  );
}
