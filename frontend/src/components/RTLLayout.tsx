import { useEffect, ReactNode } from 'react';

// material-ui
import { CacheProvider } from '@emotion/react';
import createCache, { StylisPlugin } from '@emotion/cache';

// third-party
import rtlPlugin from 'stylis-plugin-rtl';

// project import
import useConfig from 'hooks/useConfig';

// ==============================|| RTL LAYOUT ||============================== //

interface Props {
  children: ReactNode;
}

const RTLLayout = ({ children }: Props) => {
  const { themeDirection } = useConfig();

  useEffect(() => {
    document.dir = themeDirection;
  }, [themeDirection]);

  const cacheRtl = createCache({
    key: themeDirection === 'rtl' ? 'rtl' : 'css',
    prepend: true,
    stylisPlugins: themeDirection === 'rtl' ? [rtlPlugin as StylisPlugin] : []
  });

  return <CacheProvider value={cacheRtl}>{children}</CacheProvider>;
};

export default RTLLayout;
