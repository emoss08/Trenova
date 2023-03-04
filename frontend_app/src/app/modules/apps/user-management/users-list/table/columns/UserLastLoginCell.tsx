import {FC} from 'react'

type Props = {
  last_login?: string
}

const UserLastLoginCell: FC<Props> = ({last_login}) => (
  <div className='badge badge-light fw-bolder'>{last_login}</div>
)

export {UserLastLoginCell}
