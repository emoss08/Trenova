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
import { User } from "@/types/user";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useMutation, useQuery, useQueryClient } from "react-query";
import {
  getDepartments,
  getJobTitles,
  getOrganizations,
} from "@/requests/OrganizationRequestFactory";
import { Loader2 } from "lucide-react";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Department, Organization } from "@/types/organization";
import { Switch } from "@/components/ui/switch";
import { Separator } from "@/components/ui/separator";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Button } from "@/components/ui/button";
import { useErrorStore } from "@/stores/errorStore";
import { FormikProps, useFormikContext, withFormik } from "formik";
import { cn } from "@/lib/utils";
import axios from "@/lib/axiosConfig";
import { toast } from "react-toastify";

interface EditUserDialogProps {
  user: User;
  isOpen: boolean;
  onClose: () => void;
}

export const EditUserDialog: React.FC<EditUserDialogProps> = ({
  user,
  isOpen,
  onClose,
}) => {
  const [buttonStatus, setButtonStatus] = React.useState<
    "idle" | "processing" | "error" | "success"
  >("idle");
  const { errorMessages, setErrorMessages } = useErrorStore();
  const formikProps = useFormikContext();
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: EditUserFormValues) => axios.put(`/users/${values.id}/`, values),
    {
      onSuccess: () => {
        queryClient.invalidateQueries("users");
        onClose();
        setButtonStatus("success");
        toast("🚀 Successfully updated user!", {
          closeOnClick: true,
          autoClose: 1500,
        });
      },
      onError: (error: any) => {
        setErrorMessages(error.response.data);
        console.log(error);
      },
      onSettled: () => {
        setButtonStatus("idle");
      },
    }
  );

  const { data: organizationData, isLoading: isOrganizationsLoading } =
    useQuery(["organization"], () => getOrganizations(), {
      enabled: isOpen,
    });

  const { data: departmentData, isLoading: isDepartmentsLoading } = useQuery(
    ["department"],
    () => getDepartments(),
    {
      enabled: isOpen,
    }
  );

  const { data: jobTitleData, isLoading: isJobTitlesLoading } = useQuery(
    ["job_title"],
    () => getJobTitles(),
    {
      enabled: isOpen,
    }
  );

  const isLoading =
    isOrganizationsLoading || isDepartmentsLoading || isJobTitlesLoading;

  interface EditUserFormValues {
    id: string;
    username: string;
    organization: string;
    email: string;
    department?: string;
    is_superuser: boolean;
    is_staff: boolean;
    is_active: boolean;
    profile?: {
      id: string;
      first_name: string;
      last_name: string;
      user: string;
      job_title: string;
      address_line_1: string;
      address_line_2?: string;
      city: string;
      state: string;
      zip_code: string;
      phone_number?: string;
      profile_picture: string;
      is_phone_verified: boolean;
    };
  }

  const InnerEditUserForm = (props: FormikProps<EditUserFormValues>) => {
    const {
      touched,
      errors,
      handleSubmit,
      handleBlur,
      handleChange,
      isSubmitting,
      values,
      setFieldValue,
    } = props;

    const renderErrorMessages = (): JSX.Element | null => {
      if (errorMessages.length > 0) {
        return (
          <div className="flex flex-col space-y-2 text-center">
            {errorMessages.map((message: any, index: any) => {
              return (
                <p key={index} className={cn("text-sm text-rose-700")}>
                  {message}
                </p>
              );
            })}
          </div>
        );
      }
      return null;
    };

    return (
      <>
        <form onSubmit={handleSubmit}>
          {isLoading ? (
            <>
              <div className="mt-2 grid place-items-center">
                <Loader2 className="mr-2 h-10 w-10 animate-spin" />
              </div>
            </>
          ) : (
            <>
              <ScrollArea className="overflow-y-auto max-h-[70vh] pr-4">
                {renderErrorMessages()}
                <div className="mt-4">
                  <div className="flex flex-col sm:flex-row">
                    <div className="flex-1">
                      <Label htmlFor="organization">Organization</Label>
                      <Select
                        value={values.organization}
                        onValueChange={(value) =>
                          setFieldValue("organization", value)
                        }
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Select an organization" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectGroup>
                            <SelectLabel>Organizations</SelectLabel>
                            {organizationData &&
                              organizationData.map(
                                (organization: Organization) => (
                                  <SelectItem value={organization.id}>
                                    {organization.name}
                                  </SelectItem>
                                )
                              )}
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                      {errors.organization && touched.organization && (
                        <p className={cn("text-sm text-rose-700")}>
                          {errors.organization}
                        </p>
                      )}
                    </div>
                    <div className="flex-1 mt-4 sm:mt-0 sm:ml-4">
                      <Label htmlFor="email">Username</Label>
                      <Input
                        id="username"
                        name="username"
                        placeholder="Username"
                        onBlur={handleBlur}
                        onChange={handleChange}
                        value={values.username}
                      />
                    </div>
                  </div>
                  <div className="flex flex-col sm:flex-row mt-2">
                    <div className="flex-1">
                      <Label htmlFor="department_id">Department</Label>
                      <Select
                        defaultValue={values.department}
                        onValueChange={(value) => {
                          setFieldValue("department", value);
                        }}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Select an department" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectGroup>
                            <SelectLabel>Departments</SelectLabel>
                            {departmentData &&
                              departmentData.map((department: Department) => (
                                <SelectItem value={department.id}>
                                  {department.name}
                                </SelectItem>
                              ))}
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="flex-1 mt-4 sm:mt-0 sm:ml-4">
                      <Label htmlFor="email">Email</Label>
                      <Input
                        id="email"
                        name="email"
                        placeholder="Email"
                        value={values.email}
                        onBlur={handleBlur}
                        onChange={handleChange}
                      />
                    </div>
                  </div>
                  <div className="flex-1 mt-2">
                    <div className="items-top flex space-x-2">
                      <div className="flex justify-between items-center">
                        <div className="space-y-1 mr-5">
                          <Label htmlFor="is_active">Is Active</Label>
                          <div className="text-sm font-semibold text-gray-500">
                            Whether this user should be treated as active.
                            Unselect this instead of deleting accounts.
                          </div>
                        </div>
                        <label className="flex items-center cursor-pointer">
                          <Switch
                            id="is_active"
                            name="is_active"
                            checked={values.is_active}
                            onBlur={handleBlur}
                            onChange={(checked) => {
                              // Create a fake event object
                              const fakeEvent = {
                                target: {
                                  name: "is_active",
                                  value: checked,
                                },
                              };
                              // Call Formik's handleChange
                              handleChange(fakeEvent);
                              console.log("Checked?", checked);
                            }}
                          />
                          <span className="ml-2 text-sm font-medium text-gray-500">
                            Yes
                          </span>
                        </label>
                      </div>
                    </div>
                  </div>
                  <div className="flex-1 mt-2">
                    <div className="items-top flex space-x-2">
                      <div className="flex justify-between items-center">
                        <div className="space-y-1 mr-5">
                          <Label htmlFor="is_superuser">Is Superuser</Label>
                          <div className="text-sm font-semibold text-gray-500">
                            Designates that this user has all permissions
                            without explicitly assigning them.
                          </div>
                        </div>
                        <label className="flex items-center cursor-pointer">
                          <Switch
                            id="is_superuser"
                            name="is_superuser"
                            defaultChecked={values.is_superuser}
                            onBlur={handleBlur}
                            onChange={handleChange}
                          />
                          <span className="ml-2 text-sm font-medium text-gray-500">
                            Yes
                          </span>
                        </label>
                      </div>
                    </div>
                  </div>
                  <div className="flex-1 mt-2">
                    <div className="items-top flex space-x-2">
                      <div className="flex justify-between items-center">
                        <div className="space-y-1 mr-5">
                          <Label htmlFor="">Is Staff</Label>
                          <div className="text-sm font-semibold text-gray-500">
                            Designates whether the user can log into this admin
                            site.
                          </div>
                        </div>
                        <label className="flex items-center cursor-pointer">
                          <Switch
                            id="is_staff"
                            name="is_staff"
                            defaultChecked={values.is_staff}
                            onBlur={handleBlur}
                            onChange={handleChange}
                          />
                          <span className="ml-2 text-sm font-medium text-gray-500">
                            Yes
                          </span>
                        </label>
                      </div>
                    </div>
                  </div>
                  <h3 className="mt-6 text-center">Profile Details</h3>
                  <Separator className="mt-2" />
                  <div className="flex flex-col sm:flex-row mt-2">
                    <div className="flex-1">
                      <Label htmlFor="first_name">First Name</Label>
                      <Input
                        id="first_name"
                        name="first_name"
                        placeholder="First Name"
                        value={values.profile?.first_name}
                        onBlur={handleBlur}
                        onChange={handleChange}
                      />
                    </div>
                    <div className="flex-1 mt-4 sm:mt-0 sm:ml-4">
                      <Label htmlFor="last_name">Last Name</Label>
                      <Input
                        id="last_name"
                        name="last_name"
                        placeholder="Last Name"
                        value={values.profile?.last_name}
                        onBlur={handleBlur}
                        onChange={handleChange}
                      />
                    </div>
                  </div>
                  <div className="flex flex-col sm:flex-row mt-2">
                    <div className="flex-1">
                      <Label htmlFor="address_line_1">Address Line 1</Label>
                      <Input
                        id="address_line_1"
                        name="address_line_1"
                        placeholder="Address Line 1"
                        value={values.profile?.address_line_1}
                        onBlur={handleBlur}
                        onChange={handleChange}
                      />
                    </div>
                    <div className="flex-1 mt-4 sm:mt-0 sm:ml-4">
                      <Label htmlFor="address_line_2">Address Line 2</Label>
                      <Input
                        id="address_line_2"
                        name="address_line_2"
                        placeholder="Address Line 2"
                        value={values.profile?.address_line_2}
                        onBlur={handleBlur}
                        onChange={handleChange}
                      />
                    </div>
                  </div>
                  <div className="flex flex-col sm:flex-row mt-2">
                    <div className="flex-1">
                      <Label htmlFor="city">City</Label>
                      <Input
                        id="city"
                        name="city"
                        placeholder="City"
                        value={values.profile?.city}
                        onBlur={handleBlur}
                        onChange={handleChange}
                      />
                    </div>
                    <div className="flex-1 mt-4 sm:mt-0 sm:ml-4">
                      <Label htmlFor="state">State</Label>
                      <Input
                        id="state"
                        name="state"
                        placeholder="state"
                        value={values.profile?.state}
                        onBlur={handleBlur}
                        onChange={handleChange}
                      />
                    </div>
                    <div className="flex-1 mt-4 sm:mt-0 sm:ml-4">
                      <Label htmlFor="zip_code">Zip/Postal Code</Label>
                      <Input
                        id="zip_code"
                        name="zip_code"
                        placeholder="zip_code"
                        value={values.profile?.zip_code}
                      />
                    </div>
                  </div>
                  <div className="flex flex-col sm:flex-row mt-2">
                    <div className="flex-1">
                      <Label htmlFor="city">Job Title</Label>
                      <Select
                        defaultValue={values.profile?.job_title}
                        onValueChange={(value) => {
                          setFieldValue("profile.job_title", value);
                        }}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Select an job title" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectGroup>
                            <SelectLabel>Job Titles</SelectLabel>
                            {jobTitleData &&
                              jobTitleData.map((jobTitle: any) => (
                                <SelectItem value={jobTitle.id}>
                                  {jobTitle.name}
                                </SelectItem>
                              ))}
                          </SelectGroup>
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="flex-1 mt-4 sm:mt-0 sm:ml-4">
                      <Label htmlFor="phone_number">Phone Number</Label>
                      <Input
                        id="phone_number"
                        name="phone_number"
                        placeholder="Phone Number"
                        value={values.profile?.phone_number}
                        onBlur={handleBlur}
                        onChange={handleChange}
                      />
                    </div>
                  </div>
                  <div className="flex-1 mt-2">
                    <div className="items-top flex space-x-2">
                      <div className="flex justify-between items-center">
                        <div className="space-y-1 mr-5">
                          <Label htmlFor="is_phone_verified">
                            Is Phone Verified
                          </Label>
                          <div className="text-sm font-semibold text-gray-500">
                            Whether this user has verified their phone number.
                          </div>
                        </div>
                        <label className="flex items-center cursor-pointer">
                          <Switch
                            id="is_phone_verified"
                            name="is_phone_verified"
                            defaultChecked={values.profile?.is_phone_verified}
                            onCheckedChange={(checked) => {
                              setFieldValue(
                                "profile.is_phone_verified",
                                checked
                              );
                            }}
                          />
                          <span className="ml-2 text-sm font-medium text-gray-500">
                            Yes
                          </span>
                        </label>
                      </div>
                    </div>
                  </div>
                </div>
              </ScrollArea>
            </>
          )}

          <DialogFooter>
            <Button
              disabled={isSubmitting || buttonStatus === "processing"}
              type="submit"
              className={cn(
                "inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:opacity-50 disabled:pointer-events-none ring-offset-background border border-input h-10 py-2 px-4 mt-2",
                buttonStatus === "error"
                  ? "bg-red-500 text-white"
                  : buttonStatus === "success"
                  ? "bg-green-500 text-white"
                  : "hover:bg-accent hover:text-accent-foreground"
              )}
            >
              {buttonStatus === "processing" ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Processing...
                </>
              ) : buttonStatus === "success" ? (
                "Success"
              ) : buttonStatus === "error" ? (
                "Invalid, try again."
              ) : (
                "Submit"
              )}
            </Button>
          </DialogFooter>
        </form>
      </>
    );
  };

  const UserEditDialogForm = withFormik<{}, EditUserFormValues>({
    mapPropsToValues: () => ({
      id: user.id,
      username: user.username,
      organization: user.organization,
      email: user.email,
      department: user.department,
      is_superuser: user.is_superuser,
      is_staff: user.is_staff,
      is_active: user.is_active,
      profile: {
        id: user.profile?.id,
        first_name: user.profile?.first_name,
        last_name: user.profile?.last_name,
        user: user.profile?.user,
        organization: user.profile?.organization,
        job_title: user.profile?.job_title,
        address_line_1: user.profile?.address_line_1,
        address_line_2: user.profile?.address_line_2,
        city: user.profile?.city,
        state: user.profile?.state,
        zip_code: user.profile?.zip_code,
        phone_number: user.profile?.phone_number,
        is_phone_verified: user.profile?.is_phone_verified,
      },
    }),
    handleSubmit: async (values, { setSubmitting }) => {
      setButtonStatus("processing");
      try {
        mutation.mutate(values);
      } catch (error: any) {
        setButtonStatus("error");
      }
      setSubmitting(false);
    },
  })(InnerEditUserForm);

  const resetFormAndButton = () => {
    setButtonStatus("idle");
    formikProps.resetForm();
  };
  return (
    <>
      <Dialog
        open={isOpen}
        onOpenChange={() => {
          onClose();
          resetFormAndButton();
        }}
      >
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>{user.username}</DialogTitle>
            <DialogDescription>
              You are currently editing the profile of{" "}
              {user.profile?.first_name ?? "-"} {user.profile?.last_name ?? "-"}{" "}
              ({user.username}).
            </DialogDescription>
          </DialogHeader>
          <UserEditDialogForm />
        </DialogContent>
      </Dialog>
    </>
  );
};
