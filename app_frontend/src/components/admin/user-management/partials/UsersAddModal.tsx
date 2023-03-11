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

import { MTSVG } from "@/components/elements/MTSVG";
import SvgArr009 from "@/components/svgs/SvgArr009";
import { useEffect, useState } from "react";
import Button from "react-bootstrap/Button";
import { Form, Modal } from "react-bootstrap";
import SvgArr061 from "@/components/svgs/SvgArr061";
import * as Yup from "yup";
import { Field, useFormik } from "formik";
import clsx from "clsx";
import axios from "axios";
import { DangerAlert } from "@/components/partials/DangerAlert";
import SvgGen016 from "@/components/svgs/SvgGen016";
import { toast } from "react-toastify";
import { phoneRegex, stateChoices, zipCodeRegex } from "@/utils/FieldHelpers";
import Select from "react-select";
import StateSelect from "@/components/partials/fields/StateSelect";

type JobTitlesOptionType = {
  id: string;
  name: string;
};

const UsersSchema = Yup.object().shape({
  username: Yup.string()
    .min(5, "Minimum 5 characters")
    .max(30, "Maximum 30 characters or fewer. Letters, digits and @/./+/-/_ only")
    .required("Username is required"),
  email: Yup.string()
    .email("Invalid email address")
    .required("Email is required"),
  password: Yup.string()
    .min(5, "Minimum 5 characters")
    .max(128, "Maximum 128 characters")
    .required("Password is required"),
  passwordConfirm: Yup.string()
    .oneOf([Yup.ref("password"), undefined], "Passwords must match")
    .required("Password confirmation is required"),
  jobTitle: Yup.string()
    .required("Job title is required"),
  firstName: Yup.string()
    .max(255, "Maximum 255 characters")
    .required("First name is required"),
  lastName: Yup.string()
    .max(255, "Maximum 255 characters")
    .required("Last name is required"),
  addressLine1: Yup.string()
    .min(5, "Minimum 5 characters")
    .max(100, "Maximum 100 characters")
    .required("Address line 1 is required"),
  addressLine2: Yup.string()
    .min(5, "Minimum 5 characters")
    .max(50, "Maximum 100 characters"),
  city: Yup.string()
    .min(5, "Minimum 5 characters")
    .max(50, "Maximum 100 characters")
    .required("City is required"),
  state: Yup.string()
    .max(2, "Maximum 2 characters")
    .required("State is required"),
  zipCode: Yup.string()
    .matches(zipCodeRegex, "Zip code must be in the format xxxxx or xxxxx-xxxx")
    .required("Zip code is required"),
  phoneNumber: Yup.string()
    .matches(phoneRegex, "Phone number must be in the format (xxx) xxx-xxxx")
});


