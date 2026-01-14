"use client";

import useSWR from "swr";
import { apiClient } from "@/lib/apiClient";
import { fetcher } from "@/lib/fetcher";
import Link from "next/link";
import { Plus, MessageSquare, Trash2, Bot, Clock, X, Loader2 } from "lucide-react";
import { useState } from "react";
import { clsx } from "clsx";
import type { ApiResponse, ChatSession, Agent } from "@/types";

export default function SessionsListPage() {
  const { data: sessions, error, isLoading, mutate } = useSWR<ChatSession[]>(
    "/api/sessions",
    fetcher,
    {
      revalidateOnFocus: false,
    }
  );
  const [isDeleting, setIsDeleting] = useState<string | null>(null);
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);

  // Fetch agents for creating new session
  const { data: agents, error: agentsError } = useSWR<Agent[]>(
    "/api/agents",
    fetcher,
    {
      revalidateOnFocus: false,
    }
  );

  const handleDelete = async (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    if (!confirm("确定要删除这个会话吗？")) return;
    setIsDeleting(id);
    try {
      await apiClient.delete(`/api/sessions/${id}`);
      mutate(); // 刷新列表
    } catch (err) {
      alert("删除失败");
    } finally {
      setIsDeleting(null);
    }
  };

  const handleOpenCreateModal = () => {
    setIsCreateModalOpen(true);
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));
    
    if (days === 0) {
      return "今天";
    } else if (days === 1) {
      return "昨天";
    } else if (days < 7) {
      return `${days} 天前`;
    } else {
      return date.toLocaleDateString("zh-CN");
    }
  };

  if (error) return <div className="p-4 text-red-500">加载失败: {error.message}</div>;
  if (isLoading) return <div className="p-8 text-center text-gray-500">加载中...</div>;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">会话</h1>
          <p className="text-sm text-gray-500 mt-1">管理和查看您的对话历史</p>
        </div>
        <div className="flex items-center space-x-3">
          <button
            onClick={handleOpenCreateModal}
            className="inline-flex items-center px-4 py-2 bg-emerald-600 text-white text-sm font-medium rounded-md hover:bg-emerald-500 transition-colors shadow-sm"
          >
            <Plus className="h-4 w-4 mr-2" />
            新建会话
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {sessions?.map((session) => (
          <Link
            key={session.id}
            href={`/sessions/${session.id}`}
            className="group relative bg-white border border-gray-200 rounded-xl p-6 hover:shadow-md transition-all hover:border-emerald-300 cursor-pointer"
          >
            <div className="flex items-start justify-between">
              <div className="flex items-center space-x-3 flex-1 min-w-0">
                <div className="h-10 w-10 bg-emerald-100 rounded-lg flex items-center justify-center text-emerald-600 flex-shrink-0">
                  <MessageSquare className="h-6 w-6" />
                </div>
                <div className="flex-1 min-w-0">
                  <h3 className="text-base font-semibold text-gray-900 truncate">
                    {session.title}
                  </h3>
                  <div className="flex items-center space-x-2 mt-1">
                    <Bot className="h-3 w-3 text-gray-400" />
                    <span className="text-xs text-gray-500 truncate">
                      Agent: {session.agent_id.slice(0, 8)}...
                    </span>
                  </div>
                </div>
              </div>
              <button
                onClick={(e) => handleDelete(session.id, e)}
                disabled={isDeleting === session.id}
                className="opacity-0 group-hover:opacity-100 p-1.5 text-gray-400 hover:text-red-600 rounded-md hover:bg-red-50 transition-all flex-shrink-0"
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </div>

            <div className="mt-4 pt-4 border-t border-gray-100 flex items-center justify-between text-xs text-gray-400">
              <div className="flex items-center space-x-1">
                <Clock className="h-3 w-3" />
                <span>{formatDate(session.updated_at)}</span>
              </div>
              <span className="text-emerald-600 group-hover:text-emerald-700">
                查看 →
              </span>
            </div>
          </Link>
        ))}

        {sessions?.length === 0 && (
          <div className="col-span-full py-12 text-center bg-gray-50 rounded-xl border border-dashed border-gray-300">
            <MessageSquare className="h-12 w-12 text-gray-300 mx-auto mb-3" />
            <h3 className="text-sm font-medium text-gray-900">还没有会话</h3>
            <p className="text-sm text-gray-500 mt-1">
              {agents && agents.length > 0
                ? "点击右上角按钮创建新会话"
                : "请先创建一个 Agent，然后才能创建会话"}
            </p>
          </div>
        )}
      </div>

      {/* Create Session Modal */}
      {isCreateModalOpen && (
        <CreateSessionModal
          agents={agents || []}
          agentsError={agentsError}
          onClose={() => setIsCreateModalOpen(false)}
          onSuccess={(sessionId) => {
            mutate();
            setIsCreateModalOpen(false);
            window.location.href = `/sessions/${sessionId}`;
          }}
        />
      )}
    </div>
  );
}

