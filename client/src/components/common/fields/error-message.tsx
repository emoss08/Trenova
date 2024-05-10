import { faTriangleExclamation } from "@fortawesome/pro-regular-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";

export function ErrorMessage({ formError }: { formError?: string }) {
  return (
    <div className="mt-2 inline-block rounded bg-red-50 px-2 py-1 text-xs leading-tight text-red-500 dark:bg-red-300 dark:text-red-800 ">
      {formError ? formError : "An Error has occurred. Please try again."}
    </div>
  );
}

export function FieldErrorMessage({ formError }: { formError?: string }) {
  return (
    <>
      <div className="pointer-events-none absolute inset-y-0 right-0 mr-2.5 mt-1.5">
        <FontAwesomeIcon
          icon={faTriangleExclamation}
          className="text-red-500"
        />
      </div>
      <ErrorMessage formError={formError} />
    </>
  );
}