export default function UsersAddModal() {
  const [show, setShow] = useState(false);
  const handleClose = () => setShow(false);
  const handleShow = () => setShow(true);
  const [loading, setLoading] = useState(false);
  const [jobTitles, setJobTitles] = useState<JobTitlesOptionType[]>([]);
  const [jobTitlesLoading, setJobTitlesLoading] = useState(false);
  const [selectedState, setSelectedState] = useState(null);

  useEffect(() => {
    const fetchJobTitles = async () => {
      setJobTitlesLoading(true);
      try {
        const { data: jobTitles } = await axios.get("http://localhost:8000/api/job_titles/");
        console.log(jobTitles.results);
        setJobTitles(jobTitles.results);
      } catch (error: any) {
        console.log(error);
      } finally {
        setJobTitlesLoading(false);
      }
    };
    fetchJobTitles();
  }, []);

  const formik = useFormik({
    initialValues: {
      username: "",
      email: "",
      password: "",
      passwordConfirm: "",
      jobTitle: "",
      firstName: "",
      lastName: "",
      addressLine1: "",
      addressLine2: "",
      city: "",
      state: "",
      zipCode: ""
    },
    validationSchema: UsersSchema,
    onSubmit: async (values, { setStatus, setSubmitting }) => {
      setLoading(true);
      const data = {
        username: values.username,
        email: values.email,
        password: values.password,
        profile: {
          title: values.jobTitle,
          first_name: values.firstName,
          last_name: values.lastName,
          address_line_1: values.addressLine1,
          address_line_2: values.addressLine2,
          city: values.city,
          state: values.state,
          zip_code: values.zipCode
        }
      };

      axios.post("http://localhost:8000/api/users/", data, {
        headers: {
          "Content-Type": "application/json"
        }
      }).then((res) => {
        if (res.status === 201) {
          formik.resetForm();
          toast("User added successfully", {
            position: "top-right",
            autoClose: 5000,
            hideProgressBar: false
          });
        }
      }).catch(() => {
        toast("Something went wrong. Please try again later.", {
          position: "top-right",
          autoClose: 5000,
          hideProgressBar: false
        });
        setStatus("Something went wrong. Please try again later.");
        setSubmitting(false);
      }).then(() => {
        setSubmitting(false);
        setLoading(false);
      });
    }
  });

  return (
    <>
      <Button type="button"
              className="btn btn-primary"
              onClick={handleShow}>
        <MTSVG icon={<SvgArr009 />} className={"svg-icon-2"} />
        Add User
      </Button>
      <Modal
        dialogClassName={"modal-dialog modal-dialog-centered mw-650px"}
        show={show}
        tabIndex={-1}
        onHide={handleClose}
      >
        <Form
          className={"form"}
          noValidate
          id="mt_modal_add_user_form"
          onSubmit={formik.handleSubmit}
        >
          <Modal.Header className={"modal-header"}>
            <h2>Add User</h2>
            <div
              className="btn btn-icon btn-sm btn-active-light-primary ms-2"
              aria-label={"Close"}
              onClick={handleClose}>
              <MTSVG icon={<SvgArr061 />} className={"svg-icon-2x"} />
            </div>
          </Modal.Header>
          <Modal.Body className={"py-10 px-lg-16"}>
            <div className="scroll-y me-n7 pe-7"
                 id="kt_modal_new_address_scroll"
                 data-mt-scroll="true"
                 data-mt-scroll-activate="{default: false, lg: true}"
                 data-kt-scroll-max-height="auto"
                 data-mt-scroll-dependencies="#kt_modal_new_address_header"
                 data-mt-scroll-wrappers="#kt_modal_new_address_scroll"
                 data-kt-scroll-offset="300px"
                 style={{ maxHeight: "667px" }}
            >
              {formik.status && (
                <DangerAlert
                  title={formik.status}
                  message={"Something went wrong when processing your request. No need to worry we notified a human."}
                />
              )}

              {/* Username Field */}
              <Form.Group className="d-flex flex-column mb-8 fv-row">
                <Form.Label className="d-flex align-items-center fs-6 fw-semibold mb-2">
                  <span className="required">Username</span>
                </Form.Label>
                <Form.Control
                  placeholder="Enter Username"
                  {...formik.getFieldProps("username")}
                  className={clsx(
                    { "is-invalid": formik.touched.username && formik.errors.username },
                    { "is-valid": formik.touched.username && !formik.errors.username }
                  )}
                  type="text"
                  name="username" />
                {formik.touched.username && formik.errors.username && (
                  <div className="fv-plugins-message-container">
                    <div className="fv-help-block">{formik.errors.username}</div>
                  </div>
                )}
              </Form.Group>

              {/* Email Field */}
              <Form.Group className="d-flex flex-column mb-8 fv-row">
                <Form.Label className="d-flex align-items-center fs-6 fw-semibold mb-2">
                  <span className="required">Email Address</span>
                </Form.Label>
                <Form.Control
                  type="email"
                  placeholder="Enter Email Address"
                  {...formik.getFieldProps("email")}
                  className={clsx(
                    { "is-invalid": formik.touched.email && formik.errors.email },
                    { "is-valid": formik.touched.email && !formik.errors.email }
                  )}
                  name="email" />
                {formik.touched.email && formik.errors.email && (
                  <div className="fv-plugins-message-container">
                    <div className="fv-help-block">{formik.errors.email}</div>
                  </div>
                )}
                <Form.Text className="text-muted">
                  We&quot;ll never share your email with anyone else.
                </Form.Text>
              </Form.Group>

              {/* Password Field */}
              <Form.Group className="d-flex flex-column mb-8 fv-row">
                <Form.Label className="d-flex align-items-center fs-6 fw-semibold mb-2">
                  <span className="required">Password</span>
                </Form.Label>
                <Form.Control
                  type="password"
                  autoComplete="off"
                  placeholder="Enter Password"
                  {...formik.getFieldProps("password")}
                  className={clsx(
                    { "is-invalid": formik.touched.password && formik.errors.password },
                    { "is-valid": formik.touched.password && !formik.errors.password }
                  )}
                />
                {formik.touched.password && formik.errors.password && (
                  <div className="fv-plugins-message-container">
                    <div className="fv-help-block">{formik.errors.password}</div>
                  </div>
                )}
              </Form.Group>

              {/* Confirm Password Field */}
              <div className="d-flex flex-column mb-8 fv-row">
                <Form.Label className="d-flex align-items-center fs-6 fw-semibold mb-2">
                  <span className="required">Confirm Password</span>
                </Form.Label>
                <Form.Control
                  type="password"
                  autoComplete="off"
                  placeholder="Confirm Password"
                  {...formik.getFieldProps("passwordConfirm")}
                  className={clsx(
                    { "is-invalid": formik.touched.passwordConfirm && formik.errors.passwordConfirm },
                    { "is-valid": formik.touched.passwordConfirm && !formik.errors.passwordConfirm }
                  )}
                />
                {formik.touched.passwordConfirm && formik.errors.passwordConfirm && (
                  <div className="fv-plugins-message-container">
                    <div className="fv-help-block">{formik.errors.passwordConfirm}</div>
                  </div>
                )}
              </div>

              {/* Job Title Field */}
              <Form.Group className="d-flex flex-column mb-8 fv-row">
                <Form.Label className="d-flex align-items-center fs-6 fw-semibold mb-2">
                  <span className="required">Job Title</span>
                </Form.Label>
                <Form.Select
                  {...formik.getFieldProps("jobTitle")}
                  className={clsx(
                    { "is-invalid": formik.touched.jobTitle && formik.errors.jobTitle },
                    { "is-valid": formik.touched.jobTitle && !formik.errors.jobTitle }
                  )}>
                  {jobTitlesLoading ? (
                    <option value="">Loading...</option>
                  ) : (
                    <>
                      <option value="">Select Job Title</option>
                      {jobTitles.map((jobTitle) => (
                        <option key={jobTitle.id} value={jobTitle.id}>
                          {jobTitle.name}
                        </option>
                      ))}
                    </>
                  )}
                </Form.Select>
                {formik.touched.jobTitle && formik.errors.jobTitle && (
                  <div className="fv-plugins-message-container">
                    <div className="fv-help-block">{formik.errors.jobTitle}</div>
                  </div>
                )}
              </Form.Group>

              {/* First Name and Last Name */}
              <div className="row mb-5">
                <Form.Group className="col-md-6 fv-row fv-plugins-icon-container">
                  <Form.Label className="required fs-5 fw-semibold mb-2">First name</Form.Label>
                  <Form.Control
                    type="text"
                    placeholder="Enter First Name"
                    {...formik.getFieldProps("firstName")}
                    className={clsx(
                      { "is-invalid": formik.touched.firstName && formik.errors.firstName },
                      { "is-valid": formik.touched.firstName && !formik.errors.firstName }
                    )}
                  />
                  {formik.touched.firstName && formik.errors.firstName && (
                    <div className="fv-plugins-message-container">
                      <div className="fv-help-block">{formik.errors.firstName}</div>
                    </div>
                  )}
                </Form.Group>
                <Form.Group className="col-md-6 fv-row fv-plugins-icon-container">
                  <Form.Label className="required fs-5 fw-semibold mb-2">Last name</Form.Label>
                  <Form.Control
                    type="text"
                    placeholder="Enter Last Name"
                    {...formik.getFieldProps("lastName")}
                    className={clsx(
                      { "is-invalid": formik.touched.lastName && formik.errors.lastName },
                      { "is-valid": formik.touched.lastName && !formik.errors.lastName }
                    )}
                  />
                  {formik.touched.lastName && formik.errors.lastName && (
                    <div className="fv-plugins-message-container">
                      <div className="fv-help-block">{formik.errors.lastName}</div>
                    </div>
                  )}
                </Form.Group>
              </div>

              {/* Address Line 1*/}
              <Form.Group className="d-flex flex-column mb-8 fv-row">
                <Form.Label className="d-flex align-items-center fs-6 fw-semibold mb-2">
                  <span className="required">Address Line 1</span>
                </Form.Label>
                <Form.Control
                  type="text"
                  placeholder="Enter Address Line 1"
                  {...formik.getFieldProps("addressLine1")}
                  className={clsx(
                    { "is-invalid": formik.touched.addressLine1 && formik.errors.addressLine1 },
                    { "is-valid": formik.touched.addressLine1 && !formik.errors.addressLine1 }
                  )}
                />
                {formik.touched.addressLine1 && formik.errors.addressLine1 && (
                  <div className="fv-plugins-message-container">
                    <div className="fv-help-block">{formik.errors.addressLine1}</div>
                  </div>
                )}
              </Form.Group>

              {/* Address Line 2*/}
              <Form.Group className="d-flex flex-column mb-8 fv-row">
                <Form.Label className="d-flex align-items-center fs-6 fw-semibold mb-2">
                  <span>Address Line 2</span>
                </Form.Label>
                <Form.Control
                  type="text"
                  placeholder="Enter Address Line 2"
                  {...formik.getFieldProps("addressLine2")}
                  className={clsx(
                    { "is-invalid": formik.touched.addressLine2 && formik.errors.addressLine2 },
                    { "is-valid": formik.touched.addressLine2 && !formik.errors.addressLine2 }
                  )}
                />
                {formik.touched.addressLine2 && formik.errors.addressLine2 && (
                  <div className="fv-plugins-message-container">
                    <div className="fv-help-block">{formik.errors.addressLine2}</div>
                  </div>
                )}
              </Form.Group>

              {/* City */}
              <Form.Group className="d-flex flex-column mb-8 fv-row">
                <Form.Label className="d-flex align-items-center fs-6 fw-semibold mb-2">
                  <span className="required">City</span>
                </Form.Label>
                <Form.Control
                  type="text"
                  placeholder="Enter City"
                  {...formik.getFieldProps("city")}
                  className={clsx(
                    { "is-invalid": formik.touched.city && formik.errors.city },
                    { "is-valid": formik.touched.city && !formik.errors.city }
                  )}
                />
                {formik.touched.city && formik.errors.city && (
                  <div className="fv-plugins-message-container">
                    <div className="fv-help-block">{formik.errors.city}</div>
                  </div>
                )}
              </Form.Group>

              {/* State & Zip Code */}
              <div className="row mb-5">
                <Form.Group className="col-md-6 fv-row fv-plugins-icon-container">
                  <Form.Label className="required fs-5 fw-semibold mb-2">State</Form.Label>
                  <StateSelect
                    options={
                      stateChoices.map((state) => ({
                        value: state.abbr,
                        label: state.name,
                      }))
                    }
                    className={clsx(
                      { "is-invalid": formik.touched.state && formik.errors.state },
                      { "is-valid": formik.touched.state && !formik.errors.state }
                    )}
                    isSearchable={true}
                    isClearable={true}
                    name="state"
                    placeholder="Select State"
                  />
                  {/*<Form.Select*/}
                  {/*  {...formik.getFieldProps("state")}*/}
                  {/*  className={clsx(*/}
                  {/*    { "is-invalid": formik.touched.state && formik.errors.state },*/}
                  {/*    { "is-valid": formik.touched.state && !formik.errors.state }*/}
                  {/*  )}*/}
                  {/*>*/}
                  {/*  <>*/}
                  {/*    <option value="">Select State</option>*/}
                  {/*    {stateChoices.map((state) => (*/}
                  {/*      <option key={state.abbr} value={state.abbr}>*/}
                  {/*        {state.name}*/}
                  {/*      </option>*/}
                  {/*    ))}*/}
                  {/*  </>*/}
                  {/*</Form.Select>*/}
                  {formik.touched.state && formik.errors.state && (
                    <div className="fv-plugins-message-container">
                      <div className="fv-help-block">{formik.errors.state}</div>
                    </div>
                  )}
                </Form.Group>
                <Form.Group className="col-md-6 fv-row fv-plugins-icon-container">
                  <Form.Label className="required fs-5 fw-semibold mb-2">Zip Code</Form.Label>
                  <Form.Control
                    type="text"
                    placeholder="Enter Last Name"
                    {...formik.getFieldProps("zipCode")}
                    className={clsx(
                      { "is-invalid": formik.touched.zipCode && formik.errors.zipCode },
                      { "is-valid": formik.touched.zipCode && !formik.errors.zipCode }
                    )}
                  />
                  {formik.touched.zipCode && formik.errors.zipCode && (
                    <div className="fv-plugins-message-container">
                      <div className="fv-help-block">{formik.errors.zipCode}</div>
                    </div>
                  )}
                </Form.Group>
              </div>

            </div>
          </Modal.Body>
          <Modal.Footer className={"modal-footer flex-center"}>
            <button type="reset"
                    id="mt_modal_add_user_form_cancel"
                    onClick={handleClose}
                    className="btn btn-light me-3">
              Close
            </button>
            <button
              type="submit"
              id="mt_modal_add_user_form_submit"
              className="btn btn-primary"
              disabled={formik.isSubmitting || !formik.isValid}
            >
              {!loading && <span className="indicator-label">
                <MTSVG icon={<SvgGen016 />} />
                Submit
              </span>}
              {loading && (
                <span className="indicator-progress" style={{ display: "block" }}>Please wait...
                    <span className="spinner-border spinner-border-sm align-middle ms-2"></span>
                  </span>
              )}
            </button>
          </Modal.Footer>
        </Form>
      </Modal>
    </>
  );
}