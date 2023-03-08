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
import Image from "next/image";
import {FC} from 'react'
import { toAbsoluteUrl } from "@/utils/AssetHelpers";
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
      data-kt-menu-trigger='hover'
      data-kt-menu-placement='left-start'
      data-kt-menu-flip='bottom'
    >
      <a href='#' className='menu-link px-5'>
        <span className='menu-title position-relative'>
          Language
          <span className='fs-8 rounded bg-light px-3 py-2 position-absolute translate-middle-y top-50 end-0'>
            {/*{currentLanguage?.name}{' '}*/}
            <img
              className='w-15px h-15px rounded-1 ms-2'
              // src={currentLanguage?.flag}
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
