"use client";

import { useConfig } from "@/lib/config-context";
import { useRouter } from "next/navigation";
import ServiceStatusBar from "@/components/ServiceStatusBar";

export default function Navbar() {
  const { config, logout } = useConfig();
  const router = useRouter();

  if (!config) return null;

  return (
    <nav className="bg-white border-b border-gray-200 shadow-sm">
      <div className="max-w-7xl mx-auto px-4 py-3 flex items-center justify-between">
        <button
          onClick={() => router.push("/dashboard")}
          className="flex items-center gap-2 hover:opacity-80 transition"
        >
          <span className="text-2xl">🚨</span>
          <span className="font-bold text-lg text-gray-900">
            MissionProgress
          </span>
        </button>

        <div className="flex items-center gap-4">
          {config.teamId && (
            <div className="text-sm text-right">
              {config.teamName && config.teamName !== config.teamId && (
                <div className="font-medium text-gray-900 leading-tight">
                  {config.teamName}
                </div>
              )}
              <div className="font-mono text-xs text-gray-400">
                {config.teamId}
              </div>
            </div>
          )}
          <button
            onClick={() => router.push("/teams")}
            className="text-sm text-blue-600 hover:text-blue-800 font-medium transition"
          >
            เปลี่ยนทีม
          </button>
          <button
            onClick={() => {
              logout();
              router.push("/");
            }}
            className="text-sm text-red-600 hover:text-red-800 font-medium transition"
          >
            ออกจากระบบ
          </button>
        </div>
      </div>

      {/* Services status bar */}
      <div className="max-w-7xl mx-auto px-4 pb-2">
        <ServiceStatusBar apiUrl={config.apiUrl} />
      </div>
    </nav>
  );
}
