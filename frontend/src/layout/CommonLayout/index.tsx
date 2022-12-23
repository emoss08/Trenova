import { lazy, Suspense } from 'react';
import { Outlet } from 'react-router-dom';

// material-ui
import { styled } from '@mui/material/styles';
import LinearProgress, { LinearProgressProps } from '@mui/material/LinearProgress';

const Header = lazy(() => import('./Header'));
const FooterBlock = lazy(() => import('./FooterBlock'));

// ==============================|| Loader ||============================== //

const LoaderWrapper = styled('div')(({ theme }) => ({
  position: 'fixed',
  top: 0,
  left: 0,
  zIndex: 2001,
  width: '100%',
  '& > * + *': {
    marginTop: theme.spacing(2)
  }
}));

export interface LoaderProps extends LinearProgressProps {}

const Loader = () => (
  <LoaderWrapper>
    <LinearProgress color="primary" />
  </LoaderWrapper>
);

// ==============================|| MINIMAL LAYOUT ||============================== //

const CommonLayout = ({ layout = 'blank' }: { layout?: string }) => (
  <>
    {(layout === 'landing' || layout === 'simple') && (
      <Suspense fallback={<Loader />}>
        <Header layout={layout} />
        <Outlet />
        <FooterBlock isFull={layout === 'landing'} />
      </Suspense>
    )}
    {layout === 'blank' && <Outlet />}
  </>
);

export default CommonLayout;
