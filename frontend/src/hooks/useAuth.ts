import { useContext } from 'react';

// auth provider
import { MontaAuthContext } from '../contexts/MontaContext';

// ==============================|| AUTH HOOKS ||============================== //

const useAuth = () => {
  const context = useContext(MontaAuthContext);

  if (!context) throw new Error('context must be use inside provider');

  return context;
};

export default useAuth;
