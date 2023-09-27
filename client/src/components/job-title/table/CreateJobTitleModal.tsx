/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import { Box, Button, Group, Modal, SimpleGrid } from "@mantine/core";
import React from "react";
import { notifications } from "@mantine/notifications";
import { useForm, yupResolver } from "@mantine/form";
import { jobTitleTableStore as store } from "@/stores/UserTableStore";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { JobTitle, JobTitleFormValues } from "@/types/accounts";
import { jobTitleSchema } from "@/helpers/schemas/AccountsSchema";
import { SelectInput } from "@/components/common/fields/SelectInput";
import { statusChoices } from "@/helpers/constants";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { ValidatedTextArea } from "@/components/common/fields/TextArea";
import { jobFunctionChoices } from "@/helpers/choices";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableStoreProps } from "@/types/tables";

function CreateJobTitleModalForm(): React.ReactElement {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);

  const form = useForm<JobTitleFormValues>({
    validate: yupResolver(jobTitleSchema),
    initialValues: {
      status: "A",
      name: "",
      description: "",
      jobFunction: "",
    },
  });

  const mutation = useCustomMutation<
    JobTitleFormValues,
    Omit<TableStoreProps<JobTitle>, "drawerOpen">
  >(
    form,
    store,
    notifications,
    {
      method: "POST",
      path: "/job_titles/",
      successMessage: "Job Title created successfully.",
      queryKeysToInvalidate: ["job-title-table-data"],
      closeModal: true,
      errorMessage: "Failed to create job title.",
    },
    () => setLoading(false),
  );

  const submitForm = (values: JobTitleFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => submitForm(values))}>
      <Box className={classes.div}>
        <Box>
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <SelectInput<JobTitleFormValues>
              form={form}
              data={statusChoices}
              className={classes.fields}
              name="status"
              label="Status"
              placeholder="Status"
              variant="filled"
              withAsterisk
            />
            <ValidatedTextInput<JobTitleFormValues>
              form={form}
              className={classes.fields}
              name="name"
              label="Name"
              placeholder="Name"
              variant="filled"
              withAsterisk
            />
          </SimpleGrid>
          <ValidatedTextArea<JobTitleFormValues>
            form={form}
            className={classes.fields}
            name="description"
            label="Description"
            placeholder="Description"
            variant="filled"
          />
          <SelectInput<JobTitleFormValues>
            form={form}
            data={jobFunctionChoices}
            className={classes.fields}
            name="jobFunction"
            label="Job Function"
            placeholder="Job Function"
            variant="filled"
            clearable
            withAsterisk
          />
          <Group position="right" mt="md">
            <Button
              color="white"
              type="submit"
              className={classes.control}
              loading={loading}
            >
              Submit
            </Button>
          </Group>
        </Box>
      </Box>
    </form>
  );
}

export function CreateJobTitleModal(): React.ReactElement {
  const [showCreateModal, setShowCreateModal] = store.use("createModalOpen");

  return (
    <Modal.Root
      opened={showCreateModal}
      onClose={() => setShowCreateModal(false)}
      styles={{
        inner: {
          section: {
            overflowY: "visible",
          },
        },
      }}
    >
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Create Job Title</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <CreateJobTitleModalForm />
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
