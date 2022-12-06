import axios, { AxiosResponse } from "axios";
import LocalStorageService from "./LocalStorageService";

class ApiService {
  /**
   * @description set the default HTTP request headers
   */
  public static setHeader(): void {
    axios.defaults.headers.common["Authorization"] = `Token ${LocalStorageService.getToken()}`;
    axios.defaults.headers.common["accept"] = "application/json";
  }

  /**
   * @description send the GET HTTP request
   * @param resource: string
   * @param params: any
   * @returns Promise<AxiosResponse>
   */
  public static query(resource: string, params: any): Promise<AxiosResponse> {
    return axios.get(`${resource}`, params).catch(error => {
      throw new Error(`[RWV] ApiService ${error}`);
    });
  }

  /**
   * @description send the GET HTTP request
   * @param resource: string
   * @param slug: string
   * @returns Promise<AxiosResponse>
   */
  public static get(resource: string, slug: string = ""): Promise<AxiosResponse> {
    return axios.get(`${resource}/${slug}`).catch(error => {
      throw new Error(`[RWV] ApiService ${error}`);
    });
  }

  /**
   * @description set the POST HTTP request
   * @param resource: string
   * @param params: any
   * @returns Promise<AxiosResponse>
   */
  public static post<Type>(resource: string, params: any): Promise<AxiosResponse> {
    return axios.post(`${resource}`, params);
  }

  /**
   * @description send the UPDATE HTTP request
   * @param resource: string
   * @param slug: string
   * @param params: any
   * @returns Promise<AxiosResponse>
   */
  public static update(resource: string, slug: string, params: any): Promise<AxiosResponse> {
    return axios.put(`${resource}/${slug}`, params);
  }

  /**
   * @description Send the PUT HTTP request
   * @param resource: string
   * @param params: any
   * @returns Promise<AxiosResponse>
   */
  public static put(resource: string, params: any): Promise<AxiosResponse> {
    return axios.put(`${resource}`, params);
  }

  /**
   * @description Send the DELETE HTTP request
   * @param resource: string
   * @returns Promise<AxiosResponse>
   */
  public static delete(resource: string): Promise<AxiosResponse> {
    return axios.delete(`${resource}`).catch(error => {
      throw new Error(`[RWV] ApiService ${error}`);
    });
  }

  /**
   * @description Send the PATCH HTTP request
   * @param resource: string
   * @param params: any
   * @returns Promise<AxiosResponse>
   */
  public static patch(resource: string, params: any): Promise<AxiosResponse> {
    return axios.patch(`${resource}`, params);
  }
}

export default ApiService;