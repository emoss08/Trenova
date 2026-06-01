import { EmailProfileAutocompleteField } from "@/components/autocomplete-fields";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { EmailProfileAssignment } from "@/types/email";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { RouteIcon, SaveIcon } from "lucide-react";
import { useEffect } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
import { emailPurposes } from "./email-profile-constants";

const purposeAssignmentsSchema = z.object({
  General: z.string(),
  Billing: z.string(),
  Reporting: z.string(),
  Operations: z.string(),
  Authentication: z.string(),
  Notifications: z.string(),
});

type PurposeAssignmentsFormValues = z.infer<typeof purposeAssignmentsSchema>;

const emptyAssignments = Object.fromEntries(
  emailPurposes.map((purpose) => [purpose, ""]),
) as PurposeAssignmentsFormValues;

function toFormValues(assignments: EmailProfileAssignment[]): PurposeAssignmentsFormValues {
  const values = { ...emptyAssignments };
  for (const assignment of assignments) {
    if (emailPurposes.includes(assignment.purpose) && assignment.profileId) {
      values[assignment.purpose] = assignment.profileId;
    }
  }
  return values;
}

function toPayload(values: PurposeAssignmentsFormValues): EmailProfileAssignment[] {
  return emailPurposes
    .map((purpose) => ({
      purpose,
      profileId: values[purpose],
    }))
    .filter((assignment) => assignment.profileId);
}

export function PurposeAssignmentsPanel() {
  const queryClient = useQueryClient();
  const assignmentsQuery = useQuery(queries.email.assignments());
  const form = useForm<PurposeAssignmentsFormValues>({
    resolver: zodResolver(purposeAssignmentsSchema),
    defaultValues: emptyAssignments,
  });

  useEffect(() => {
    form.reset(toFormValues(assignmentsQuery.data ?? []));
  }, [assignmentsQuery.data, form]);

  const saveAssignments = useMutation({
    mutationFn: (values: PurposeAssignmentsFormValues) =>
      apiService.emailService.updateAssignments(toPayload(values)),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: queries.email.assignments().queryKey });
      toast.success("Purpose assignments updated");
    },
    onError: (error) => {
      toast.error("Failed to update purpose assignments", {
        description: error instanceof Error ? error.message : "An unexpected error occurred",
      });
    },
  });

  return (
    <section className="rounded-md border border-border bg-background">
      <div className="flex flex-col gap-1 border-b border-border px-4 py-3">
        <div className="flex items-center gap-2 text-sm font-medium">
          <RouteIcon className="size-4 text-muted-foreground" />
          Purpose Assignments
        </div>
        <p className="text-xs text-muted-foreground">
          Assign each email purpose to an active sender profile. Clearing a purpose removes its
          assignment on save.
        </p>
      </div>
      <div className="p-4">
        <FormProvider {...form}>
          <Form onSubmit={form.handleSubmit((values) => saveAssignments.mutate(values))}>
            <FormGroup cols={3}>
              {emailPurposes.map((purpose) => (
                <FormControl key={purpose}>
                  <EmailProfileAutocompleteField<PurposeAssignmentsFormValues>
                    control={form.control}
                    name={purpose}
                    label={purpose}
                    placeholder="Unassigned"
                    clearable
                    noResultsMessage="No active email profiles found."
                    initialLimit={20}
                  />
                </FormControl>
              ))}
            </FormGroup>
            <div className="mt-4 flex justify-end">
              <Button
                type="submit"
                disabled={!form.formState.isDirty}
                isLoading={saveAssignments.isPending}
                loadingText="Saving..."
              >
                <SaveIcon />
                Save Assignments
              </Button>
            </div>
          </Form>
        </FormProvider>
      </div>
    </section>
  );
}
