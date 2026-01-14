import axios, { AxiosInstance, AxiosResponse } from 'axios';
import { ApiResponse } from '@/types';

// 基础配置：后端地址 (开发环境默认为 localhost:8888)
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://127.0.0.1:8888";

const instance: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    "Content-Type": "application/json",
  },
});

// 请求拦截器：自动注入 Token
instance.interceptors.request.use((config) => {
  if (typeof window !== "undefined") {
    const token = localStorage.getItem("token");
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
  }
  return config;
});

// 响应拦截器：统一处理错误
instance.interceptors.response.use(
  (response: AxiosResponse<ApiResponse>) => {
    // 这里直接返回 data，也就是 ApiResponse 对象
    // 注意：这里只是运行时的解包
    return response.data as any;
  },
  (error) => {
    // 处理 401 未授权
    if (error.response?.status === 401) {
      // 如果已经在登录页，不要跳转，让登录页自己处理错误
      if (typeof window !== "undefined" && !window.location.pathname.includes("/login")) {
        // 清除失效 Token
        localStorage.removeItem("token");
        window.location.href = "/login";
      }
    }
    return Promise.reject(error);
  }
);

// 核心：封装一个带泛型的 API 客户端，彻底解决类型问题
export const apiClient = {
  // 告诉 TS，get 方法返回的是 Promise<ApiResponse<T>>
  get: <T>(url: string) => instance.get<any, ApiResponse<T>>(url),
  post: <T>(url: string, data?: any) => instance.post<any, ApiResponse<T>>(url, data),
  put: <T>(url: string, data?: any) => instance.put<any, ApiResponse<T>>(url, data),
  delete: <T>(url: string) => instance.delete<any, ApiResponse<T>>(url),
  patch: <T>(url: string, data?: any) => instance.patch<any, ApiResponse<T>>(url, data),
};
