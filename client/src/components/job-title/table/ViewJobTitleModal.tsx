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

import {
  Box,
  Button,
  Group,
  Modal,
  Select,
  SimpleGrid,
  Textarea,
  TextInput,
} from "@mantine/core";
import React from "react";
import { jobTitleTableStore as store } from "@/stores/UserTableStore";
import { JobTitle } from "@/types/accounts";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { statusChoices } from "@/lib/constants";
import { jobFunctionChoices } from "@/lib/choices";

type ViewJobTitleModalFormProps = {
  jobTitle: JobTitle;
};

export function ViewJobTitleModalForm({
  jobTitle,
}: ViewJobTitleModalFormProps): React.ReactElement {
  const { classes } = useFormStyles();

  return (
    <Box className={classes.div}>
      <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
        <Select
          data={statusChoices}
          className={classes.fields}
          readOnly
          value={jobTitle.status}
          label="Status"
          variant="filled"
        />
        <TextInput
          value={jobTitle.name}
          readOnly
          className={classes.fields}
          label="Name"
          variant="filled"
        />
      </SimpleGrid>
      <Textarea
        value={jobTitle.description || ""}
        className={classes.fields}
        label="Description"
        readOnly
        variant="filled"
      />
      <Select
        data={jobFunctionChoices}
        value={jobTitle.jobFunction}
        readOnly
        label="Job Function"
        className={classes.fields}
        variant="filled"
      />
      <Group position="right" mt="md">
        <Button
          color="white"
          type="submit"
          onClick={() => {
            store.set("viewModalOpen", false);
            store.set("editModalOpen", true);
          }}
          className={classes.control}
        >
          Edit Job Title
        </Button>
      </Group>
    </Box>
  );
}

export function ViewJobTitleModal(): React.ReactElement {
  const [showViewModal, setShowViewModal] = store.use("viewModalOpen");
  const [jobTitle] = store.use("selectedRecord");

  return (
    <Modal.Root opened={showViewModal} onClose={() => setShowViewModal(false)}>
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>View Job Title</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          {jobTitle && <ViewJobTitleModalForm jobTitle={jobTitle} />}
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
