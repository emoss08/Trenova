import axios from "axios";
import { AuthModel, UserModel } from "./_models";
import ApiService from "../../../../_monta/helpers/ApiService";

const API_URL = process.env.REACT_APP_API_URL;

export const GET_USER_BY_ACCESSTOKEN_URL: string = `${API_URL}/token/verify/`;
export const LOGIN_URL: string = `${API_URL}/token/provision/`;
export const REGISTER_URL: string = `${API_URL}/register`;
export const REQUEST_PASSWORD_URL: string = `${API_URL}/forgot_password`;

export function login(username: string, password: string) {

  return ApiService.post<AuthModel>(LOGIN_URL, { username, password });
}

export function register(
  email: string,
  firstname: string,
  lastname: string,
  password: string,
  password_confirmation: string
) {
  return ApiService.post(REGISTER_URL, {
    email,
    first_name: firstname,
    last_name: lastname,
    password,
    password_confirmation
  });

}

export function requestPassword(email: string) {
  return axios.post<{ result: boolean }>(REQUEST_PASSWORD_URL, {
    email
  });
}

export function getUserByToken(token: string) {
  return ApiService.post<UserModel>(GET_USER_BY_ACCESSTOKEN_URL, {
    token: token
  });
}
