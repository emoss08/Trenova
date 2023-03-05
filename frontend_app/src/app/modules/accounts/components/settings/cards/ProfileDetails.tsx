/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * Monta is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * Monta is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with Monta.  If not, see <https://www.gnu.org/licenses/>.
 */

import React, { useEffect, useState, memo } from "react";
import { useMutation, useQuery, useQueryClient } from "react-query";
import { useAuth } from "../../../../auth";
import axios from "axios";
import { getFullUser } from "../../../../auth/core/_requests";
import { ProfileDetailsContentLoader } from "../ProfileDetailsContentLoader";
import { createGlobalStore } from "../../../../../../_monta/helpers/zustand";
import clsx from "clsx";
import { useThemeMode } from "../../../../../../_monta/partials";
import { MTToastNotify } from "../../../../../../_monta/helpers/components/MTToastNotify";
import { Formik, Form, Field, ErrorMessage } from "formik";
import * as Yup from "yup";


export interface IProfileDetails {
  last_login: string;
  is_superuser: boolean;
  id: string;
  department?: string;
  username: string;
  email: string;
  is_staff: boolean;
  date_joined: string;
  groups: string[];
  user_permissions: string[];
  profile: {
    id: string
    user: string
    first_name: string
    last_name: string
    profile_picture?: string
    address_line_1: string
    address_line_2: string
    city: string
    state: string
    zip_code: string
    phone_number: string
    is_phone_verified: boolean
  };
}

const store = createGlobalStore<IProfileDetails>({
  last_login: "",
  is_superuser: false,
  id: "",
  department: "",
  username: "",
  email: "",
  is_staff: false,
  date_joined: "",
  groups: [""],
  user_permissions: [""],
  profile: {
    id: "",
    user: "",
    first_name: "",
    last_name: "",
    profile_picture: "",
    address_line_1: "",
    address_line_2: "",
    city: "",
    state: "",
    zip_code: "",
    phone_number: "",
    is_phone_verified: false
  }
});

const validationSchema = Yup.object().shape({
  profile: Yup.object().shape({
    first_name: Yup.string().required("First Name is required"),
    last_name: Yup.string().required("Last Name is required"),
    address_line_1: Yup.string().required("Address Line 1 is required"),
    city: Yup.string().required("City is required"),
    state: Yup.string().required("State is required"),
    zip_code: Yup.string()
      .required("Zip Code is required")
      .matches(/^\d{5}(?:[-\s]\d{4})?$/, "Invalid Zip Code format"),
    phone_number: Yup.string()
      .required("Phone Number is required")
      .matches(/^\D?(\d{3})\D?\D?(\d{3})\D?(\d{4})$/, "Invalid Phone Number format")
  })
});

