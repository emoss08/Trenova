import {FC, PropsWithChildren} from 'react'
import {HeaderProps} from 'react-table'
import {useListView} from '../../core/ListViewProvider'
import {User} from '../../core/_models'

type Props = {
  tableProps: PropsWithChildren<HeaderProps<User>>
}

const UserSelectionHeader: FC<Props> = ({tableProps}) => {
  const {isAllSelected, onSelectAll} = useListView()
  return (
    <th {...tableProps.column.getHeaderProps()} className='w-10px pe-2'>
      <div className='form-check form-check-sm form-check-custom form-check-solid me-3'>
        <input
          className='form-check-input'
          type='checkbox'
          data-kt-check={isAllSelected}
          data-kt-check-target='#kt_table_users .form-check-input'
          checked={isAllSelected}
          onChange={onSelectAll}
        />
      </div>
    </th>
  )
}

export {UserSelectionHeader}
