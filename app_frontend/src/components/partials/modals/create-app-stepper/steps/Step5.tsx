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

/* eslint-disable jsx-a11y/anchor-is-valid */

import { toAbsoluteUrl } from "@/utils/helpers/AssetHelpers";

const Step5 = () => {
  return (
    <>
      <div data-mt-stepper-element='content'>
        <div className='w-100 text-center'>
          {/* begin::Heading */}
          <h1 className='fw-bold text-dark mb-3'>Release!</h1>
          {/* end::Heading */}

          {/* begin::Description */}
          <div className='text-muted fw-semibold fs-3'>
            Submit your app to kickstart your project.
          </div>
          {/* end::Description */}

          {/* begin::Illustration */}
          <div className='text-center px-4 py-15'>
            <img
              src={toAbsoluteUrl('/media/illustrations/sketchy-1/9.png')}
              alt=''
              className='mw-100 mh-300px'
            />
          </div>
          {/* end::Illustration */}
        </div>
      </div>
    </>
  )
}

export {Step5}
