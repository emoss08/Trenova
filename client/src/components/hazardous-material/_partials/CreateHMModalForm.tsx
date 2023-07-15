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

import { Box, Button, Group, SimpleGrid } from "@mantine/core";
import { ValidatedTextInput } from "@/components/ui/fields/TextInput";
import { ValidatedTextArea } from "@/components/ui/fields/TextArea";
import { SelectInput } from "@/components/ui/fields/SelectInput";
import React from "react";
import { useMutation, useQueryClient } from "react-query";
import axios from "@/lib/AxiosConfig";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useForm, yupResolver } from "@mantine/form";
import { statusChoices } from "@/lib/utils";
import { useFormStyles } from "@/styles/FormStyles";
import { HazardousMaterialFormValues } from "@/types/apps/commodities";
import { hazardousMaterialTableStore } from "@/stores/CommodityStore";
import { hazardousMaterialSchema } from "@/utils/apps/commodities/schema";
import {
  hazardousClassChoices,
  packingGroupChoices,
} from "@/utils/apps/commodities";

export const CreateHMModalForm = () => {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: HazardousMaterialFormValues) =>
      axios.post("/hazardous_materials/", values),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries({
            queryKey: ["hazardous-material-table-data"],
          })
          .then(() => {
            notifications.show({
              title: "Success",
              message: "Hazardous Material created successfully",
              color: "green",
              withCloseButton: true,
              icon: <FontAwesomeIcon icon={faCheck} />,
            });
            hazardousMaterialTableStore.set("createModalOpen", false);
          });
      },
      onError: (error: any) => {
        const { data } = error.response;
        if (data.type === "validation_error") {
          data.errors.forEach((error: any) => {
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

  const form = useForm<HazardousMaterialFormValues>({
    validate: yupResolver(hazardousMaterialSchema),
    initialValues: {
      status: "A",
      name: "",
      description: "",
      hazard_class: "1.3",
      packing_group: "",
      erg_number: "",
      proper_shipping_name: "",
    },
  });

  const submitForm = (values: HazardousMaterialFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <>
      <form onSubmit={form.onSubmit((values) => submitForm(values))}>
        <Box className={classes.div}>
          <Box>
            <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
              <SelectInput
                form={form}
                data={statusChoices}
                className={classes.fields}
                name="status"
                label="Status"
                placeholder="Status"
                variant="filled"
                withAsterisk
              />
              <ValidatedTextInput
                form={form}
                className={classes.fields}
                name="name"
                label="Name"
                placeholder="Name"
                variant="filled"
                withAsterisk
              />
            </SimpleGrid>
            <ValidatedTextArea
              form={form}
              className={classes.fields}
              name="description"
              label="Description"
              placeholder="Description"
              variant="filled"
            />
            <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
              <SelectInput
                form={form}
                data={hazardousClassChoices}
                className={classes.fields}
                name="hazard_class"
                label="Hazard Class"
                placeholder="Hazard Class"
                variant="filled"
                withAsterisk
              />
              <SelectInput
                form={form}
                data={packingGroupChoices}
                className={classes.fields}
                name="packing_group"
                label="Packing Group"
                placeholder="Packing Group"
                variant="filled"
                clearable
              />
            </SimpleGrid>
            <ValidatedTextInput
              form={form}
              className={classes.fields}
              name="erg_number"
              label="ERG Number"
              placeholder="ERG Number"
              variant="filled"
            />
            <ValidatedTextArea
              form={form}
              className={classes.fields}
              name="proper_shipping_name"
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
        </Box>
      </form>
    </>
  );
};
