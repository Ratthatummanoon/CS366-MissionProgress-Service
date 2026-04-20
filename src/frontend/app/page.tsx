"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useConfig } from "@/lib/config-context";

const PRESET_TEAMS = [
  "TEAM-ALPHA",
  "TEAM-BRAVO",
  "TEAM-CHARLIE",
  "TEAM-DELTA",
  "TEAM-ECHO",
];

export default function LoginPage() {
  const router = useRouter();
  const { config, setConfig, isReady } = useConfig();
  const [apiUrl, setApiUrl] = useState(config?.apiUrl || "");
  const [apiKey, setApiKey] = useState(config?.apiKey || "");
  const [teamId, setTeamId] = useState(config?.teamId || PRESET_TEAMS[0]);
  const [customTeam, setCustomTeam] = useState("");
  const [useCustom, setUseCustom] = useState(false);
  const [error, setError] = useState("");

  if (!isReady) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
      </div>
    );
  }

  if (config) {
    router.push("/dashboard");
    return null;
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    const finalTeam = useCustom ? customTeam.trim() : teamId;
    if (!apiUrl.trim()) return setError("กรุณากรอก API URL");
    if (!apiKey.trim()) return setError("กรุณากรอก API Key");
    if (!finalTeam) return setError("กรุณาเลือกหรือกรอกชื่อทีม");

    try {
      new URL(apiUrl.trim());
    } catch {
      return setError("API URL ไม่ถูกต้อง");
    }

    setConfig({
      apiUrl: apiUrl.trim(),
      apiKey: apiKey.trim(),
      teamId: finalTeam,
    });
    router.push("/dashboard");
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <span className="text-5xl">🚨</span>
          <h1 className="text-2xl font-bold mt-4 text-gray-900">
            MissionProgress
          </h1>
          <p className="text-gray-500 mt-1">Rescue Team Dashboard</p>
        </div>

        <form
          onSubmit={handleSubmit}
          className="bg-white rounded-xl shadow-sm border border-gray-200 p-6 space-y-5"
        >
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              API Gateway URL
            </label>
            <input
              type="url"
              value={apiUrl}
              onChange={(e) => setApiUrl(e.target.value)}
              placeholder="https://xxxxxxxx.execute-api.us-east-1.amazonaws.com/v1"
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              API Key
            </label>
            <input
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder="x-api-key value"
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Rescue Team
            </label>
            {!useCustom ? (
              <div className="space-y-2">
                <div className="grid grid-cols-2 gap-2">
                  {PRESET_TEAMS.map((t) => (
                    <button
                      type="button"
                      key={t}
                      onClick={() => setTeamId(t)}
                      className={`rounded-lg border px-3 py-2 text-sm font-medium transition ${
                        teamId === t
                          ? "border-blue-500 bg-blue-50 text-blue-700"
                          : "border-gray-200 bg-white text-gray-700 hover:bg-gray-50"
                      }`}
                    >
                      {t}
                    </button>
                  ))}
                </div>
                <button
                  type="button"
                  onClick={() => setUseCustom(true)}
                  className="text-xs text-blue-600 hover:underline"
                >
                  กรอกชื่อทีมเอง
                </button>
              </div>
            ) : (
              <div className="space-y-2">
                <input
                  type="text"
                  value={customTeam}
                  onChange={(e) => setCustomTeam(e.target.value)}
                  placeholder="เช่น TEAM-FOXTROT"
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
                />
                <button
                  type="button"
                  onClick={() => setUseCustom(false)}
                  className="text-xs text-blue-600 hover:underline"
                >
                  เลือกจากรายการ
                </button>
              </div>
            )}
          </div>

          {error && (
            <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded-lg px-3 py-2">
              {error}
            </div>
          )}

          <button
            type="submit"
            className="w-full bg-blue-600 text-white rounded-lg px-4 py-2.5 font-medium hover:bg-blue-700 transition"
          >
            เข้าสู่ระบบ
          </button>
        </form>
      </div>
    </div>
  );
}
