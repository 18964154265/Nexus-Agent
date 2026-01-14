"use client";

import { useState } from "react";
import { useForm } from "react-hook-form";
import { useRouter } from "next/navigation";
import { apiClient } from "@/lib/apiClient";
import { Loader2 } from "lucide-react";
import Link from "next/link";
import type { ApiResponse, LoginResp } from "@/types";

export default function LoginPage() {
  const [isRegister, setIsRegister] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState("");
  const router = useRouter();

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm();

  const onSubmit = async (data: any) => {
    setIsLoading(true);
    setErrorMsg("");

    try {
      if (isRegister) {
        // 注册
        await apiClient.post("/api/auth/register", {
          email: data.email,
          password: data.password,
          name: data.name || data.email.split("@")[0],
        });
        // 注册成功后自动切换到登录或直接登录（视后端实现而定，这里先切换回登录）
        setIsRegister(false);
        setErrorMsg("注册成功，请登录");
      } else {
        // 登录
        const res = await apiClient.post<LoginResp>("/api/auth/login", {
          email: data.email,
          password: data.password,
        });

        // 后端返回格式: { code: 0, data: { access_token: "...", refresh_token: "...", user: {...} }, message: "success" }
        if (res.code === 0 && res.data) {
          const token = res.data.access_token;
          if (token) {
            localStorage.setItem("token", token);
            router.push("/dashboard");
          } else {
            setErrorMsg("登录响应中缺少 access_token");
          }
        } else {
          setErrorMsg(res.message || "登录失败");
        }
      }
    } catch (err: any) {
      console.error("Login error:", err);
      // 现在错误已经被 apiClient 拦截器统一格式化了
      setErrorMsg(err.message || err.response?.data?.message || "请求失败，请检查网络或后端");
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-gray-50 px-4 py-12 sm:px-6 lg:px-8">
      <div className="w-full max-w-md space-y-8 bg-white p-8 shadow rounded-xl">
        <div className="text-center">
          <h2 className="text-3xl font-bold tracking-tight text-gray-900">
            {isRegister ? "创建新账户" : "登录 Nexus Agent"}
          </h2>
          <p className="mt-2 text-sm text-gray-600">
            {isRegister ? "已有账户？" : "还没有账户？"}
            <button
              onClick={() => {
                setIsRegister(!isRegister);
                setErrorMsg("");
              }}
              className="ml-1 font-medium text-emerald-600 hover:text-emerald-500"
            >
              {isRegister ? "直接登录" : "立即注册"}
            </button>
          </p>
        </div>

        <form className="mt-8 space-y-6" onSubmit={handleSubmit(onSubmit)}>
          <div className="-space-y-px rounded-md shadow-sm">
            {isRegister && (
              <div className="mb-4">
                <label className="block text-sm font-medium text-gray-700">用户名</label>
                <input
                  {...register("name")}
                  type="text"
                  className="relative block w-full rounded-md border-0 py-1.5 text-gray-900 ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:z-10 focus:ring-2 focus:ring-inset focus:ring-emerald-600 sm:text-sm sm:leading-6 px-3"
                  placeholder="Your Name"
                />
              </div>
            )}
            <div className="mb-4">
              <label className="block text-sm font-medium text-gray-700">邮箱</label>
              <input
                {...register("email", { required: "邮箱必填" })}
                type="email"
                className="relative block w-full rounded-md border-0 py-1.5 text-gray-900 ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:z-10 focus:ring-2 focus:ring-inset focus:ring-emerald-600 sm:text-sm sm:leading-6 px-3"
                placeholder="name@example.com"
              />
              {errors.email && (
                <p className="text-red-500 text-xs mt-1">{errors.email.message as string}</p>
              )}
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">密码</label>
              <input
                {...register("password", { required: "密码必填", minLength: { value: 6, message: "最少6位" } })}
                type="password"
                className="relative block w-full rounded-md border-0 py-1.5 text-gray-900 ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:z-10 focus:ring-2 focus:ring-inset focus:ring-emerald-600 sm:text-sm sm:leading-6 px-3"
                placeholder="Password"
              />
               {errors.password && (
                <p className="text-red-500 text-xs mt-1">{errors.password.message as string}</p>
              )}
            </div>
          </div>

          {errorMsg && (
            <div className="text-red-500 text-sm text-center bg-red-50 p-2 rounded">
              {errorMsg}
            </div>
          )}

          <div>
            <button
              type="submit"
              disabled={isLoading}
              className="group relative flex w-full justify-center rounded-md bg-emerald-600 px-3 py-2 text-sm font-semibold text-white hover:bg-emerald-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-600 disabled:opacity-50"
            >
              {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {isRegister ? "注册" : "登录"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