// Create Session Modal Component
interface CreateSessionModalProps {
  agents: Agent[];
  agentsError: Error | undefined;
  onClose: () => void;
  onSuccess: (sessionId: string) => void;
}

function CreateSessionModal({ agents, agentsError, onClose, onSuccess }: CreateSessionModalProps) {
  const [selectedAgentId, setSelectedAgentId] = useState<string>("");
  const [isCreating, setIsCreating] = useState(false);
  const [error, setError] = useState<string>("");

  const handleCreate = async () => {
    if (!selectedAgentId) {
      setError("请选择一个 Agent");
      return;
    }

    setIsCreating(true);
    setError("");
    try {
      const res = await apiClient.post<ChatSession>(`/api/sessions`, {
        agent_id: selectedAgentId,
      });
      
      if (res.code === 0 && res.data) {
        onSuccess(res.data.id);
      } else {
        setError(res.message || "创建会话失败");
      }
    } catch (err: any) {
      console.error("Create session error:", err);
      const errorMessage = err?.message || err?.response?.data?.message || "创建会话失败";
      setError(errorMessage);
    } finally {
      setIsCreating(false);
    }
  };

  return (
    <div 
      className="fixed inset-0 z-50 flex items-center justify-center bg-white"
      onClick={onClose}
    >
      <div
        className="bg-white rounded-xl shadow-xl w-full max-w-md mx-4"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200">
          <h2 className="text-lg font-semibold text-gray-900">创建新会话</h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Content */}
        <div className="px-6 py-4">
          {agentsError ? (
            <div className="text-sm text-red-500 py-4">
              加载 Agent 失败，请刷新页面重试
            </div>
          ) : agents.length === 0 ? (
            <div className="text-sm text-gray-500 py-4 text-center">
              <Bot className="h-12 w-12 text-gray-300 mx-auto mb-3" />
              <p>暂无可用 Agent</p>
              <p className="mt-2">请先前往 Agents 页面创建 Agent</p>
            </div>
          ) : (
            <>
              <div className="mb-4">
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  选择 Agent
                </label>
                <select
                  value={selectedAgentId}
                  onChange={(e) => {
                    setSelectedAgentId(e.target.value);
                    setError("");
                  }}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm bg-white focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500"
                  style={{
                    color: selectedAgentId ? '#059669' : '#111827'
                  }}
                >
                  <option value="" style={{ color: '#111827' }}>请选择...</option>
                  {agents.map((agent) => (
                    <option key={agent.id} value={agent.id} style={{ color: '#111827' }}>
                      {agent.name} {agent.description ? `- ${agent.description}` : ""}
                    </option>
                  ))}
                </select>
              </div>

              {error && (
                <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-md text-sm text-red-600">
                  {error}
                </div>
              )}

              {/* Agent Info Preview */}
              {selectedAgentId && (
                <div className="mb-4 p-4 bg-gray-50 rounded-lg border border-gray-200">
                  {(() => {
                    const agent = agents.find(a => a.id === selectedAgentId);
                    if (!agent) return null;
                    return (
                      <div className="space-y-2">
                        <div className="flex items-center space-x-2">
                          <Bot className="h-4 w-4 text-emerald-600" />
                          <span className="text-sm font-medium text-gray-900">{agent.name}</span>
                        </div>
                        {agent.description && (
                          <p className="text-xs text-gray-600">{agent.description}</p>
                        )}
                        <div className="flex items-center space-x-4 text-xs text-gray-500">
                          <span>模型: {agent.model_name}</span>
                          {agent.temperature !== undefined && (
                            <span>温度: {agent.temperature}</span>
                          )}
                        </div>
                      </div>
                    );
                  })()}
                </div>
              )}
            </>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end space-x-3 px-6 py-4 border-t border-gray-200">
          <button
            onClick={onClose}
            disabled={isCreating}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 transition-colors disabled:opacity-50"
          >
            取消
          </button>
          <button
            onClick={handleCreate}
            disabled={!selectedAgentId || isCreating || agents.length === 0}
            className={clsx(
              "px-4 py-2 text-sm font-medium text-white rounded-md transition-colors",
              !selectedAgentId || isCreating || agents.length === 0
                ? "bg-gray-300 cursor-not-allowed"
                : "bg-emerald-600 hover:bg-emerald-500"
            )}
          >
            {isCreating ? (
              <span className="flex items-center">
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                创建中...
              </span>
            ) : (
              "创建"
            )}
          </button>
        </div>
      </div>
    </div>
  );
}

