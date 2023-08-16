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

import { useQuery, useQueryClient } from "react-query";
import {
  Box,
  Button,
  Group,
  Modal,
  Select,
  SimpleGrid,
  Skeleton,
  Stack,
  Textarea,
  TextInput,
} from "@mantine/core";
import React from "react";
import { jobTitleTableStore as store } from "@/stores/UserTableStore";
import { getJobTitleDetails } from "@/requests/OrganizationRequestFactory";
import { JobTitle } from "@/types/apps/accounts";
import { useFormStyles } from "@/styles/FormStyles";
import { statusChoices } from "@/lib/utils";
import { jobFunctionChoices } from "@/utils/apps/accounts";

type ViewJobTitleModalFormProps = {
  jobTitle: JobTitle;
};

export function ViewJobTitleModalForm({
  jobTitle,
}: ViewJobTitleModalFormProps) {
  const { classes } = useFormStyles();

  return (
    <Box className={classes.div}>
      <Box>
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
          value={jobTitle.job_function}
          readOnly
          label="Account Type"
          className={classes.fields}
          variant="filled"
        />
        <Group position="right" mt="md">
          <Button
            color="white"
            type="submit"
            onClick={() => {
              store.set("selectedRecord", jobTitle);
              store.set("viewModalOpen", false);
              store.set("editModalOpen", true);
            }}
            className={classes.control}
          >
            Edit Job Title
          </Button>
        </Group>
      </Box>
    </Box>
  );
}

export function ViewJobTitleModal(): React.ReactElement {
  const [showViewModal, setShowViewModal] = store.use("viewModalOpen");
  const [jobTitle] = store.use("selectedRecord");
  const queryClient = useQueryClient();

  const { data: jobTitleData, isLoading: isJobTitleDataLoading } = useQuery({
    queryKey: ["jobTitle", jobTitle?.id],
    queryFn: () => {
      if (!jobTitle) {
        return Promise.resolve(null);
      }
      return getJobTitleDetails(jobTitle.id);
    },
    enabled: showViewModal,
    initialData: () => queryClient.getQueryData(["jobTitle", jobTitle?.id]),
  });

  return (
    <Modal.Root opened={showViewModal} onClose={() => setShowViewModal(false)}>
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>View Job Title</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          {isJobTitleDataLoading ? (
            <Stack>
              <Skeleton height={400} />
            </Stack>
          ) : (
            <>
              {jobTitleData && (
                <ViewJobTitleModalForm jobTitle={jobTitleData} />
              )}
            </>
          )}
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
