"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useConfig } from "@/lib/config-context";
import ServiceStatusBar from "@/components/ServiceStatusBar";

interface DiscoveredUrls {
  rescueTeamUrl: string;
  manageDispatchUrl: string;
  rescueRequestUrl: string;
}

export default function LoginPage() {
  const router = useRouter();
  const { config, setConfig, isReady } = useConfig();
  const [apiUrl, setApiUrl] = useState(config?.apiUrl || "");
  const [apiKey, setApiKey] = useState(config?.apiKey || "");
  const [error, setError] = useState("");

  // Auto-discovered dependency service URLs from /health
  const [discoveredUrls, setDiscoveredUrls] = useState<DiscoveredUrls>({
    rescueTeamUrl: config?.rescueTeamUrl || "",
    manageDispatchUrl: config?.manageDispatchUrl || "",
    rescueRequestUrl: config?.rescueRequestUrl || "",
  });
  const [discovering, setDiscovering] = useState(false);

  const fetchServiceConfig = useCallback(async (url: string) => {
    if (!url) return;
    try {
      new URL(url);
    } catch {
      return;
    }
    setDiscovering(true);
    try {
      const clean = url.replace(/\/+$/, "");
      const res = await fetch(`${clean}/health`, {
        signal: AbortSignal.timeout(5000),
      });
      if (res.ok) {
        const data = await res.json();
        const urls = data.service_urls ?? {};
        setDiscoveredUrls({
          rescueTeamUrl: urls.rescueTeamUrl || "",
          manageDispatchUrl: urls.manageDispatchUrl || "",
          rescueRequestUrl: urls.rescueRequestUrl || "",
        });
      }
    } catch {
      // ignore — ServiceStatusBar handles error display
    } finally {
      setDiscovering(false);
    }
  }, []);

  // Debounce: auto-discover dependency URLs when apiUrl changes
  useEffect(() => {
    const t = setTimeout(() => fetchServiceConfig(apiUrl), 600);
    return () => clearTimeout(t);
  }, [apiUrl, fetchServiceConfig]);

  // If already fully configured (has teamId), skip to dashboard
  useEffect(() => {
    if (isReady && config?.apiUrl && config?.apiKey && config?.teamId) {
      router.push("/dashboard");
    }
  }, [isReady, config, router]);

  if (!isReady) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
      </div>
    );
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    if (!apiUrl.trim()) return setError("กรุณากรอก API URL");
    if (!apiKey.trim()) return setError("กรุณากรอก API Key");
    try {
      new URL(apiUrl.trim());
    } catch {
      return setError("API URL ไม่ถูกต้อง");
    }

    setConfig({
      apiUrl: apiUrl.trim(),
      apiKey: apiKey.trim(),
      rescueTeamUrl: discoveredUrls.rescueTeamUrl || undefined,
      manageDispatchUrl: discoveredUrls.manageDispatchUrl || undefined,
      rescueRequestUrl: discoveredUrls.rescueRequestUrl || undefined,
    });
    router.push("/teams");
  };

  const discoveryRows = [
    { label: "RescueTeam", url: discoveredUrls.rescueTeamUrl },
    { label: "ManageDispatch", url: discoveredUrls.manageDispatchUrl },
    { label: "RescueRequest", url: discoveredUrls.rescueRequestUrl },
  ];

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
              placeholder="https://xxxxxxxx.execute-api.ap-southeast-1.amazonaws.com/v1"
              className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
              autoComplete="url"
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
              autoComplete="current-password"
            />
          </div>

          {/* Auto-discovered dependency services */}
          {apiUrl && (
            <div className="rounded-lg border border-gray-200 bg-gray-50 px-4 py-3 space-y-2">
              <div className="flex items-center justify-between">
                <p className="text-xs font-medium text-gray-600">
                  Dependency Services
                </p>
                {discovering && (
                  <span className="text-xs text-blue-500 animate-pulse">
                    ตรวจสอบ…
                  </span>
                )}
              </div>
              {discoveryRows.map(({ label, url }) => (
                <div key={label} className="flex items-center gap-2 text-xs">
                  <span
                    className={`w-1.5 h-1.5 rounded-full shrink-0 ${
                      url ? "bg-green-500" : "bg-gray-300"
                    }`}
                  />
                  <span className="text-gray-700 w-28 shrink-0">{label}</span>
                  {url ? (
                    <span className="text-gray-400 truncate" title={url}>
                      {(() => {
                        try {
                          return new URL(url).hostname;
                        } catch {
                          return url;
                        }
                      })()}
                    </span>
                  ) : (
                    <span className="text-gray-300 italic">not configured</span>
                  )}
                </div>
              ))}
            </div>
          )}

          {error && (
            <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded-lg px-3 py-2">
              {error}
            </div>
          )}

          <button
            type="submit"
            className="w-full bg-blue-600 text-white rounded-lg px-4 py-2.5 font-medium hover:bg-blue-700 transition"
          >
            เชื่อมต่อ →
          </button>
        </form>

        {/* Services status */}
        {apiUrl && (
          <div className="mt-4 bg-white rounded-xl border border-gray-200 px-4 py-3 shadow-sm">
            <p className="text-xs font-medium text-gray-500 mb-2">
              Services Status
            </p>
            <ServiceStatusBar apiUrl={apiUrl} />
          </div>
        )}
      </div>
    </div>
  );
}
