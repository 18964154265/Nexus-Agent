"use client";

import useSWR from "swr";
import { apiClient } from "@/lib/apiClient";
import { fetcher } from "@/lib/fetcher";
import Link from "next/link";
import { Plus, Bot, Settings, Trash2 } from "lucide-react";
import { useState } from "react";
import type { ApiResponse, Agent } from "@/types";

export default function AgentsListPage() {
  const { data: agents, error, isLoading, mutate } = useSWR<Agent[]>("/api/agents", fetcher, {
    revalidateOnFocus: false,
  });
  const [isDeleting, setIsDeleting] = useState<string | null>(null);

  const handleDelete = async (id: string) => {
    if (!confirm("确定要删除这个 Agent 吗？")) return;
    setIsDeleting(id);
    try {
      await apiClient.delete(`/api/agents/${id}`);
      mutate(); // 刷新列表
    } catch (err) {
      alert("删除失败");
    } finally {
      setIsDeleting(null);
    }
  };

  if (error) return <div className="p-4 text-red-500">加载失败: {error.message}</div>;
  if (isLoading) return <div className="p-8 text-center text-gray-500">加载中...</div>;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Agents</h1>
          <p className="text-sm text-gray-500 mt-1">管理和配置您的智能体团队</p>
        </div>
        <Link
          href="/dashboard/agents/new"
          className="inline-flex items-center px-4 py-2 bg-emerald-600 text-white text-sm font-medium rounded-md hover:bg-emerald-500 transition-colors shadow-sm"
        >
          <Plus className="h-4 w-4 mr-2" />
          新建 Agent
        </Link>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {agents?.map((agent) => (
          <div
            key={agent.id}
            className="group relative bg-white border border-gray-200 rounded-xl p-6 hover:shadow-md transition-shadow"
          >
            <div className="flex items-start justify-between">
              <div className="flex items-center space-x-3">
                <div className="h-10 w-10 bg-emerald-100 rounded-lg flex items-center justify-center text-emerald-600">
                  <Bot className="h-6 w-6" />
                </div>
                <div>
                  <h3 className="text-base font-semibold text-gray-900">{agent.name}</h3>
                  <span className="text-xs text-gray-500 px-2 py-0.5 bg-gray-100 rounded-full">
                    {agent.model_name}
                  </span>
                </div>
              </div>
              <div className="flex space-x-1 opacity-0 group-hover:opacity-100 transition-opacity">
                {/* 暂时没有编辑页，先占位 */}
                <button className="p-1.5 text-gray-400 hover:text-emerald-600 rounded-md hover:bg-emerald-50">
                   <Settings className="h-4 w-4" />
                </button>
                <button 
                  onClick={() => handleDelete(agent.id)}
                  disabled={isDeleting === agent.id}
                  className="p-1.5 text-gray-400 hover:text-red-600 rounded-md hover:bg-red-50"
                >
                   <Trash2 className="h-4 w-4" />
                </button>
              </div>
            </div>
            
            <p className="mt-4 text-sm text-gray-600 line-clamp-2 min-h-[40px]">
              {agent.description || "暂无描述"}
            </p>

            <div className="mt-4 pt-4 border-t border-gray-100 flex items-center justify-between text-xs text-gray-400">
               <span>ID: {agent.id.slice(0, 8)}...</span>
               <span>{new Date(agent.created_at).toLocaleDateString()}</span>
            </div>
          </div>
        ))}

        {agents?.length === 0 && (
          <div className="col-span-full py-12 text-center bg-gray-50 rounded-xl border border-dashed border-gray-300">
            <Bot className="h-12 w-12 text-gray-300 mx-auto mb-3" />
            <h3 className="text-sm font-medium text-gray-900">还没有 Agent</h3>
            <p className="text-sm text-gray-500 mt-1">点击右上角按钮创建一个新的智能体</p>
          </div>
        )}
      </div>
    </div>
  );
}

