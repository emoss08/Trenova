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

import { FC, useRef, useEffect, PropsWithChildren } from "react";
import clsx from 'clsx'
import { checkIsActive } from '@/utils/helpers/RouteHelpers'
import { useRouter } from "next/router";
import Image from "next/image";

type Props = {
  to: string
  title: string
  icon?: string
  fontIcon?: string
  menuTrigger?: 'click' | `{default:'click', lg: 'hover'}`
  menuPlacement?: 'right-start' | 'bottom-start' | 'left-start'
  hasArrow?: boolean
  hasBullet?: boolean
  isMega?: boolean
}

const MenuInnerWithSub: FC<Props & PropsWithChildren> = ({
  children,
  to,
  title,
  icon,
  fontIcon,
  menuTrigger,
  menuPlacement,
  hasArrow = false,
  hasBullet = false,
  isMega = false,
}) => {
  const menuItemRef = useRef<HTMLDivElement>(null)
  const {pathname} = useRouter()

  useEffect(() => {
    if (menuItemRef.current && menuTrigger && menuPlacement) {
      menuItemRef.current.setAttribute('data-mt-menu-trigger', menuTrigger)
      menuItemRef.current.setAttribute('data-mt-menu-placement', menuPlacement)
    }
  }, [menuTrigger, menuPlacement])

  return (
    <div ref={menuItemRef} className='menu-item menu-lg-down-accordion me-lg-1'>
      <span
        className={clsx('menu-link py-3', {
          active: checkIsActive(pathname, to),
        })}
      >
        {hasBullet && (
          <span className='menu-bullet'>
            <span className='bullet bullet-dot'></span>
          </span>
        )}

        {icon && (
          <span className='menu-icon'>
            <Image src={icon} className='svg-icon-2'  alt={"img"}/>
          </span>
        )}

        {fontIcon && (
          <span className='menu-icon'>
            <i className={clsx('bi fs-3', fontIcon)}></i>
          </span>
        )}

        <span className='menu-title'>{title}</span>

        {hasArrow && <span className='menu-arrow'></span>}
      </span>
      <div
        className={clsx(
          'menu-sub menu-sub-lg-down-accordion menu-sub-lg-dropdown',
          isMega ? 'w-100 w-lg-600px p-5 p-lg-5' : 'menu-rounded-0 py-lg-4 w-lg-225px'
        )}
        data-mt-menu-dismiss='true'
      >
        {children}
      </div>
    </div>
  )
}

export {MenuInnerWithSub}
