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
import { Box, Button, Group, Modal, SimpleGrid } from "@mantine/core";
import { useMutation, useQueryClient } from "react-query";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useForm, yupResolver } from "@mantine/form";
import axios from "@/lib/AxiosConfig";
import { useFormStyles } from "@/styles/FormStyles";
import {
  HazardousMaterial,
  HazardousMaterialFormValues,
} from "@/types/apps/commodities";
import { hazardousMaterialTableStore as store } from "@/stores/CommodityStore";
import { hazardousMaterialSchema } from "@/utils/apps/commodities/schema";
import { SelectInput } from "@/components/ui/fields/SelectInput";
import { statusChoices } from "@/lib/utils";
import { ValidatedTextInput } from "@/components/ui/fields/TextInput";
import { ValidatedTextArea } from "@/components/ui/fields/TextArea";
import {
  hazardousClassChoices,
  packingGroupChoices,
} from "@/utils/apps/commodities";
import { APIError } from "@/types/server";

type Props = {
  hazardousMaterial: HazardousMaterial;
};

function EditHMModalForm({ hazardousMaterial }: Props): React.ReactElement {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: HazardousMaterialFormValues) =>
      axios.put(`/hazardous_materials/${hazardousMaterial.id}/`, values),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries({
            queryKey: ["hazardous-material-table-data"],
          })
          .then(() =>
            queryClient.invalidateQueries({
              queryKey: ["hazardousMaterial", hazardousMaterial.id],
            }),
          )
          .finally(() => {
            notifications.show({
              title: "Success",
              message: "Hazardous Material updated successfully",
              color: "green",
              withCloseButton: true,
              icon: <FontAwesomeIcon icon={faCheck} />,
            });
            store.set("editModalOpen", false);
          });
      },
      onError: (error: any) => {
        const { data } = error.response;
        if (data.type === "validation_error") {
          data.errors.forEach((e: APIError) => {
            form.setFieldError(e.attr, e.detail);
            if (e.attr === "non_field_errors") {
              notifications.show({
                title: "Error",
                message: e.detail,
                color: "red",
                withCloseButton: true,
                icon: <FontAwesomeIcon icon={faXmark} />,
                autoClose: 10_000, // 10 seconds
              });
            }
          });
        }
      },
      onSettled: () => {
        setLoading(false);
      },
    },
  );

  const form = useForm<HazardousMaterialFormValues>({
    validate: yupResolver(hazardousMaterialSchema),
    initialValues: {
      status: hazardousMaterial.status,
      name: hazardousMaterial.name,
      description: hazardousMaterial.description,
      hazardClass: hazardousMaterial.hazardClass,
      packingGroup: hazardousMaterial.packingGroup,
      ergNumber: hazardousMaterial.ergNumber,
      properShippingName: hazardousMaterial.properShippingName,
    },
  });

  const submitForm = (values: HazardousMaterialFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => submitForm(values))}>
      <Box className={classes.div}>
        <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
          <SelectInput<HazardousMaterialFormValues>
            form={form}
            data={statusChoices}
            className={classes.fields}
            name="status"
            label="Status"
            placeholder="Status"
            variant="filled"
            withAsterisk
          />
          <ValidatedTextInput<HazardousMaterialFormValues>
            form={form}
            className={classes.fields}
            name="name"
            label="Name"
            placeholder="Name"
            variant="filled"
            withAsterisk
          />
        </SimpleGrid>
        <ValidatedTextArea<HazardousMaterialFormValues>
          form={form}
          className={classes.fields}
          name="description"
          label="Description"
          placeholder="Description"
          variant="filled"
        />
        <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
          <SelectInput<HazardousMaterialFormValues>
            form={form}
            data={hazardousClassChoices}
            className={classes.fields}
            name="hazardClass"
            label="Hazard Class"
            placeholder="Hazard Class"
            variant="filled"
            withAsterisk
          />
          <SelectInput<HazardousMaterialFormValues>
            form={form}
            data={packingGroupChoices}
            className={classes.fields}
            name="packingGroup"
            label="Packing Group"
            placeholder="Packing Group"
            variant="filled"
            clearable
          />
        </SimpleGrid>
        <ValidatedTextInput<HazardousMaterialFormValues>
          form={form}
          className={classes.fields}
          name="ergNumber"
          label="ERG Number"
          placeholder="ERG Number"
          variant="filled"
        />
        <ValidatedTextArea<HazardousMaterialFormValues>
          form={form}
          className={classes.fields}
          name="properShippingName"
          label="Proper Shipping Name"
          placeholder="Proper Shipping Name"
          variant="filled"
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
    </form>
  );
}

export function EditHMModal(): React.ReactElement {
  const [showEditModal, setShowEditModal] = store.use("editModalOpen");
  const [hazardousMaterial] = store.use("selectedRecord");

  return (
    <Modal.Root opened={showEditModal} onClose={() => setShowEditModal(false)}>
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Edit Revenue Code</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          {hazardousMaterial && (
            <EditHMModalForm hazardousMaterial={hazardousMaterial} />
          )}
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
