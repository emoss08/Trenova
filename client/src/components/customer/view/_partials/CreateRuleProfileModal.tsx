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
import { useMutation, useQuery, useQueryClient } from "react-query";
import { Box, Button, Group, Modal } from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useForm } from "@mantine/form";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { customerStore as store } from "@/stores/CustomerStore";
import axios from "@/helpers/AxiosConfig";
import { CustomerRuleProfileFormValues } from "@/types/customer";
import { APIError } from "@/types/server";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { ValidatedMultiSelect } from "@/components/common/fields/MultiSelect";
import { getDocumentClassifications } from "@/services/BillingRequestService";
import { DocumentClassification } from "@/types/billing";

type Props = {
  customerId: string;
};

function CreateRuleProfileModalForm({ customerId }: Props) {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const { data: documentClasses, isLoading } = useQuery({
    queryKey: ["documentClassifications"],
    queryFn: async () => getDocumentClassifications(),
    enabled: store.get("activeTab") === "profile",
    initialData: () => queryClient.getQueryData("documentClassifications"),
    staleTime: Infinity,
    retry: false,
    refetchOnWindowFocus: false,
  });

  const selectDocumentClassData =
    documentClasses?.map((item: DocumentClassification) => ({
      value: item.id,
      label: item.name,
    })) || [];

  const mutation = useMutation(
    (values: CustomerRuleProfileFormValues) =>
      axios.post("/customer_rule_profiles/", values),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries({
            queryKey: ["customerRuleProfile", customerId],
          })
          .then(() => {
            notifications.show({
              title: "Success",
              message: "Customer Rule Profile created successfully",
              color: "green",
              withCloseButton: true,
              icon: <FontAwesomeIcon icon={faCheck} />,
            });
            store.set("createRuleProfileModalOpen", false);
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

  const form = useForm<CustomerRuleProfileFormValues>({
    validateInputOnChange: true,
    initialValues: {
      name: "",
      customer: customerId,
      documentClass: [""],
    },
  });

  const submitForm: (values: CustomerRuleProfileFormValues) => void = (
    values: CustomerRuleProfileFormValues,
  ) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => submitForm(values))}>
      <Box className={classes.div}>
        <ValidatedTextInput<CustomerRuleProfileFormValues>
          form={form}
          name="name"
          label="Name"
          placeholder="Enter name"
          variant="filled"
          withAsterisk
        />
        <ValidatedMultiSelect<CustomerRuleProfileFormValues>
          form={form}
          name="document_class"
          data={selectDocumentClassData}
          placeholder="Select document class"
          label="Document Class"
          isLoading={isLoading}
          variant="filled"
          withAsterisk
          creatable
          getCreateLabel={(query) => `+ Create ${query}`}
          onCreate={(query) => {
            // This is a reference to the object that will be updated asynchronously.
            const item = {
              value: "", // or some default value
              label: "", // or some default value
            };

            axios
              .post("/document_classifications/", { name: query })
              .then(async (response) => {
                if (response.status === 201) {
                  await queryClient.invalidateQueries({
                    queryKey: ["documentClassifications"],
                  });

                  notifications.show({
                    title: "Success",
                    message: "Document Classification created successfully",
                    color: "green",
                    withCloseButton: true,
                    icon: <FontAwesomeIcon icon={faCheck} />,
                  });

                  // Update the properties of the item reference
                  item.value = response.data.id;
                  item.label = response.data.name;
                }
              })
              .catch((error) => {
                const { data } = error.response;
                if (data.type === "validation_error") {
                  data.errors.forEach((e: APIError) => {
                    notifications.show({
                      title: "Error",
                      message: e.detail,
                      color: "red",
                      withCloseButton: true,
                      icon: <FontAwesomeIcon icon={faXmark} />,
                      autoClose: 10_000, // 10 seconds
                    });
                  });
                }
              });
            return item;
          }}
        />
      </Box>
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
    </form>
  );
}

export function CreateRuleProfileModal({
  customerId,
}: Props): React.ReactElement {
  const [showCreateModal, setShowCreateModal] = store.use(
    "createRuleProfileModalOpen",
  );

  return (
    <Modal.Root
      opened={showCreateModal}
      onClose={() => setShowCreateModal(false)}
      styles={{
        // section is causing the overflow issue
        inner: {
          section: {
            overflow: "visible",
          },
        },
      }}
    >
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Create Customer Rule Profile</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <CreateRuleProfileModalForm customerId={customerId} />
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
