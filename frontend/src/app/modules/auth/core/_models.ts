export interface AuthModel {
  token: string
  refreshToken?: string
}

export interface UserAddressModel {
  addressLine: string
  city: string
  state: string
  postCode: string
}

export interface UserCommunicationModel {
  email: boolean
  sms: boolean
  phone: boolean
}

export interface UserEmailSettingsModel {
  emailNotification?: boolean
  sendCopyToPersonalEmail?: boolean
  activityRelatesEmail?: {
    youHaveNewNotifications?: boolean
    youAreSentADirectMessage?: boolean
    someoneAddsYouAsAsAConnection?: boolean
    uponNewOrder?: boolean
    newMembershipApproval?: boolean
    memberRegistration?: boolean
  }
  updatesFromKeenthemes?: {
    newsAboutKeenthemesProductsAndFeatureUpdates?: boolean
    tipsOnGettingMoreOutOfKeen?: boolean
    thingsYouMissedSindeYouLastLoggedIntoKeen?: boolean
    newsAboutStartOnPartnerProductsAndOtherServices?: boolean
    tipsOnStartBusinessProducts?: boolean
  }
}

export interface UserSocialNetworksModel {
  linkedIn: string
  facebook: string
  twitter: string
  instagram: string
}

export interface UserModel {
  user: {
  id: string;
  organization: string;
  department: string;
  username: string;
  email: string;
  profile: {
    user: string;
    first_name: string;
    last_name: string;
    title: string;
    address: string;
    address_line_1: string;
    address_line_2: string;
    city: string;
    state: string;
    zip_code: string;
    phone: string;
  }
  }
}

