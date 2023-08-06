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
import { Modal, Skeleton, Stack } from "@mantine/core";
import React from "react";
import { jobTitleTableStore } from "@/stores/UserTableStore";
import { getJobTitleDetails } from "@/requests/OrganizationRequestFactory";
import { ViewJobTitleModalForm } from "@/components/job-title/table/_Partials/ViewJobTitleModalForm";

export const ViewJobTitleModal: React.FC = () => {
  const [showViewModal, setShowViewModal] =
    jobTitleTableStore.use("viewModalOpen");
  const [jobTitle] = jobTitleTableStore.use("selectedRecord");
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

  if (!showViewModal) return null;

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
};
