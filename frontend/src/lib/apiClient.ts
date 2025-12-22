import axios from "axios";

// 基础配置：后端地址 (开发环境默认为 localhost:8888)
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://127.0.0.1:8888";

export const apiClient = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    "Content-Type": "application/json",
  },
});

// 请求拦截器：自动注入 Token
apiClient.interceptors.request.use((config) => {
  if (typeof window !== "undefined") {
    const token = localStorage.getItem("token");
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
  }
  return config;
});

// 响应拦截器：统一处理错误
apiClient.interceptors.response.use(
  (response) => {
    // 后端约定：code === 0 表示成功，否则为业务错误
    // 返回 response.data，这样前端可以直接访问 code, data, message
    return response.data;
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
      // 将错误信息包装成统一格式，方便前端处理
      const errorData = error.response?.data || {};
      return Promise.reject({
        ...error,
        code: errorData.code || 40100,
        message: errorData.message || "Unauthorized",
        data: errorData.data || null,
      });
    }
    // 其他错误也统一格式
    const errorData = error.response?.data || {};
    return Promise.reject({
      ...error,
      code: errorData.code || error.response?.status || -1,
      message: errorData.message || error.message || "请求失败",
      data: errorData.data || null,
    });
  }
);
