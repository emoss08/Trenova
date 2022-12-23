import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

// project import
import config from 'config';
import useAuth from 'hooks/useAuth';

// types
import { GuardProps } from 'types/auth';

// ==============================|| GUEST GUARD ||============================== //

const GuestGuard = ({ children }: GuardProps) => {
  const { isLoggedIn } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (isLoggedIn) {
      navigate(config.defaultPath, { replace: true });
    }
  }, [isLoggedIn, navigate]);

  return children;
};

export default GuestGuard;
