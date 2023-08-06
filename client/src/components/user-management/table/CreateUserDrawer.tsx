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

import {
  Badge,
  Box,
  Button,
  Divider,
  Drawer,
  Group,
  ScrollArea,
  SimpleGrid,
  Skeleton,
  Stack,
  Tabs,
  Text,
  TransferList,
  TransferListData,
} from "@mantine/core";
import React, { useState } from "react";
import { useForm, yupResolver } from "@mantine/form";
import { useMutation, useQuery, useQueryClient } from "react-query";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck } from "@fortawesome/pro-solid-svg-icons";
import * as Yup from "yup";
import { faUser, faUserShield } from "@fortawesome/pro-duotone-svg-icons";
import { useContextMenu } from "mantine-contextmenu";
import { IconCopy, IconNote } from "@tabler/icons-react";
import { useClipboard } from "@mantine/hooks";
import { SelectInput } from "@/components/ui/fields/SelectInput";
import { SwitchInput } from "@/components/ui/fields/SwitchInput";
import { JobTitle, UserFormValues } from "@/types/apps/accounts";
import axios from "@/lib/AxiosConfig";
import {
  getDepartments,
  getJobTitles,
  getOrganizations,
} from "@/requests/OrganizationRequestFactory";
import { ValidatedTextInput } from "@/components/ui/fields/TextInput";
import { StateSelect } from "@/components/ui/fields/StateSelect";
import { CityAutoCompleteField } from "@/components/ui/fields/CityAutoCompleteField";
import { userTableStore } from "@/stores/UserTableStore";
import { Organization , Department } from "@/types/apps/organization";
import { useFormStyles } from "@/styles/FormStyles";

const initialValues: TransferListData = [
  [
    { value: "react", label: "React" },
    { value: "ng", label: "Angular" },
    { value: "next", label: "Next.js" },
    { value: "blitz", label: "Blitz.js" },
    { value: "gatsby", label: "Gatsby.js" },
    { value: "vue", label: "Vue" },
    { value: "jq", label: "jQuery" },
  ],
  [],
];

