import React from 'react'
import UserManagement from './user-management'

describe('<UserManagement />', () => {
  it('renders', () => {
    // see: https://on.cypress.io/mounting-react
    cy.mount(<UserManagement />)
  })
})