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

import React from "react";
import { Box, Button, Group, Modal } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useForm, yupResolver } from "@mantine/form";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { ValidatedTextArea } from "@/components/common/fields/TextArea";
import { useCommentTypeStore as store } from "@/stores/DispatchStore";
import { commentTypeSchema } from "@/lib/schemas/DispatchSchema";
import {
  CommentType,
  CommentTypeFormValues as FormValues,
} from "@/types/dispatch";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableStoreProps } from "@/types/tables";

function EditCommentTypeModalForm({
  commentType,
}: {
  commentType: CommentType;
}) {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);

  const form = useForm<FormValues>({
    validate: yupResolver(commentTypeSchema),
    initialValues: {
      name: commentType.name,
      description: commentType.description,
    },
  });

  const mutation = useCustomMutation<
    FormValues,
    Omit<TableStoreProps<CommentType>, "drawerOpen">
  >(
    form,
    store,
    notifications,
    {
      method: "PUT",
      path: `/comment_types/${commentType.id}/`,
      successMessage: "Comment Type updated successfully.",
      queryKeysToInvalidate: ["comment-types-table-data"],
      additionalInvalidateQueries: ["commentTypes"],
      closeModal: true,
      errorMessage: "Failed to update comment type.",
    },
    () => setLoading(false),
  );

  const submitForm = (values: FormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => submitForm(values))}>
      <Box className={classes.div}>
        <ValidatedTextInput<FormValues>
          form={form}
          name="name"
          label="Name"
          placeholder="Name"
          description="Unique Name of the Comment Type"
          withAsterisk
        />
        <ValidatedTextArea<FormValues>
          form={form}
          name="description"
          label="Description"
          description="Description of the Comment Type"
          placeholder="Description"
          withAsterisk
        />
        <Group position="right" mt="md">
          <Button type="submit" className={classes.control} loading={loading}>
            Submit
          </Button>
        </Group>
      </Box>
    </form>
  );
}

export function EditCommentTypeModal() {
  const [showEditModal, setShowEditModal] = store.use("editModalOpen");
  const [commentType] = store.use("selectedRecord");

  return (
    <Modal.Root opened={showEditModal} onClose={() => setShowEditModal(false)}>
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Edit Comment Type</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          {commentType && (
            <EditCommentTypeModalForm commentType={commentType} />
          )}
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
