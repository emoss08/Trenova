export type MontaUserProfile = {
  uid: string;
  title: string;
  firstName: string;
  lastName: string;
  addressLine1: string;
  addressLine2?: string;
  city: string;
  state: string;
  zipCode: string;
  phone: string;
};

export type MontaUser = {
  uid: string;
  organization: string;
  department: string;
  email: string;
  username: string;
  profile: MontaUserProfile;
};

export type UserContextType = {
  uid?: string;
  isAuthenticated: boolean;
  isLoading: boolean;
  token?: string | null;
  user?: MontaUser | null | undefined;
  authenticate: (username: string, password: string) => Promise<({ isAuthenticated: true } & ProvisionResult) | { isAuthenticated: false }>;
  logout: () => void;
};

export type ProvisionResult = {
  token: string;
  user: {
    id: string;
    username: string;
    organization: string;
    department: string;
    email: string;
    profile: {
      id: string;
      first_name: string;
      last_name: string;
      title: string;
      address_line_1: string;
      address_line_2?: string;
      city: string;
      state: string;
      zip_code: string;
      phone: string;
    };
  };
};