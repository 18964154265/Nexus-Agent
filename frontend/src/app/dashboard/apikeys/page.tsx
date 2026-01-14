"use client";

import { useState } from "react";
import useSWR from "swr";
import { apiClient } from "@/lib/apiClient";
import { fetcher } from "@/lib/fetcher";
import { Plus, Copy, Check, Trash2, AlertTriangle, X } from "lucide-react";
import { clsx } from "clsx";
import type { ApiResponse, APIKey } from "@/types";

// Create API Key Response type
interface CreateAPIKeyResp {
  id: string;
  name: string;
  prefix: string;
  key: string; // The RAW key (sk-nx-...), ONLY available here!
  created_at: string;
}

// Mock data for demonstration
const mockKeys: APIKey[] = [
  {
    id: "1",
    name: "My Laptop",
    prefix: "sk-nx-ab12cd34",
    last_used_at: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString(),
    created_at: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
  },
  {
    id: "2",
    name: "CI/CD Pipeline",
    prefix: "sk-nx-ef56gh78",
    last_used_at: new Date(Date.now() - 1 * 60 * 60 * 1000).toISOString(),
    created_at: new Date(Date.now() - 15 * 24 * 60 * 60 * 1000).toISOString(),
  },
  {
    id: "3",
    name: "Development Server",
    prefix: "sk-nx-ij90kl12",
    created_at: new Date(Date.now() - 5 * 24 * 60 * 60 * 1000).toISOString(),
  },
];

