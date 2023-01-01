import { ReactElement } from 'react';
import { MontaUser } from './user';

// third-party

// ==============================|| AUTH TYPES  ||============================== //

export type GuardProps = {
  children: ReactElement | null;
};

export interface AuthProps {
  isAuthenticated: boolean;
  isLoading: boolean;
  user?: MontaUser | null;
  isInitialized: boolean;

  token?: string | null;
}

export interface AuthActionProps {
  type: string;
  payload?: AuthProps;
}
