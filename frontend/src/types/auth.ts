import { ReactElement } from 'react';

// third-party
import { MontaUser } from '../contexts/MontaContext';

// ==============================|| AUTH TYPES  ||============================== //

export type GuardProps = {
  children: ReactElement | null;
};

export interface AuthProps {
  isAuthenticated: boolean;
  isInitialized?: boolean;
  user?: MontaUser | null;
  token?: string | null;
}

export interface AuthActionProps {
  type: string;
  payload?: AuthProps;
}
