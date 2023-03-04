import {Navigate, Route, Routes, Outlet} from 'react-router-dom'
import {PageLink, PageTitle} from '../../../../_metronic/layout/core'
import {Private} from './components/Private'
import {Group} from './components/Group'
import {Drawer} from './components/Drawer'

const chatBreadCrumbs: Array<PageLink> = [
  {
    title: 'Chat',
    path: '/apps/chat/private-chat',
    isSeparator: false,
    isActive: false,
  },
  {
    title: '',
    path: '',
    isSeparator: true,
    isActive: false,
  },
]

const ChatPage = () => {
  return (
    <Routes>
      <Route element={<Outlet />}>
        <Route
          path='private-chat'
          element={
            <>
              <PageTitle breadcrumbs={chatBreadCrumbs}>Private chat</PageTitle>
              <Private />
            </>
          }
        />
        <Route
          path='group-chat'
          element={
            <>
              <PageTitle breadcrumbs={chatBreadCrumbs}>Group chat</PageTitle>
              <Group />
            </>
          }
        />
        <Route
          path='drawer-chat'
          element={
            <>
              <PageTitle breadcrumbs={chatBreadCrumbs}>Drawer chat</PageTitle>
              <Drawer />
            </>
          }
        />
        <Route index element={<Navigate to='/apps/chat/private-chat' />} />
      </Route>
    </Routes>
  )
}

export default ChatPage
