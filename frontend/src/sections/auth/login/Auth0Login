import { useState } from 'react';

// material-ui
import { Button, FormHelperText, Grid } from '@mui/material';

// project import
import useAuth from 'hooks/useAuth';
import useScriptRef from 'hooks/useScriptRef';
import AnimateButton from 'components/@extended/AnimateButton';

// assets
import { LockOutlined } from '@ant-design/icons';

// ============================|| AUTH0 - LOGIN ||============================ //

const AuthLogin = () => {
  const { login } = useAuth();
  const scriptedRef = useScriptRef();

  const [error, setError] = useState(null);
  const loginHandler = async () => {
    try {
      await login();
    } catch (err: any) {
      if (scriptedRef.current) {
        setError(err.message);
      }
    }
  };

  return (
    <Grid container justifyContent="center" alignItems="center" spacing={2}>
      {error && (
        <Grid item xs={12}>
          <FormHelperText error>{error}</FormHelperText>
        </Grid>
      )}

      <Grid item xs={12}>
        <AnimateButton>
          <Button onClick={loginHandler} variant="contained" fullWidth startIcon={<LockOutlined />}>
            Log in with Auth0
          </Button>
        </AnimateButton>
      </Grid>
    </Grid>
  );
};

export default AuthLogin;
