import React from 'react'
import { MasterLayout } from './MasterLayout'

describe('<MasterLayout />', () => {
  it('renders', () => {
    // see: https://on.cypress.io/mounting-react
    cy.mount(<MasterLayout />)
  })
})