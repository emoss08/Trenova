import { type API_ENDPOINTS } from "@/types/server";
import "axios";

declare module "axios" {
  interface Axios {
    get<T = any, R = AxiosResponse<T>, D = any>(
      url: API_ENDPOINTS,
      config?: AxiosRequestConfig<D>,
    ): Promise<R>;

    delete<T = any, R = AxiosResponse<T>, D = any>(
      url: API_ENDPOINTS,
      config?: AxiosRequestConfig<D>,
    ): Promise<R>;

    head<T = any, R = AxiosResponse<T>, D = any>(
      url: API_ENDPOINTS,
      config?: AxiosRequestConfig<D>,
    ): Promise<R>;

    options<T = any, R = AxiosResponse<T>, D = any>(
      url: API_ENDPOINTS,
      config?: AxiosRequestConfig<D>,
    ): Promise<R>;

    post<T = any, R = AxiosResponse<T>, D = any>(
      url: API_ENDPOINTS,
      data?: D,
      config?: AxiosRequestConfig<D>,
    ): Promise<R>;

    put<T = any, R = AxiosResponse<T>, D = any>(
      url: API_ENDPOINTS,
      data?: D,
      config?: AxiosRequestConfig<D>,
    ): Promise<R>;

    patch<T = any, R = AxiosResponse<T>, D = any>(
      url: API_ENDPOINTS,
      data?: D,
      config?: AxiosRequestConfig<D>,
    ): Promise<R>;
  }
}
