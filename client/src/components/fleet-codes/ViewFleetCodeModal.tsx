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
  Select,
  SimpleGrid,
  Skeleton,
  TextInput,
} from "@mantine/core";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { FleetCode } from "@/types/dispatch";
import { useFleetCodeStore as store } from "@/stores/DispatchStore";
import { yesAndNoChoicesBoolean } from "@/lib/constants";
import { useUsers } from "@/hooks/useUsers";
import { TChoiceProps } from "@/types";
import { BooleanSelectInput } from "@/components/common/fields/BooleanSelect";

type Props = {
  fleetCode: FleetCode;
  users: ReadonlyArray<TChoiceProps>;
};

function ViewFleetCodeModalForm({ fleetCode, users }: Props) {
  const { classes } = useFormStyles();

  return (
    <Box className={classes.div}>
      <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
        <BooleanSelectInput
          name="isActive"
          label="Is Active"
          className={classes.fields}
          description="Is this Fleet Code active?"
          variant="filled"
          withAsterisk
          value={fleetCode.isActive}
          readOnly
          data={yesAndNoChoicesBoolean}
        />
        <TextInput
          name="code"
          label="Code"
          variant="filled"
          className={classes.fields}
          placeholder="Code"
          value={fleetCode.code}
          readOnly
          description="Unique Code of the Fleet Code"
          withAsterisk
        />
      </SimpleGrid>
      <TextInput
        name="description"
        variant="filled"
        label="Description"
        value={fleetCode.description}
        className={classes.fields}
        readOnly
        description="Description of the Fleet Code"
        placeholder="Description"
        withAsterisk
      />
      <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
        <TextInput
          name="revenueGoal"
          label="Revenue Goal"
          placeholder="Revenue Goal"
          variant="filled"
          className={classes.fields}
          value={fleetCode.revenueGoal}
          readOnly
          withAsterisk
          description="Revenue goal for Fleet Code"
        />
        <TextInput
          name="deadheadGoal"
          label="Deadhead Goal"
          className={classes.fields}
          placeholder="Deadhead Goal"
          variant="filled"
          value={fleetCode.deadheadGoal}
          readOnly
          withAsterisk
          description="Deadhead goal for Fleet Code"
        />
        <TextInput
          name="mileageGoal"
          className={classes.fields}
          label="Mileage Goal"
          placeholder="Mileage Goal"
          variant="filled"
          value={fleetCode.mileageGoal}
          readOnly
          withAsterisk
          description="Mileage goal for Fleet Code"
        />
        <Select
          name="manager"
          className={classes.fields}
          label="Manager"
          value={fleetCode.manager}
          variant="filled"
          readOnly
          placeholder="Manager"
          description="Manger of the Fleet Code"
          data={users}
        />
      </SimpleGrid>
      <Group position="right" mt="md">
        <Button
          color="white"
          type="submit"
          className={classes.control}
          onClick={() => {
            store.set("selectedRecord", fleetCode);
            store.set("viewModalOpen", false);
            store.set("editModalOpen", true);
          }}
        >
          Edit Fleet Code
        </Button>
      </Group>
    </Box>
  );
}

export function ViewFleetCodeModal() {
  const [showViewModal, setShowViewModal] = store.use("viewModalOpen");
  const [fleetCode] = store.use("selectedRecord");
  const { selectUsersData } = useUsers(showViewModal);

  if (!showViewModal) return null;

  return (
    <Modal.Root opened={showViewModal} onClose={() => setShowViewModal(false)}>
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>View Fleet code</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <Suspense fallback={<Skeleton height={400} />}>
            {fleetCode && (
              <ViewFleetCodeModalForm
                fleetCode={fleetCode}
                users={selectUsersData}
              />
            )}
          </Suspense>
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
