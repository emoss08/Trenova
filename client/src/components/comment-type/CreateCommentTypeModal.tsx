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

import React, { Suspense } from "react";
import { Box, Button, Group, Modal, Skeleton } from "@mantine/core";
import { useMutation, useQueryClient } from "react-query";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useForm, yupResolver } from "@mantine/form";
import { useCommentTypeStore as store } from "@/stores/DispatchStore";
import { useFormStyles } from "@/assets/styles/FormStyles";
import axios from "@/helpers/AxiosConfig";
import { APIError } from "@/types/server";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { ValidatedTextArea } from "@/components/common/fields/TextArea";
import { CommentTypeFormValues } from "@/types/dispatch";
import { commentTypeSchema } from "@/helpers/schemas/DispatchSchema";

function CreateCommentTypeModalForm() {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: CommentTypeFormValues) => axios.post("/comment_types/", values),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries({
            queryKey: ["comment-types-table-data"],
          })
          .then(() => {
            notifications.show({
              title: "Success",
              message: "Comment Type created successfully",
              color: "green",
              withCloseButton: true,
              icon: <FontAwesomeIcon icon={faCheck} />,
            });
            store.set("createModalOpen", false);
          });
      },
      onError: (error: any) => {
        const { data } = error.response;
        if (data.type === "validation_error") {
          data.errors.forEach((e: APIError) => {
            form.setFieldError(e.attr, e.detail);
            if (e.attr === "nonFieldErrors") {
              notifications.show({
                title: "Error",
                message: e.detail,
                color: "red",
                withCloseButton: true,
                icon: <FontAwesomeIcon icon={faXmark} />,
                autoClose: 10_000, // 10 seconds
              });
            } else if (
              e.attr === "All" &&
              e.detail ===
                "Comment Type with this Name and Organization already exists."
            ) {
              form.setFieldError("name", e.detail);
            }
          });
        }
      },
      onSettled: () => {
        setLoading(false);
      },
    },
  );

  const form = useForm<CommentTypeFormValues>({
    validate: yupResolver(commentTypeSchema),
    initialValues: {
      name: "",
      description: "",
    },
  });

  const submitForm = (values: CommentTypeFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => submitForm(values))}>
      <Box className={classes.div}>
        <ValidatedTextInput<CommentTypeFormValues>
          form={form}
          name="name"
          label="Name"
          placeholder="Name"
          description="Unique Name of the comment type"
          withAsterisk
        />
        <ValidatedTextArea<CommentTypeFormValues>
          form={form}
          name="description"
          label="Description"
          description="Description of the comment type."
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

export function CreateCommentTypeModal() {
  const [showCreateModal, setShowCreateModal] = store.use("createModalOpen");

  if (!showCreateModal) return null;

  return (
    <Modal.Root
      opened={showCreateModal}
      onClose={() => setShowCreateModal(false)}
    >
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Create Comment Type</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <Suspense fallback={<Skeleton height={400} />}>
            <CreateCommentTypeModalForm />
          </Suspense>
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
