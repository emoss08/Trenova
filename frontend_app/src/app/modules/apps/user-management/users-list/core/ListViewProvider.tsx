import {FC, useState, createContext, useContext, useMemo} from 'react'
import {
  ID,
  calculatedGroupingIsDisabled,
  calculateIsAllDataSelected,
  groupingOnSelect,
  initialListView,
  ListViewContextProps,
  groupingOnSelectAll,
  WithChildren,
} from '../../../../../../_metronic/helpers'
import {useQueryResponse, useQueryResponseData} from './QueryResponseProvider'

const ListViewContext = createContext<ListViewContextProps>(initialListView)

const ListViewProvider: FC<WithChildren> = ({children}) => {
  const [selected, setSelected] = useState<Array<ID>>(initialListView.selected)
  const [itemIdForUpdate, setItemIdForUpdate] = useState<ID>(initialListView.itemIdForUpdate)
  const {isLoading} = useQueryResponse()
  const data = useQueryResponseData()
  const disabled = useMemo(() => calculatedGroupingIsDisabled(isLoading, data), [isLoading, data])
  const isAllSelected = useMemo(() => calculateIsAllDataSelected(data, selected), [data, selected])

  return (
    <ListViewContext.Provider
      value={{
        selected,
        itemIdForUpdate,
        setItemIdForUpdate,
        disabled,
        isAllSelected,
        onSelect: (id: ID) => {
          groupingOnSelect(id, selected, setSelected)
        },
        onSelectAll: () => {
          groupingOnSelectAll(isAllSelected, setSelected, data)
        },
        clearSelected: () => {
          setSelected([])
        },
      }}
    >
      {children}
    </ListViewContext.Provider>
  )
}

const useListView = () => useContext(ListViewContext)

export {ListViewProvider, useListView}
