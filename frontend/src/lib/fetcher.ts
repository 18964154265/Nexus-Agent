import { apiClient } from "@/lib/apiClient";
import { ApiResponse } from "@/types";

// 定义一个自定义错误类，方便 UI 层获取错误信息
export class ApiError extends Error {
  constructor(public code: number, message: string) {
    super(message);
    this.name = 'ApiError';
  }
}

export const fetcher = async <T>(url: string): Promise<T> => {
  // 1. 发起请求，此时 res 已经是 ApiResponse<T> 类型，不需要 as unknown
  const res = await apiClient.get<T>(url);

  // 2. 集中处理业务逻辑错误
  if (res.code !== 0) {
    // 3. 抛出错误，让 SWR 的 error 状态捕获
    throw new ApiError(res.code, res.message || '请求失败');
  }

  // 4. 返回纯净的业务数据 (T)
  return res.data;
};

