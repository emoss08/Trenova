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

import {FC, useEffect} from 'react'
import clsx from 'clsx'
import {useLayout} from '@/utils/layout/LayoutProvider'

import {DrawerComponent} from '@/utils/assets/ts/components'
import {PropsWithChildren} from "react";
import { useRouter } from "next/router";


const Content: FC<PropsWithChildren> = ({children}) => {
  const {classes} = useLayout()
  const location = useRouter()
  useEffect(() => {
    DrawerComponent.hideAll()
  }, [location])

  return (
    <div id='mt_content_container' className={'container-xxl'}>
      {children}
    </div>
  )
}

export {Content}