const ProfileDetails: React.FC = () => {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [loading, setLoading] = useState(false);
  const { mode } = useThemeMode();
  const { currentUser } = useAuth();
  const queryClient = useQueryClient();

  const [user, setUser] = useState<IProfileDetails>();

  const query = useQuery(
    "user",
    () => getFullUser(currentUser?.id).then((response) => response.data),
    {
      onSuccess: (data: IProfileDetails) => {
        setUser(data);
        store.setAll(data);
      }
    }
  );

  const mutation = useMutation(
    (values: IProfileDetails) =>
      axios.put(`http://localhost:8000/api/users/${currentUser?.id}/`, values),
    {
      onSuccess: (response) => {
        const { data } = response;
        setUser(data);
        queryClient.setQueryData("user", (prevData: IProfileDetails | undefined) => {
          if (prevData) {
            return { ...prevData, ...data };
          }
          return undefined;
        });
        setTimeout(() => {
          setLoading(false);
          setIsSubmitting(false);
          window.location.reload();
        }, 2000);
      },
      onError: (error) => {
        console.log(error);
      }
    }
  );

  const handleSubmit = (values: IProfileDetails) => {
    MTToastNotify({
      message: "Profile updated successfully!",
      theme: mode === "dark" ? "dark" : "light",
      icon: "🚀",
      autoClose: 2000
    });
    mutation.mutate(values);
    setLoading(true);
    setIsSubmitting(true);
  };
  // Clear global store when this component gets destroyed
  useEffect(() => {
    return () => {
      store.reset();
    };
  }, []);

  return (
    <div className="card mb-5 mb-xl-10">
      <div
        className="card-header border-0 cursor-pointer"
        role="button"
        data-bs-toggle="collapse"
        data-bs-target="#kt_account_profile_details"
        aria-expanded="true"
        aria-controls="kt_account_profile_details"
      >
        <div className="card-title m-0">
          <h3 className="fw-bolder m-0">Profile Details</h3>
        </div>
      </div>
      <div id="kt_account_profile_details" className="collapse show">
        {query.isLoading ? (
          <div>
            <ProfileDetailsContentLoader />
          </div>
        ) : query.isError ? (
          <div>Error loading user data.</div>
        ) : query.data && user ? (
          <Formik<IProfileDetails>
            initialValues={{
              last_login: query.data.last_login,
              is_superuser: query.data.is_superuser,
              id: query.data.id,
              department: query.data.department,
              username: query.data.username,
              email: query.data.email,
              is_staff: query.data.is_staff,
              date_joined: query.data.date_joined,
              groups: query.data.groups,
              user_permissions: query.data.user_permissions,
              profile: {
                id: query.data.id,
                user: query.data.profile.user,
                first_name: query.data.profile.first_name,
                last_name: query.data.profile.last_name,
                profile_picture: query.data.profile.profile_picture,
                address_line_1: query.data.profile.address_line_1,
                address_line_2: query.data.profile.address_line_2,
                city: query.data.profile.city,
                state: query.data.profile.state,
                zip_code: query.data.profile.zip_code,
                phone_number: query.data.profile.phone_number,
                is_phone_verified: query.data.profile.is_phone_verified
              }
            }}
            onSubmit={handleSubmit}
            validationSchema={validationSchema}
          >
            {({ errors, touched }) => (
              <Form noValidate className="form">
                <div className="card-body border-top p-9">
                  <div className="row mb-6">
                    <label className="col-lg-4 col-form-label required fw-bold fs-6">
                      Full Name
                    </label>
                    <div className="col-lg-8">
                      <div className="row">
                        <div className="col-lg-6 fv-row">
                          <div className="mb-3 mb-lg-0">
                            <Field
                              className={clsx(
                                "form-control form-control-lg form-control-solid",
                                touched?.profile?.first_name && errors?.profile?.first_name
                                  ? "is-invalid" : ""
                              )}
                              placeholder="First Name"
                              name="profile.first_name"
                            />
                            <ErrorMessage name="profile.first_name">
                              {(msg) => (
                                <div className="fv-plugins-message-container invalid-feedback">
                                  <div className="fv-help-block">{msg}</div>
                                </div>
                              )}
                            </ErrorMessage>
                          </div>
                        </div>

                        <div className="col-lg-6 fv-row">
                          <div className="mb-3 mb-lg-0">
                            <Field
                              className={clsx(
                                "form-control form-control-lg form-control-solid",
                                touched?.profile?.last_name && errors?.profile?.last_name
                                  ? "is-invalid"
                                  : ""
                              )}
                              placeholder="Last name"
                              name="profile.last_name"
                            />
                            <ErrorMessage name="profile.last_name">
                              {(msg) => (
                                <div className="fv-plugins-message-container invalid-feedback">
                                  <div className="fv-help-block">{msg}</div>
                                </div>
                              )}
                            </ErrorMessage>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div className="row mb-6">
                    <label className="col-lg-4 col-form-label required fw-bold fs-6">
                      Address Line 1
                    </label>

                    <div className="col-lg-8 fv-row">
                      <div className="mb-3 mb-lg-0">
                        <Field
                          className={clsx(
                            "form-control form-control-lg form-control-solid",
                            touched?.profile?.address_line_1 && errors?.profile?.address_line_1
                              ? "is-invalid"
                              : ""
                          )}
                          placeholder="Address Line 1"
                          name="profile.address_line_1"
                        />
                        <ErrorMessage name="profile.address_line_1">
                          {(msg) => (
                            <div className="fv-plugins-message-container invalid-feedback">
                              <div className="fv-help-block">{msg}</div>
                            </div>
                          )}
                        </ErrorMessage>
                      </div>
                    </div>
                  </div>

                  <div className="row mb-6">
                    <label className="col-lg-4 col-form-label fw-bold fs-6">Address Line 2</label>

                    <div className="col-lg-8 fv-row">
                      <div className="mb-3 mb-lg-0">
                        <Field
                          className={clsx(
                            "form-control form-control-lg form-control-solid",
                            touched?.profile?.address_line_2 && errors?.profile?.address_line_2
                              ? "is-invalid"
                              : ""
                          )}
                          placeholder="Address Line 2"
                          name="profile.address_line_2"
                        />
                        <ErrorMessage name="profile.address_line_2">
                          {(msg) => (
                            <div className="fv-plugins-message-container invalid-feedback">
                              <div className="fv-help-block">{msg}</div>
                            </div>
                          )}
                        </ErrorMessage>
                      </div>
                    </div>
                  </div>

                  <div className="row mb-6">
                    <label className="col-lg-4 col-form-label required fw-bold fs-6">City</label>

                    <div className="col-lg-8 fv-row">
                      <div className="mb-3 mb-lg-0">
                        <Field
                          className={clsx(
                            "form-control form-control-lg form-control-solid",
                            touched?.profile?.city && errors?.profile?.city ? "is-invalid" : ""
                          )}
                          placeholder="City"
                          name="profile.city"
                        />
                        <ErrorMessage name="profile.city">
                          {(msg) => (
                            <div className="fv-plugins-message-container invalid-feedback">
                              <div className="fv-help-block">{msg}</div>
                            </div>
                          )}
                        </ErrorMessage>
                      </div>
                    </div>
                  </div>

                  <div className="row mb-6">
                    <label className="col-lg-4 col-form-label required fw-bold fs-6">
                      State
                    </label>

                    <div className="col-lg-8 fv-row">
                      <div className="mb-3 mb-lg-0">
                        <Field
                          className={clsx(
                            "form-control form-control-lg form-control-solid",
                            touched?.profile?.state && errors?.profile?.state ? "is-invalid" : ""
                          )}
                          placeholder="State"
                          name="profile.state"
                        />
                        <ErrorMessage name="profile.state">
                          {(msg) => (
                            <div className="fv-plugins-message-container invalid-feedback">
                              <div className="fv-help-block">{msg}</div>
                            </div>
                          )}
                        </ErrorMessage>
                      </div>
                    </div>
                  </div>

                  <div className="row mb-6">
                    <label className="col-lg-4 col-form-label required fw-bold fs-6">
                      Zip Code
                    </label>

                    <div className="col-lg-8 fv-row">
                      <div className="mb-3 mb-lg-0">
                        <Field
                          className={clsx(
                            "form-control form-control-lg form-control-solid",
                            touched?.profile?.zip_code && errors?.profile?.zip_code
                              ? "is-invalid"
                              : ""
                          )}
                          placeholder="Zip Code"
                          name="profile.zip_code"
                        />
                        <ErrorMessage name="profile.zip_code">
                          {(msg) => (
                            <div className="fv-plugins-message-container invalid-feedback">
                              <div className="fv-help-block">{msg}</div>
                            </div>
                          )}
                        </ErrorMessage>
                      </div>
                    </div>
                  </div>

                  <div className="row mb-6">
                    <label className="col-lg-4 col-form-label fw-bold fs-6">Contact Phone</label>

                    <div className="col-lg-8 fv-row">
                      <div className="mb-3 mb-lg-0">
                        <Field
                          className={clsx(
                            "form-control form-control-lg form-control-solid",
                            touched?.profile?.phone_number && errors?.profile?.phone_number
                              ? "is-invalid"
                              : ""
                          )}
                          placeholder="Phone number"
                          name="profile.phone_number"
                          type="tel"
                        />
                        <ErrorMessage name="profile.phone_number">
                          {(msg) => (
                            <div className="fv-plugins-message-container invalid-feedback">
                              <div className="fv-help-block">{msg}</div>
                            </div>
                          )}
                        </ErrorMessage>
                      </div>
                    </div>
                  </div>
                </div>

                <div className="card-footer d-flex justify-content-end py-6 px-9">
                  <button type="submit" className="btn btn-primary" disabled={isSubmitting}>
                    {!isSubmitting && "Save Changes"}
                    {isSubmitting && (
                      <span className="indicator-progress" style={{ display: "block" }}>
                    Please wait...{" "}
                        <span className="spinner-border spinner-border-sm align-middle ms-2"></span>
                  </span>
                    )}
                  </button>
                </div>
              </Form>
            )}
          </Formik>
        ) : null}
      </div>
    </div>
  );
};

export default memo(ProfileDetails)