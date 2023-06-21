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
  Box,
  Button,
  createStyles,
  Divider,
  Drawer,
  Group,
  rem,
  SimpleGrid,
  Skeleton,
  Stack,
  Text,
} from "@mantine/core";
import React from "react";
import { useForm, yupResolver } from "@mantine/form";
import { useMutation, useQuery, useQueryClient } from "react-query";
import { SelectInput } from "@/components/ui/fields/SelectInput";
import { Department, Organization } from "@/types/organization";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck } from "@fortawesome/pro-solid-svg-icons";
import * as Yup from "yup";
import { SwitchInput } from "@/components/ui/fields/SwitchInput";
import { JobTitle } from "@/types/apps/accounts";
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

interface CreateUserFormValues {
  organization: string;
  username: string;
  department?: string;
  email: string;
  profile: {
    job_title: string;
    organization: string;
    first_name: string;
    last_name: string;
    address_line_1: string;
    address_line_2?: string;
    city: string;
    state: string;
    zip_code: string;
    phone_number?: string;
  };
}

const useStyles = createStyles((theme) => {
  const BREAKPOINT = theme.fn.smallerThan("sm");

  return {
    fields: {
      marginTop: rem(10),
    },
    control: {
      [BREAKPOINT]: {
        flex: 1,
      },
    },
    text: {
      color: theme.colorScheme === "dark" ? "white" : "black",
    },
    invalid: {
      backgroundColor:
        theme.colorScheme === "dark"
          ? theme.fn.rgba(theme.colors.red[8], 0.15)
          : theme.colors.red[0],
    },
    invalidIcon: {
      color: theme.colors.red[theme.colorScheme === "dark" ? 7 : 6],
    },
    div: {
      marginBottom: rem(10),
    },
  };
});

export const CreateUserDrawer: React.FC = () => {
  const { classes } = useStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const [checked, setChecked] = React.useState(false);
  const [showCreateUserDrawer, setShowCreateUserDrawer] =
    userTableStore.use("drawerOpen");
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: CreateUserFormValues) => axios.post(`/users/`, values),
    {
      onSuccess: () => {
        queryClient.invalidateQueries("user-table-data").then(() => {
          notifications.show({
            title: "Success",
            message: "User created successfully",
            color: "green",
            withCloseButton: true,
            icon: <FontAwesomeIcon icon={faCheck} />,
          });
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

  const form = useForm<CreateUserFormValues>({
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

  const submitForm = (values: CreateUserFormValues) => {
    form.setFieldValue("profile.organization", form.values.organization);
    setLoading(true);
    mutation.mutate(values);
  };

  const { data: organizationsData, isLoading: isOrganizationsLoading } =
    useQuery({
      queryKey: ["organizations"],
      queryFn: () => getOrganizations(),
      enabled: showCreateUserDrawer,
      initialData: () => {
        return queryClient.getQueryData("organizations");
      },
      staleTime: Infinity,
    });

  const { data: departmentsData, isLoading: isDepartmentLoading } = useQuery({
    queryKey: ["departments"],
    queryFn: () => getDepartments(),
    enabled: showCreateUserDrawer,
    initialData: () => {
      return queryClient.getQueryData("departments");
    },
    staleTime: Infinity,
  });

  const { data: jobTitleData, isLoading: isJobTitleLoading } = useQuery({
    queryKey: ["job_titles"],
    queryFn: () => getJobTitles(),
    enabled: showCreateUserDrawer,
    initialData: () => {
      return queryClient.getQueryData("job_titles");
    },
    staleTime: Infinity,
  });

  const isLoading =
    isDepartmentLoading || isJobTitleLoading || isOrganizationsLoading;

  const selectOrganizationData =
    organizationsData?.map((organization: Organization) => ({
      value: organization.id,
      label: organization.name,
    })) || [];

  const selectDepartmentData =
    departmentsData?.map((department: Department) => ({
      value: department.id,
      label: department.name,
    })) || [];

  const selectJobTitleData =
    jobTitleData?.map((job_title: JobTitle) => ({
      value: job_title.id,
      label: job_title.name,
    })) || [];

  if (!showCreateUserDrawer) return null;

  return (
    <>
      <Drawer
        opened={showCreateUserDrawer}
        onClose={() => setShowCreateUserDrawer(false)}
        title="Create User"
        size="lg"
      >
        {isLoading ? (
          <Stack>
            <Skeleton height={300} />
            <Skeleton height={500} />
          </Stack>
        ) : (
          <>
            <Divider variant="dashed" />
            <form onSubmit={form.onSubmit((values) => submitForm(values))}>
              <Box className={classes.div}>
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
                      withAsterisk
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
                      searchable={true}
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
        )}
      </Drawer>
    </>
  );
};