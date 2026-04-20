"use client";

import { useConfig } from "@/lib/config-context";
import { useRouter } from "next/navigation";

export default function Navbar() {
  const { config, logout } = useConfig();
  const router = useRouter();

  if (!config) return null;

  return (
    <nav className="bg-white border-b border-gray-200 px-4 py-3 shadow-sm">
      <div className="max-w-7xl mx-auto flex items-center justify-between">
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
          <div className="text-sm text-gray-600">
            <span className="font-medium text-gray-900">{config.teamId}</span>
          </div>
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
    </nav>
  );
}
