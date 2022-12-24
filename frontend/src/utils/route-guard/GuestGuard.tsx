import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

// project import
import config from 'config';
import useAuth from 'hooks/useAuth';

// types
import { GuardProps } from 'types/auth';

// ==============================|| GUEST GUARD ||============================== //

const GuestGuard = ({ children }: GuardProps) => {
  const { isAuthenticated } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (isAuthenticated) {
      navigate(config.defaultPath, { replace: true });
    }
  }, [isAuthenticated, navigate]);

  return children;
};

export default GuestGuard;
