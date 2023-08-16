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
import { Box, Button, Group, Modal, SimpleGrid, Skeleton } from "@mantine/core";
import { useMutation, useQuery, useQueryClient } from "react-query";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useForm, yupResolver } from "@mantine/form";
import { commodityTableStore as store } from "@/stores/CommodityStore";
import {
  CommodityFormValues,
  HazardousMaterial,
} from "@/types/apps/commodities";
import { getHazardousMaterials } from "@/requests/CommodityRequestFactory";
import { TChoiceProps } from "@/types";
import { useFormStyles } from "@/styles/FormStyles";
import axios from "@/lib/AxiosConfig";
import { APIError } from "@/types/server";
import { commoditySchema } from "@/utils/apps/commodities/schema";
import { ValidatedTextInput } from "@/components/ui/fields/TextInput";
import { ValidatedTextArea } from "@/components/ui/fields/TextArea";
import { SelectInput } from "@/components/ui/fields/SelectInput";
import { yesAndNoChoices } from "@/lib/utils";
import { unitOfMeasureChoices } from "@/utils/apps/commodities";

type CreateCommodityModalFormProps = {
  selectHazmatData: TChoiceProps[];
};

export function CreateCommodityModalForm({
  selectHazmatData,
}: CreateCommodityModalFormProps) {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: CommodityFormValues) => axios.post("/commodities/", values),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries({
            queryKey: ["commodity-table-data"],
          })
          .then(() => {
            notifications.show({
              title: "Success",
              message: "Commodity created successfully",
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
            if (e.attr === "non_field_errors") {
              notifications.show({
                title: "Error",
                message: e.detail,
                color: "red",
                withCloseButton: true,
                icon: <FontAwesomeIcon icon={faXmark} />,
                autoClose: 10_000, // 10 seconds
              });
            } else if (
              e.attr === "__all__" &&
              e.detail ===
                "Commodity with this Name and Organization already exists."
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

  const form = useForm<CommodityFormValues>({
    validate: yupResolver(commoditySchema),
    initialValues: {
      name: "",
      description: "",
      min_temp: undefined,
      max_temp: undefined,
      set_point_temp: undefined,
      unit_of_measure: undefined,
      hazmat: "",
      is_hazmat: "N",
    },
  });

  React.useEffect(() => {
    if (form.values.hazmat) {
      form.setFieldValue("is_hazmat", "Y");
    } else {
      form.setFieldValue("is_hazmat", "N");
    }
  }, [form.values.hazmat]);
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

export function CreateCommodityModal() {
  const [showCreateModal, setShowCreateModal] = store.use("createModalOpen");
  const queryClient = useQueryClient();

  const { data: hazmatData } = useQuery({
    queryKey: "hazmat-data",
    queryFn: () => getHazardousMaterials(),
    enabled: showCreateModal,
    initialData: () => queryClient.getQueryData("hazmat-data"),
    staleTime: Infinity,
  });

  const selectHazmatData =
    hazmatData?.map((hazardousMaterial: HazardousMaterial) => ({
      value: hazardousMaterial.id,
      label: hazardousMaterial.name,
    })) || [];

  if (!showCreateModal) return null;

  return (
    <Modal.Root
      opened={showCreateModal}
      onClose={() => setShowCreateModal(false)}
    >
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Create Commodity</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <Suspense fallback={<Skeleton height={400} />}>
            <CreateCommodityModalForm selectHazmatData={selectHazmatData} />
          </Suspense>
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
