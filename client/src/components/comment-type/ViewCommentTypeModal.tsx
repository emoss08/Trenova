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
import {
  Box,
  Button,
  Group,
  Modal,
  Skeleton,
  Textarea,
  TextInput,
} from "@mantine/core";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { CommentType } from "@/types/dispatch";
import { useCommentTypeStore as store } from "@/stores/DispatchStore";

function ViewDelayCodeModalForm({ commentType }: { commentType: CommentType }) {
  const { classes } = useFormStyles();

  return (
    <Box className={classes.div}>
      <TextInput
        className={classes.fields}
        value={commentType.name}
        name="name"
        label="Name"
        placeholder="Name"
        readOnly
        variant="filled"
        withAsterisk
      />
      <Textarea
        className={classes.fields}
        name="description"
        label="Description"
        placeholder="Description"
        readOnly
        variant="filled"
        value={commentType.description || ""}
        withAsterisk
      />
      <Group position="right" mt="md">
        <Button
          color="white"
          type="submit"
          className={classes.control}
          onClick={() => {
            store.set("selectedRecord", commentType);
            store.set("viewModalOpen", false);
            store.set("editModalOpen", true);
          }}
        >
          Edit Comment Type
        </Button>
      </Group>
    </Box>
  );
}

export function ViewCommentTypeModal() {
  const [showViewModal, setShowViewModal] = store.use("viewModalOpen");
  const [commentType] = store.use("selectedRecord");

  if (!showViewModal) return null;

  return (
    <Modal.Root opened={showViewModal} onClose={() => setShowViewModal(false)}>
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>View Comment Type</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <Suspense fallback={<Skeleton height={400} />}>
            {commentType && (
              <ViewDelayCodeModalForm commentType={commentType} />
            )}
          </Suspense>
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}