export default function APIKeysPage() {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [modalStep, setModalStep] = useState<"input" | "success">("input");
  const [keyName, setKeyName] = useState("");
  const [newKey, setNewKey] = useState<CreateAPIKeyResp | null>(null);
  const [isGenerating, setIsGenerating] = useState(false);
  const [copied, setCopied] = useState(false);

  // TODO: Replace with real API call
  // const { data: keys, error, isLoading, mutate } = useSWR<APIKey[]>("/api/api-keys", fetcher);
  const keys = mockKeys;
  const isLoading = false;

  const handleGenerate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!keyName.trim()) return;

    setIsGenerating(true);
    try {
      const res = await apiClient.post<CreateAPIKeyResp>("/api/api-keys", {
        name: keyName,
      });

      if (res.code === 0 && res.data) {
        setNewKey(res.data);
        setModalStep("success");
      } else {
        alert("生成失败：" + (res.message || "未知错误"));
      }
    } catch (error: any) {
      alert("生成失败：" + (error.message || "请重试"));
    } finally {
      setIsGenerating(false);
    }
  };

  const handleCopy = async () => {
    if (!newKey?.key) return;
    try {
      await navigator.clipboard.writeText(newKey.key);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (error) {
      alert("复制失败，请手动复制");
    }
  };

  const handleCloseModal = () => {
    setIsModalOpen(false);
    setModalStep("input");
    setKeyName("");
    setNewKey(null);
    setCopied(false);
    // TODO: mutate(); // Refresh the list
  };

  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`确定要撤销密钥 "${name}" 吗？此操作无法撤销，使用此密钥的应用将无法继续工作。`)) {
      return;
    }

    try {
      await apiClient.delete(`/api/api-keys/${id}`);
      alert("密钥已撤销");
      // TODO: mutate();
    } catch (error) {
      alert("撤销失败，请重试");
    }
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleDateString("zh-CN", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  };

  const formatLastUsed = (dateString?: string) => {
    if (!dateString) return "Never used";
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return "Just now";
    if (diffMins < 60) return `${diffMins} minutes ago`;
    if (diffHours < 24) return `${diffHours} hours ago`;
    if (diffDays < 7) return `${diffDays} days ago`;
    return formatDate(dateString);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">API Keys</h1>
          <p className="text-sm text-gray-500 mt-1">
            Manage access keys for your personal development environment (CLI, IDE, Scripts)
          </p>
        </div>
        <button
          onClick={() => setIsModalOpen(true)}
          className="inline-flex items-center px-4 py-2 bg-emerald-600 text-white text-sm font-medium rounded-md hover:bg-emerald-500 transition-colors"
        >
          <Plus className="h-4 w-4 mr-2" />
          Generate New Key
        </button>
      </div>

      {/* Table */}
      {isLoading ? (
        <div className="text-center py-12 text-gray-500">加载中...</div>
      ) : keys.length === 0 ? (
        <div className="text-center py-12 bg-gray-50 rounded-lg border border-dashed border-gray-300">
          <p className="text-sm text-gray-500">还没有 API Key，点击右上角按钮生成一个</p>
        </div>
      ) : (
        <div className="bg-white border border-gray-200 rounded-xl overflow-hidden">
          <table className="w-full">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Token</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status/Usage</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Created</th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {keys.map((key) => (
                <tr key={key.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className="text-sm font-medium text-gray-900">{key.name}</span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className="inline-flex items-center px-2.5 py-0.5 rounded-md text-xs font-mono bg-gray-100 text-gray-800">
                      {key.prefix}...
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className="text-sm text-gray-500">
                      {key.last_used_at ? `Last used: ${formatLastUsed(key.last_used_at)}` : "Never used"}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className="text-sm text-gray-500">{formatDate(key.created_at)}</span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right">
                    <button
                      onClick={() => handleDelete(key.id, key.name)}
                      className="inline-flex items-center px-3 py-1.5 text-sm font-medium text-red-600 hover:text-red-700 hover:bg-red-50 rounded-md transition-colors"
                    >
                      <Trash2 className="h-4 w-4 mr-1" />
                      Revoke
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Generate Key Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50" onClick={handleCloseModal}>
          <div className="bg-white rounded-xl shadow-xl w-full max-w-lg mx-4" onClick={(e) => e.stopPropagation()}>
            {modalStep === "input" ? (
              <>
                <div className="flex items-center justify-between p-6 border-b border-gray-200">
                  <h2 className="text-lg font-semibold text-gray-900">Generate New API Key</h2>
                  <button onClick={handleCloseModal} className="text-gray-400 hover:text-gray-600">
                    <X className="h-5 w-5" />
                  </button>
                </div>
                <form onSubmit={handleGenerate} className="p-6 space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
                    <input
                      type="text"
                      value={keyName}
                      onChange={(e) => setKeyName(e.target.value)}
                      placeholder="e.g., My Laptop"
                      required
                      className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500"
                    />
                    <p className="mt-1 text-xs text-gray-500">Give this key a descriptive name to identify where it's used.</p>
                  </div>
                  <div className="flex justify-end space-x-3 pt-4 border-t border-gray-200">
                    <button
                      type="button"
                      onClick={handleCloseModal}
                      className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
                    >
                      Cancel
                    </button>
                    <button
                      type="submit"
                      disabled={isGenerating || !keyName.trim()}
                      className="px-4 py-2 text-sm font-medium text-white bg-emerald-600 rounded-md hover:bg-emerald-500 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      {isGenerating ? "Generating..." : "Generate"}
                    </button>
                  </div>
                </form>
              </>
            ) : (
              <>
                <div className="flex items-center justify-between p-6 border-b border-gray-200">
                  <h2 className="text-lg font-semibold text-gray-900">API Key Generated</h2>
                  <button onClick={handleCloseModal} className="text-gray-400 hover:text-gray-600">
                    <X className="h-5 w-5" />
                  </button>
                </div>
                <div className="p-6 space-y-4">
                  {/* Warning Alert */}
                  <div className="flex items-start space-x-3 p-4 bg-amber-50 border border-amber-200 rounded-lg">
                    <AlertTriangle className="h-5 w-5 text-amber-600 flex-shrink-0 mt-0.5" />
                    <div className="flex-1">
                      <p className="text-sm font-medium text-amber-800">Important: Save this key now</p>
                      <p className="text-xs text-amber-700 mt-1">
                        For security reasons, we cannot show this key to you again. Please copy and store it in a safe place.
                      </p>
                    </div>
                  </div>

                  {/* Key Display */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Your API Key</label>
                    <div className="flex items-center space-x-2">
                      <input
                        type="text"
                        value={newKey?.key || ""}
                        readOnly
                        className="flex-1 px-3 py-2 font-mono text-sm bg-gray-50 border border-gray-300 rounded-md focus:outline-none"
                      />
                      <button
                        onClick={handleCopy}
                        className={clsx(
                          "inline-flex items-center px-4 py-2 text-sm font-medium rounded-md transition-colors",
                          copied
                            ? "bg-emerald-100 text-emerald-700"
                            : "bg-gray-100 text-gray-700 hover:bg-gray-200"
                        )}
                      >
                        {copied ? (
                          <>
                            <Check className="h-4 w-4 mr-2" />
                            Copied
                          </>
                        ) : (
                          <>
                            <Copy className="h-4 w-4 mr-2" />
                            Copy
                          </>
                        )}
                      </button>
                    </div>
                  </div>

                  <div className="flex justify-end pt-4 border-t border-gray-200">
                    <button
                      onClick={handleCloseModal}
                      className="px-4 py-2 text-sm font-medium text-white bg-emerald-600 rounded-md hover:bg-emerald-500"
                    >
                      Done
                    </button>
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

