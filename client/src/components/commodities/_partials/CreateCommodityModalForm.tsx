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
import { Box, Button, Group, SimpleGrid } from "@mantine/core";
import { ValidatedTextInput } from "@/components/ui/fields/TextInput";
import { ValidatedTextArea } from "@/components/ui/fields/TextArea";
import { useMutation, useQueryClient } from "react-query";
import axios from "@/lib/AxiosConfig";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { APIError } from "@/types/server";
import { useForm, yupResolver } from "@mantine/form";
import { chargeTypeTableStore } from "@/stores/BillingStores";
import { useFormStyles } from "@/styles/FormStyles";
import { CommodityFormValues } from "@/types/apps/commodities";
import { ValidatedNumberInput } from "@/components/ui/fields/NumberInput";
import { SelectInput } from "@/components/ui/fields/SelectInput";
import { unitOfMeasureChoices } from "@/utils/apps/commodities";
import { TChoiceProps } from "@/types";
import { commoditySchema } from "@/utils/apps/commodities/schema";
import { yesAndNoChoices } from "@/lib/utils";

type Props = {
  selectHazmatData: TChoiceProps[];
};

export function CreateCommodityModalForm({ selectHazmatData }: Props) {
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
            chargeTypeTableStore.set("createModalOpen", false);
          });
      },
      onError: (error: any) => {
        const { data } = error.response;
        if (data.type === "validation_error") {
          data.errors.forEach((error: APIError) => {
            form.setFieldError(error.attr, error.detail);
            if (error.attr === "non_field_errors") {
              notifications.show({
                title: "Error",
                message: error.detail,
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
    }
  );

  const form = useForm<CommodityFormValues>({
    validate: yupResolver(commoditySchema),
    initialValues: {
      name: "",
      description: null,
      min_temp: 0,
      max_temp: 0,
      set_point_temp: 0,
      unit_of_measure: "BAG",
      hazmat: null,
      is_hazmat: "N",
    },
  });

  const submitForm = (values: CommodityFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <>
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
              <ValidatedNumberInput
                form={form}
                className={classes.fields}
                name="min_temp"
                label="Min Temp"
                placeholder="Min Temp"
                variant="filled"
              />
              <ValidatedNumberInput
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
    </>
  );
}
