import { useContext } from 'react';
import { ConfigContext } from 'contexts/ConfigContext';

// ==============================|| CONFIG - HOOKS  ||============================== //

const useConfig = () => useContext(ConfigContext);

export default useConfig;
