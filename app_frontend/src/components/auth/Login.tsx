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

/* eslint-disable jsx-a11y/anchor-is-valid */
import { useState } from "react";
import * as Yup from "yup";
import clsx from "clsx";
import { useFormik } from "formik";
import Link from "next/link";
import { getUserByToken, login } from "@/utils/_requests";
import { authStore, saveAuth } from "@/utils/providers/AuthGuard";
import Button from "react-bootstrap/Button";
import { Form } from "react-bootstrap";

const loginSchema = Yup.object().shape({
  username: Yup.string()
    .min(3, "Minimum 3 symbols")
    .max(50, "Maximum 50 symbols")
    .required("Email is required"),
  password: Yup.string()
    .min(3, "Minimum 3 symbols")
    .max(50, "Maximum 50 symbols")
    .required("Password is required")
});

const initialValues = {
  username: "sys",
  password: "system"
};


export function Login() {
  const [loading, setLoading] = useState(false);

  const formik = useFormik({
    initialValues,
    validationSchema: loginSchema,
    onSubmit: async (values, { setStatus, setSubmitting }) => {
      setLoading(true);
      try {
        const { data: auth } = await login(values.username, values.password);
        saveAuth(auth);
        const { data: currentUser } = await getUserByToken(auth.token);
        authStore.set('user', currentUser);
      } catch (error) {
        saveAuth(undefined);
        setStatus("The login detail is incorrect");
        setSubmitting(false);
        setLoading(false);
      } finally {
        setSubmitting(false);
        setLoading(false);
      }
    }
  });

  return (
    <Form
      className="form w-100"
      onSubmit={formik.handleSubmit}
      noValidate
      id="mt_login_signin_form"
    >
      {/* begin::Heading */}
      <div className="text-center mb-10">
        <h1 className="text-dark mb-3">Sign In to Monta</h1>
        <div className="text-gray-400 fw-bold fs-4">
          New Here?{" "}
          <Link href="/auth/registration" className="link-primary fw-bolder">
            Create an Account
          </Link>
        </div>
      </div>
      {/* begin::Heading */}

      {formik.status && (
        <div className="mb-lg-15 alert alert-danger">
          <div className="alert-text font-weight-bold">{formik.status}</div>
        </div>
      )}

      {/* begin::Form group */}
      <Form.Group className="fv-row mb-10">
        <Form.Label className="form-label fs-6 fw-bolder text-dark">Username</Form.Label>
        <Form.Control
          placeholder="Username"
          {...formik.getFieldProps("username")}
          className={clsx(
            { "is-invalid": formik.touched.username && formik.errors.username },
            {"is-valid": formik.touched.username && !formik.errors.username}
          )}
          type="text"
          name="username"
          autoComplete="off"
        />
        {formik.touched.username && formik.errors.username && (
          <div className="fv-plugins-message-container">
            <span role="alert">{formik.errors.username}</span>
          </div>
        )}
      </Form.Group>
      <Form.Group className="fv-row mb-10">
        <div className="d-flex justify-content-between mt-n5">
          <div className="d-flex flex-stack mb-2">
            <Form.Label className="form-label fw-bolder text-dark fs-6 mb-0">Password</Form.Label>
          </div>
        </div>
        <Form.Control
          type="password"
          autoComplete="off"
          {...formik.getFieldProps("password")}
          className={clsx(
            {"is-invalid": formik.touched.password && formik.errors.password
            },
            {"is-valid": formik.touched.password && !formik.errors.password}
          )}
        />
        {formik.touched.password && formik.errors.password && (
          <div className="fv-plugins-message-container">
            <div className="fv-help-block">
              <span role="alert">{formik.errors.password}</span>
            </div>
          </div>
        )}
      </Form.Group>
      {/* end::Form group */}

      {/* begin::Action */}
      <div className="text-center">
        <Button
          type="submit"
          id="mt_sign_in_submit"
          className="btn btn-lg btn-primary w-100 mb-5"
          disabled={formik.isSubmitting || !formik.isValid}
        >
          {!loading && <span className="indicator-label">Continue</span>}
          {loading && (
            <span className="indicator-progress" style={{ display: "block" }}>
              Please wait...
              <span className="spinner-border spinner-border-sm align-middle ms-2"></span>
            </span>
          )}
        </Button>
      </div>
      {/* end::Action */}
    </Form>
  );
}
