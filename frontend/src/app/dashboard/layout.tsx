"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { 
  LayoutDashboard, 
  MessageSquare, 
  Bot, 
  Key, 
  Database, 
  Server,
  LogOut,
  User
} from "lucide-react";
import { clsx } from "clsx";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const router = useRouter();

  const handleLogout = () => {
    localStorage.removeItem("token");
    router.push("/login");
  };

  const navItems = [
    { name: "概览", href: "/dashboard", icon: LayoutDashboard },
    { name: "会话", href: "/dashboard/sessions", icon: MessageSquare },
    { name: "Agents", href: "/dashboard/agents", icon: Bot },
    { name: "知识库", href: "/dashboard/knowledge", icon: Database },
    { name: "MCP 服务", href: "/dashboard/mcp", icon: Server },
    { name: "API Keys", href: "/dashboard/apikeys", icon: Key },
  ];

  return (
    <div className="flex h-screen bg-gray-100">
      {/* Sidebar */}
      <div className="w-64 bg-white border-r border-gray-200 flex flex-col">
        <div className="h-16 flex items-center px-6 border-b border-gray-200">
          <span className="text-xl font-bold text-emerald-600">Nexus Agent</span>
        </div>

        <nav className="flex-1 p-4 space-y-1">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = pathname === item.href;
            return (
              <Link
                key={item.href}
                href={item.href}
                className={clsx(
                  "flex items-center px-4 py-2 text-sm font-medium rounded-md transition-colors",
                  isActive
                    ? "bg-emerald-50 text-emerald-700"
                    : "text-gray-700 hover:bg-gray-50 hover:text-gray-900"
                )}
              >
                <Icon className="mr-3 h-5 w-5" />
                {item.name}
              </Link>
            );
          })}
        </nav>

        <div className="p-4 border-t border-gray-200">
          <button
            onClick={handleLogout}
            className="flex w-full items-center px-4 py-2 text-sm font-medium text-red-600 rounded-md hover:bg-red-50 transition-colors"
          >
            <LogOut className="mr-3 h-5 w-5" />
            退出登录
          </button>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Top Header (Optional, for user info) */}
        <header className="h-16 bg-white border-b border-gray-200 flex items-center justify-end px-8">
           <div className="flex items-center text-sm text-gray-500">
              <User className="h-4 w-4 mr-2"/>
              <span>Administrator</span>
           </div>
        </header>

        {/* Scrollable Content Area */}
        <main className="flex-1 overflow-auto p-8">
          {children}
        </main>
      </div>
    </div>
  );
}