export const CreateUserDrawer: React.FC = () => {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const [checked, setChecked] = React.useState(false);
  const [showCreateUserDrawer, setShowCreateUserDrawer] =
    userTableStore.use("createModalOpen");
  const [errorCount, setErrorCount] = userTableStore.use("errorCount");
  const queryClient = useQueryClient();
  const showContextMenu = useContextMenu();
  const clipboard = useClipboard({ timeout: 500 });
  const [groupListData, setGroupListData] =
    useState<TransferListData>(initialValues);

  const mutation = useMutation(
    (values: UserFormValues) => axios.post("/users/", values),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries(["users-table-data", "users"])
          .then(() => {
            notifications.show({
              title: "Success",
              message: "User created successfully",
              color: "green",
              withCloseButton: true,
              icon: <FontAwesomeIcon icon={faCheck} />,
            });
            setErrorCount(0);
          });
      },
      onError: (error: any) => {
        const { data } = error.response;
        if (data.type === "validation_error") {
          data.errors.forEach((error: any) => {
            form.setFieldError(error.attr, error.detail);
          });
        }
      },
      onSettled: () => {
        setLoading(false);
      },
    }
  );

  const CreateUserSchema = Yup.object().shape({
    organization: Yup.string().required("Organization is required"),
    username: Yup.string().required("Username is required"),
    email: Yup.string()
      .email("Email must be a valid email")
      .required("Email is required"),
    department: Yup.string(),
    profile: Yup.object().shape({
      first_name: Yup.string().required("First name is required"),
      last_name: Yup.string().required("Last name is required"),
      address_line_1: Yup.string().required("Address Line 1 is required"),
      address_line_2: Yup.string(),
      city: Yup.string().required("City is required"),
      state: Yup.string().required("State is required"),
      zip_code: Yup.string().required("Zip Code is required"),
      job_title: Yup.string().required("Job Title is required"),
      phone_number: Yup.string()
        .nullable()
        .test(
          "phone_number_format",
          "Phone number must be in the format (xxx) xxx-xxxx",
          (value) => {
            if (!value) {
              return true;
            } // if the string is null or undefined, skip the test
            const regex = /^\(?([0-9]{3})\)?[-. ]?([0-9]{3})[-. ]?([0-9]{4})$/;
            return regex.test(value); // apply the regex test if string exists
          }
        ),
    }),
  });

  const form = useForm<UserFormValues>({
    validate: yupResolver(CreateUserSchema),
    initialValues: {
      organization: "",
      username: "",
      email: "",
      department: "",
      profile: {
        job_title: "",
        organization: "",
        first_name: "",
        last_name: "",
        address_line_1: "",
        address_line_2: "",
        city: "",
        state: "",
        zip_code: "",
        phone_number: "",
      },
    },
  });

  const submitForm = (values: UserFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  if (form.errors && Object.keys(form.errors).length > 0) {
    setErrorCount(Object.keys(form.errors).length);
  }

  const { data: organizationsData, isLoading: isOrganizationDataLoading } =
    useQuery({
      queryKey: ["organizations"],
      queryFn: () => getOrganizations(),
      enabled: showCreateUserDrawer,
      initialData: () => queryClient.getQueryData("organizations"),
      staleTime: Infinity,
    });

  const { data: departmentsData, isLoading: isDepartmentDataLoading } =
    useQuery({
      queryKey: ["departments"],
      queryFn: () => getDepartments(),
      enabled: showCreateUserDrawer,
      initialData: () => queryClient.getQueryData("departments"),
      staleTime: Infinity,
    });

  const { data: jobTitleData, isLoading: isJobTitleDataLoading } = useQuery({
    queryKey: ["job-titles"],
    queryFn: () => getJobTitles(),
    enabled: showCreateUserDrawer,
    initialData: () => queryClient.getQueryData("job-titles"),
    staleTime: Infinity,
  });

  const isLoading =
    isDepartmentDataLoading ||
    isJobTitleDataLoading ||
    isOrganizationDataLoading;

  // Department Data mapping
  const selectOrganizationData =
    organizationsData?.map((organization: Organization) => ({
      value: organization.id,
      label: organization.name,
    })) || [];

  const organizationLabel = selectOrganizationData.find(
    (item) => item.value === form.values.organization
  )?.label;

  // Department Data mapping
  const selectDepartmentData =
    departmentsData?.map((department: Department) => ({
      value: department.id,
      label: department.name,
    })) || [];

  const departmentLabel = selectDepartmentData.find(
    (item) => item.value === form.values.department
  )?.label;

  // Job Title Data mapping
  const selectJobTitleData =
    jobTitleData?.map((job_title: JobTitle) => ({
      value: job_title.id,
      label: job_title.name,
    })) || [];

  const jobTitleLabel = selectJobTitleData.find(
    (item) => item.value === form.values.profile.job_title
  )?.label;

  const onClose = () => {
    form.reset();
    setShowCreateUserDrawer(false);
    setErrorCount(0);
  };

  if (!showCreateUserDrawer) return null;

  return (
    <Drawer
      opened={showCreateUserDrawer}
      onClose={onClose}
      title="Add New User"
      size="lg"
    >
      {isLoading ? (
        <Stack>
          <Skeleton height={300} />
          <Skeleton height={500} />
        </Stack>
      ) : (
        <Tabs defaultValue="user-info">
          <Tabs.List>
            <Tabs.Tab
              value="user-info"
              icon={<FontAwesomeIcon icon={faUser} size="sm" />}
              color={errorCount > 0 ? "red" : "black"}
              rightSection={
                errorCount > 0 && (
                  <Badge
                    w={16}
                    h={16}
                    sx={{ pointerEvents: "none" }}
                    variant="filled"
                    size="xs"
                    color="red"
                    p={0}
                  >
                    {errorCount}
                  </Badge>
                )
              }
            >
                  User Information
            </Tabs.Tab>
            <Tabs.Tab
              icon={<FontAwesomeIcon icon={faUserShield} size="sm" />}
              value="permissions"
            >
                  Permissions
            </Tabs.Tab>
          </Tabs.List>
          <form
            onSubmit={form.onSubmit((values) => {
              submitForm(values);
            })}
          >
            <Tabs.Panel value="user-info" pt="xs">
              <Box className={classes.div} mr={5}>
                {/* TODO(WOLFRED: Break form into different component) */}
                <Box mb={20}>
                  <SimpleGrid
                    cols={2}
                    breakpoints={[{ maxWidth: "sm", cols: 1 }]}
                  >
                    <SelectInput
                      form={form}
                      data={selectOrganizationData}
                      className={classes.fields}
                      name="organization"
                      label="Organization"
                      placeholder="Organization"
                      variant="filled"
                      onMouseLeave={() => {
                        form.setFieldValue(
                          "profile.organization",
                          form.values.organization
                        );
                      }}
                      withAsterisk
                      onContextMenu={
                        form.values.organization
                          ? showContextMenu([
                            {
                              key: "copy",
                              icon: <IconCopy size={16} />,
                              title: "Copy to clipboard",
                              onClick: () => {
                                clipboard.copy(organizationLabel);
                              },
                            },
                            {
                              key: "view-organization",
                              icon: <IconNote size={16} />,
                              title: `View Organization: ${organizationLabel}`,
                              onClick: () =>
                                console.log(
                                  "ID ",
                                  form.values.organization
                                ),
                            },
                          ])
                          : undefined
                      }
                    />
                    <ValidatedTextInput
                      form={form}
                      className={classes.fields}
                      name="username"
                      label="Username"
                      placeholder="Username"
                      variant="filled"
                      withAsterisk
                    />
                  </SimpleGrid>
                  <SimpleGrid
                    cols={2}
                    breakpoints={[{ maxWidth: "sm", cols: 1 }]}
                    my={5}
                  >
                    <ValidatedTextInput
                      form={form}
                      className={classes.fields}
                      name="email"
                      label="Email"
                      placeholder="Email"
                      variant="filled"
                      withAsterisk
                    />
                    <SelectInput
                      form={form}
                      data={selectDepartmentData}
                      className={classes.fields}
                      name="department"
                      label="Department"
                      placeholder="Department"
                      variant="filled"
                      onContextMenu={
                        form.values.department
                          ? showContextMenu([
                            {
                              key: "copy",
                              icon: <IconCopy size={16} />,
                              title: "Copy to clipboard",
                              onClick: () => {
                                clipboard.copy(form.values.department);
                              },
                            },
                            {
                              key: "view-department",
                              icon: <IconNote size={16} />,
                              title: `View Department: ${departmentLabel}`,
                              onClick: () =>
                                console.log(
                                  "ID ",
                                  form.values.organization
                                ),
                            },
                          ])
                          : undefined
                      }
                    />
                  </SimpleGrid>
                  <SwitchInput
                    form={form}
                    size="md"
                    onChange={(event: any) =>
                      setChecked(event.currentTarget.checked)
                    }
                    checked={checked}
                    name="is_staff"
                    label="Is User Super Admin?"
                    description="Enabling this will give the user super admin privileges."
                  />
                </Box>
                <Text fz="md">Profile Details</Text>
                <Divider m={3} variant="dashed" />
                <Box>
                  <SimpleGrid
                    cols={2}
                    breakpoints={[{ maxWidth: "sm", cols: 1 }]}
                  >
                    <ValidatedTextInput
                      form={form}
                      className={classes.fields}
                      name="profile.first_name"
                      label="First Name"
                      placeholder="First Name"
                      variant="filled"
                      withAsterisk
                    />
                    <ValidatedTextInput
                      form={form}
                      className={classes.fields}
                      name="profile.last_name"
                      label="Last Name"
                      placeholder="Last Name"
                      variant="filled"
                      withAsterisk
                    />
                  </SimpleGrid>
                  <SelectInput
                    form={form}
                    data={selectJobTitleData}
                    className={classes.fields}
                    name="profile.job_title"
                    label="Job Title"
                    placeholder="Job Title"
                    variant="filled"
                    withAsterisk
                    onContextMenu={
                      form.values.profile.job_title
                        ? showContextMenu([
                          {
                            key: "copy",
                            icon: <IconCopy size={16} />,
                            title: "Copy to clipboard",
                            onClick: () => {
                              clipboard.copy(jobTitleLabel);
                            },
                          },
                          {
                            key: "view-job-title",
                            icon: <IconNote size={16} />,
                            title: `View Job Title: ${jobTitleLabel}`,
                            onClick: () =>
                              console.log(
                                "ID ",
                                form.values.profile.job_title
                              ),
                          },
                        ])
                        : undefined
                    }
                  />
                  <SimpleGrid
                    cols={2}
                    breakpoints={[{ maxWidth: "sm", cols: 1 }]}
                  >
                    <ValidatedTextInput
                      form={form}
                      className={classes.fields}
                      name="profile.address_line_1"
                      label="Address Line 1"
                      placeholder="Address Line 1"
                      variant="filled"
                      withAsterisk
                    />
                    <ValidatedTextInput
                      form={form}
                      className={classes.fields}
                      name="profile.address_line_2"
                      label="Address Line 2"
                      placeholder="Address Line 2"
                      variant="filled"
                    />
                  </SimpleGrid>
                  <SimpleGrid
                    cols={2}
                    breakpoints={[{ maxWidth: "sm", cols: 1 }]}
                  >
                    <CityAutoCompleteField
                      form={form}
                      stateSelection={form.values.profile.state}
                      className={classes.fields}
                      name="profile.city"
                      label="City"
                      placeholder="City"
                      variant="filled"
                      withAsterisk
                    />
                    <StateSelect
                      label="State"
                      className={classes.fields}
                      placeholder="State"
                      variant="filled"
                      searchable
                      form={form}
                      name="profile.state"
                      withAsterisk
                    />
                  </SimpleGrid>
                  <ValidatedTextInput
                    form={form}
                    className={classes.fields}
                    name="profile.zip_code"
                    label="Zip Code"
                    placeholder="Zip Code"
                    variant="filled"
                    withAsterisk
                  />
                  <ValidatedTextInput
                    form={form}
                    className={classes.fields}
                    name="profile.phone_number"
                    label="Phone Number"
                    placeholder="Phone Number"
                    variant="filled"
                  />
                </Box>
              </Box>
            </Tabs.Panel>
            <Tabs.Panel value="permissions" pt="xs">
              <ScrollArea h={790} scrollbarSize={5} offsetScrollbars>
                {isLoading ? (
                  <Skeleton height={790} />
                ) : (
                  <TransferList
                    value={groupListData}
                    onChange={setGroupListData}
                    searchPlaceholder="Search..."
                    nothingFound="Nothing here"
                    listHeight={300}
                    titles={["Available groups", "Chosen groups"]}
                    breakpoint="sm"
                  />
                )}
                <Divider my={10} variant="solid" />
                <TransferList
                  value={groupListData}
                  onChange={setGroupListData}
                  searchPlaceholder="Search..."
                  nothingFound="Nothing here"
                  listHeight={300}
                  titles={[
                    "Available user permissions",
                    "Chosen user permissions",
                  ]}
                  breakpoint="sm"
                />
              </ScrollArea>
            </Tabs.Panel>
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
        </Tabs>
      )}
    </Drawer>
  );
};
