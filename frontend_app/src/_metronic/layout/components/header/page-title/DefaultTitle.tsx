import {FC} from 'react'
import clsx from 'clsx'
import {Link} from 'react-router-dom'
import {useLayout} from '../../../core/LayoutProvider'
import {usePageData} from '../../../core/PageData'

const DefaultTitle: FC = () => {
  const {pageTitle, pageDescription, pageBreadcrumbs} = usePageData()
  const {config} = useLayout()
  return (
    <div className='page-title d-flex justify-content-center flex-column me-5'>
      {/* begin::Title */}
      {pageTitle && (
        <h1 className='d-flex flex-column text-dark fw-bolder fs-3 mb-0'>
          {pageTitle}
          {pageDescription && config.pageTitle && config.pageTitle.description && (
            <>
              <span className='h-20px border-gray-200 border-start ms-3 mx-2'></span>
              <small className='text-muted fs-7 fw-bold my-1 ms-1'>{pageDescription}</small>
            </>
          )}
        </h1>
      )}
      {/* end::Title */}

      {pageBreadcrumbs && pageBreadcrumbs.length > 0 && (
        <ul className='breadcrumb breadcrumb-separatorless fw-bold fs-7 pt-1'>
          {Array.from(pageBreadcrumbs).map((item, index) => (
            <li
              className={clsx('breadcrumb-item', {
                'text-dark': !item.isSeparator && item.isActive,
                'text-muted': !item.isSeparator && !item.isActive,
              })}
              key={`${item.path}${index}`}
            >
              {!item.isSeparator ? (
                <Link className='text-muted text-hover-primary' to={item.path}>
                  {item.title}
                </Link>
              ) : (
                <span className='bullet bg-gray-200 w-5px h-2px'></span>
              )}
            </li>
          ))}
          <li className='breadcrumb-item text-dark'>{pageTitle}</li>
        </ul>
      )}
    </div>
  )
}

export {DefaultTitle}
