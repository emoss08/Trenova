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
import Image from "next/image";
import {FC} from 'react'
import { toAbsoluteUrl } from "@/utils/helpers/AssetHelpers";
import unitedStates from '../../../../../public/media/flags/united-states.svg'
import china from '../../../../../public/media/flags/china.svg'
import spain from '../../../../../public/media/flags/spain.svg'
import japan from '../../../../../public/media/flags/japan.svg'
import germany from '../../../../../public/media/flags/germany.svg'
import france from '../../../../../public/media/flags/france.svg'

const languages = [
  {
    lang: 'en',
    name: 'English',
    flag: unitedStates,
  },
  {
    lang: 'zh',
    name: 'Mandarin',
    flag: china,
  },
  {
    lang: 'es',
    name: 'Spanish',
    flag: spain,
  },
  {
    lang: 'ja',
    name: 'Japanese',
    flag: japan,
  },
  {
    lang: 'de',
    name: 'German',
    flag: germany,
  },
  {
    lang: 'fr',
    name: 'French',
    flag: france,
  },
]

const Languages: FC = () => {
  return (
    <div
      className='menu-item px-5'
      data-mt-menu-trigger='hover'
      data-mt-menu-placement='left-start'
      data-mt-menu-flip='bottom'
    >
      <a href='#' className='menu-link px-5'>
        <span className='menu-title position-relative'>
          Language
          <span className='fs-8 rounded bg-light px-3 py-2 position-absolute translate-middle-y top-50 end-0'>
            {/*{currentLanguage?.name}{' '}*/}
            <Image
              className='w-15px h-15px rounded-1 ms-2'
              src={unitedStates}
              alt='metronic'
            />
          </span>
        </span>
      </a>

      <div className='menu-sub menu-sub-dropdown w-175px py-4'>
        {languages.map((l) => (
          <div
            className='menu-item px-3'
            key={l.lang}
            onClick={() => {
              // setLanguage(l.lang)
            }}
          >
            <a
              href='#'
              // className={clsx('menu-link d-flex px-5', {active: l.lang === currentLanguage?.lang})}
            >
              <span className='symbol symbol-20px me-4'>
                <Image className='rounded-1' src={l.flag} alt='metronic' />
              </span>
              {l.name}
            </a>
          </div>
        ))}
      </div>
    </div>
  )
}

export {Languages}
