/* eslint-disable react-hooks/exhaustive-deps */
import {FC, useState, useEffect} from 'react'
import {createPortal} from 'react-dom'
import {WithChildren} from '../../../helpers'

/**
 * React portal based on https://stackoverflow.com/a/59154364
 * @param children Child elements
 * @param className CSS classname
 * @param el HTML element to create.  default: div
 */
const Portal: FC<{className?: string} & WithChildren> = ({children, className = ''}) => {
  const [container] = useState(document.createElement('div'))

  if (className) container.classList.add(className)

  useEffect(() => {
    document.body.appendChild(container)
    return () => {
      document.body.removeChild(container)
    }
  }, [])

  return createPortal(children, container)
}

export {Portal}
