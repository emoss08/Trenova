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
import { useMutation, useQuery, useQueryClient } from "react-query";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useForm, yupResolver } from "@mantine/form";
import { commodityTableStore } from "@/stores/CommodityStore";
import { getHazardousMaterials } from "@/requests/CommodityRequestFactory";
import {
  Commodity,
  CommodityFormValues,
  HazardousMaterial,
} from "@/types/apps/commodities";
import { TChoiceProps } from "@/types";
import { useFormStyles } from "@/styles/FormStyles";
import axios from "@/lib/AxiosConfig";
import { commoditySchema } from "@/utils/apps/commodities/schema";
import { ValidatedTextInput } from "@/components/ui/fields/TextInput";
import { ValidatedTextArea } from "@/components/ui/fields/TextArea";
import { SelectInput } from "@/components/ui/fields/SelectInput";
import { yesAndNoChoices } from "@/lib/utils";
import { unitOfMeasureChoices } from "@/utils/apps/commodities";

type EditCommodityModalFormProps = {
  commodity: Commodity;
  selectHazmatData: TChoiceProps[];
};

function EditCommodityModalForm({
  commodity,
  selectHazmatData,
}: EditCommodityModalFormProps) {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: CommodityFormValues) =>
      axios.put(`/commodities/${commodity.id}/`, values),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries({
            queryKey: ["commodity-table-data"],
          })
          .then(() => {
            queryClient
              .invalidateQueries({
                queryKey: ["commodity", commodity.id],
              })
              .then(() => {
                notifications.show({
                  title: "Success",
                  message: "Commodity updated successfully",
                  color: "green",
                  withCloseButton: true,
                  icon: <FontAwesomeIcon icon={faCheck} />,
                });
                commodityTableStore.set("editModalOpen", false);
              });
          });
      },
      onError: (error: any) => {
        const { data } = error.response;
        if (data.type === "validation_error") {
          data.errors.forEach((e: any) => {
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

  const form = useForm<CommodityFormValues>({
    validate: yupResolver(commoditySchema),
    initialValues: {
      name: commodity.name,
      description: commodity.description,
      min_temp: commodity.min_temp,
      max_temp: commodity.max_temp,
      set_point_temp: commodity.set_point_temp,
      unit_of_measure: commodity.unit_of_measure,
      hazmat: commodity.hazmat,
      is_hazmat: commodity.is_hazmat,
    },
  });

  // Set is_hazmat value based on hazmat value
  React.useEffect(() => {
    if (form.values.hazmat) {
      form.setFieldValue("is_hazmat", "Y");
    } else {
      form.setFieldValue("is_hazmat", "N");
    }
  });

  const submitForm = (values: CommodityFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => submitForm(values))}>
      <Box className={classes.div}>
        <Box>
          <ValidatedTextInput
            form={form}
            className={classes.fields}
            name="name"
            label="Name"
            placeholder="Name"
            variant="filled"
            withAsterisk
          />
          <ValidatedTextArea
            form={form}
            className={classes.fields}
            name="description"
            label="Description"
            placeholder="Description"
            variant="filled"
          />
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <ValidatedTextInput
              form={form}
              className={classes.fields}
              name="min_temp"
              label="Min Temp"
              placeholder="Min Temp"
              variant="filled"
            />
            <ValidatedTextInput
              form={form}
              className={classes.fields}
              name="max_temp"
              label="Max Temp"
              placeholder="Max Temp"
              variant="filled"
            />
          </SimpleGrid>
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <SelectInput
              className={classes.fields}
              data={selectHazmatData || []}
              name="hazmat"
              placeholder="Hazardous Material"
              label="Hazardous Material"
              form={form}
              variant="filled"
              clearable
            />
            <SelectInput
              className={classes.fields}
              data={yesAndNoChoices}
              name="is_hazmat"
              label="Is Hazmat"
              placeholder="Is Hazmat"
              form={form}
              variant="filled"
              withAsterisk
            />
          </SimpleGrid>
          <SelectInput
            className={classes.fields}
            data={unitOfMeasureChoices}
            name="unit_of_measure"
            placeholder="Unit of Measure"
            label="Unit of Measure"
            form={form}
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
      </Box>
    </form>
  );
}

export function EditCommodityModal() {
  const [showEditModal, setShowEditModal] =
    commodityTableStore.use("editModalOpen");
  const [commodity] = commodityTableStore.use("selectedRecord");
  const queryClient = useQueryClient();

  const { data: hazmatData } = useQuery({
    queryKey: "hazmat-data",
    queryFn: () => getHazardousMaterials(),
    enabled: showEditModal,
    initialData: () => queryClient.getQueryData("hazmat-data"),
    staleTime: Infinity,
  });

  const selectHazmatData =
    hazmatData?.map((hazardousMaterial: HazardousMaterial) => ({
      value: hazardousMaterial.id,
      label: hazardousMaterial.name,
    })) || [];

  if (!showEditModal) return null;

  return (
    <Modal.Root opened={showEditModal} onClose={() => setShowEditModal(false)}>
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Edit Commodity</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          {commodity && (
            <EditCommodityModalForm
              commodity={commodity}
              selectHazmatData={selectHazmatData}
            />
          )}
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
