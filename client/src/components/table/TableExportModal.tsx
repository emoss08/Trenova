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
import {
  Box,
  Button,
  Divider,
  Modal,
  MultiSelect,
  Select,
  Skeleton,
  Stack,
  Text,
} from "@mantine/core";
import { useForm, yupResolver } from "@mantine/form";
import { exportModelTypes, TExportModelFormValue } from "@/types/forms";
import { ExportModelSchema } from "@/utils/schema";
import { notifications } from "@mantine/notifications";
import { faCheck } from "@fortawesome/pro-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { useQuery } from "react-query";
import { getColumns } from "@/requests/ReportRequestFactory";
import axios from "@/lib/AxiosConfig";

interface Props {
  store: any;
  modelName: string;
  name: string;
}

export const TableExportModal: React.FC<Props> = ({
  store,
  modelName,
  name,
}) => {
  const [loading, setLoading] = React.useState<boolean>(false);
  const [showExportModal, setShowExportModal] = store.use("exportModalOpen");

  const { data: columnsData, isLoading: isColumnsLoading } = useQuery({
    queryKey: [`${modelName}-Columns`],
    queryFn: () => getColumns(modelName as string),
    enabled: showExportModal,
    staleTime: Infinity,
  });

  const form = useForm<TExportModelFormValue>({
    validate: yupResolver(ExportModelSchema),
    initialValues: {
      file_format: "csv",
      columns: [],
    },
  });

  const columns = columnsData?.map((column: any) => ({
    label: column,
    value: column,
  }));

  const submitForm = async (values: TExportModelFormValue) => {
    setLoading(true);

    try {
      const response = await axios.post("generate_report/", {
        model_name: modelName as string,
        file_format: values.file_format,
        columns: values.columns,
      });

      if (response.status === 202) {
        setShowExportModal(false);
        notifications.show({
          title: "Success",
          message: response.data.results,
          color: "green",
          withCloseButton: true,
          icon: <FontAwesomeIcon icon={faCheck} />,
        });
      }
    } catch (error: any) {
      notifications.show({
        title: "Error",
        message: error.response.data.error,
        color: "red",
        withCloseButton: true,
      });
    } finally {
      setLoading(false);
    }
  };

  if (!setShowExportModal) return null;

  return (
    <>
      <Modal.Root
        opened={showExportModal}
        onClose={() => setShowExportModal(false)}
        centered
        styles={{
          inner: {
            section: {
              overflowY: "visible",
            },
          },
        }}
      >
        <Modal.Overlay />
        <Modal.Content>
          <Modal.Header>
            <Modal.Title>Export {name}s</Modal.Title>
            <Modal.CloseButton />
          </Modal.Header>
          <Modal.Body>
            {isColumnsLoading ? (
              <Stack>
                <Skeleton height={150} />
                <Skeleton height={150} />
              </Stack>
            ) : (
              <>
                <form onSubmit={form.onSubmit((values) => submitForm(values))}>
                  <Box mb={10}>
                    <MultiSelect
                      data={columns}
                      label="Select Columns"
                      placeholder="Select columns"
                      clearable
                      searchable
                      nothingFound="No columns available"
                      maxDropdownHeight={160}
                      limit={20}
                      dropdownComponent="div"
                      withAsterisk
                      {...form.getInputProps("columns")}
                    />
                    <Text size="xs" color="dimmed" mt={5}>
                      Fields with underscores are related fields. For example,
                      'organization__name' is the 'name' field of the
                      organization of the record.
                    </Text>
                  </Box>
                  <Box>
                    <Select
                      label="Select Export Format"
                      placeholder="Select a format"
                      data={exportModelTypes}
                      dropdownPosition="bottom"
                      searchable
                      nothingFound="No options"
                      withAsterisk
                      {...form.getInputProps("file_format")}
                    />
                    <Text size="xs" color="dimmed" mt={5}>
                      Select a format to export (CSV, Excel, or PDF).
                    </Text>
                  </Box>
                  <Divider mt={10} />
                  <Box
                    mt={10}
                    style={{
                      display: "flex",
                      justifyContent: "flex-end",
                    }}
                  >
                    <Button
                      onClick={() => setShowExportModal(false)}
                      variant="light"
                    >
                      Cancel
                    </Button>
                    <Button
                      type="submit"
                      variant="primary"
                      ml={5}
                      loading={loading}
                      disabled={form.values.columns.length === 0}
                    >
                      Export
                    </Button>
                  </Box>
                </form>
              </>
            )}
          </Modal.Body>
        </Modal.Content>
      </Modal.Root>
    </>
  );
};
