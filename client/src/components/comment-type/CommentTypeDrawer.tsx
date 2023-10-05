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
import { Button, Drawer, Group } from "@mantine/core";
import { useForm, yupResolver } from "@mantine/form";
import { notifications } from "@mantine/notifications";
import { useFormStyles } from "@/assets/styles/FormStyles";
import {
  CommentType,
  CommentTypeFormValues as FormValues,
} from "@/types/dispatch";
import { useCommentTypeStore as store } from "@/stores/DispatchStore";
import { commentTypeSchema } from "@/lib/schemas/DispatchSchema";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableStoreProps } from "@/types/tables";
import { CommentTypeForm } from "@/components/comment-type/CreateCommentTypeModal";

function EditCommentTypeModalForm({
  commentType,
  onCancel,
}: {
  commentType: CommentType;
  onCancel: () => void;
}) {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);

  const form = useForm<FormValues>({
    validate: yupResolver(commentTypeSchema),
    initialValues: {
      status: "A",
      name: commentType.name,
      description: commentType.description,
    },
  });

  const mutation = useCustomMutation<FormValues, TableStoreProps<CommentType>>(
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
      <CommentTypeForm form={form} />
      <Group position="right" mt="md">
        <Button
          variant="subtle"
          onClick={onCancel}
          color="gray"
          type="button"
          className={classes.control}
        >
          Cancel
        </Button>
        <Button
          color="white"
          type="submit"
          className={classes.control}
          loading={loading}
        >
          Submit
        </Button>
      </Group>
    </form>
  );
}

export function CommentTypeDrawer() {
  const [drawerOpen, setDrawerOpen] = store.use("drawerOpen");
  const [commentType] = store.use("selectedRecord");
  const onCancel = () => store.set("drawerOpen", false);

  return (
    <Drawer.Root
      position="right"
      opened={drawerOpen}
      onClose={() => setDrawerOpen(false)}
    >
      <Drawer.Overlay />
      <Drawer.Content>
        <Drawer.Header>
          <Drawer.Title>
            Edit Comment Type: {commentType && commentType.name}
          </Drawer.Title>
          <Drawer.CloseButton />
        </Drawer.Header>
        <Drawer.Body>
          {commentType && (
            <EditCommentTypeModalForm
              commentType={commentType}
              onCancel={onCancel}
            />
          )}
        </Drawer.Body>
      </Drawer.Content>
    </Drawer.Root>
  );
}
