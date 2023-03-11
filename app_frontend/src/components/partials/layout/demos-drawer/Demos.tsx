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

import { toAbsoluteUrl } from '@/utils/helpers/AssetHelpers'
import React, {FC} from 'react'

const Demos: FC = () => {
  const demos = [
    {
      name: 'demo1',
      available: true,
    },
    {
      name: 'demo2',
      available: true,
    },
    {
      name: 'demo3',
      available: true,
    },
    {
      name: 'demo4',
      available: true,
    },
    {
      name: 'demo5',
      available: true,
    },
    {
      name: 'demo6',
    },
    {
      name: 'demo7',
    },
    {
      name: 'demo8',
    },
    {
      name: 'demo9',
    },
    {
      name: 'demo10',
    },
    {
      name: 'demo11',
    },
    {
      name: 'demo12',
    },
    {
      name: 'demo13',
    },
  ]

  return (
    <div className='mb-0'>
      <h3 className='fw-bolder text-center mb-6'>{process.env.NEXT_PUBLIC_APP_THEME_NAME} React Demos</h3>

      <div className='row g-5'>
        {demos.map((item, index) => (
          <div className='col-6' key={index}>
            <div
              className={`overlay overflow-hidden position-relative ${
                process.env.NEXT_PUBLIC_THEME_DEMO === item.name
                  ? 'border border-4 border-success'
                  : 'border border-4 border-gray-200'
              } rounded`}
            >
              <div className='overlay-wrapper'>
                <img
                  src={toAbsoluteUrl(`/media/demos/${item.name}.png`)}
                  alt='demo'
                  className={`w-100 ${!item.available && 'opacity-75'}`}
                />
              </div>

              <div className='overlay-layer bg-dark bg-opacity-10'>
                {item.available && (
                  <a
                    href={`${process.env.NEXT_APP_PREVIEW_REACT_URL}/${item.name}`}
                    className='btn btn-sm btn-success shadow'
                  >
                    {item.name.charAt(0).toUpperCase() + item.name.slice(1)}
                  </a>
                )}
                {!item.available && (
                  <div className='badge badge-white px-6 py-4 fw-bold fs-base shadow'>
                    Coming soon
                  </div>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

export {Demos}
