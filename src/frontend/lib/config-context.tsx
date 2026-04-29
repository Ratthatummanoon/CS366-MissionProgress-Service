"use client";

import {
  createContext,
  useContext,
  useState,
  useEffect,
  ReactNode,
} from "react";
import { initClient, getClient, clearClient } from "@/lib/api";

interface Config {
  apiUrl: string;
  apiKey: string;
  teamId?: string;
  teamName?: string;
  rescueTeamUrl?: string;
  manageDispatchUrl?: string;
  rescueRequestUrl?: string;
}

interface ConfigContextType {
  config: Config | null;
  setConfig: (config: Config) => void;
  logout: () => void;
  isReady: boolean;
}

const ConfigContext = createContext<ConfigContextType>({
  config: null,
  setConfig: () => {},
  logout: () => {},
  isReady: false,
});

const STORAGE_KEY = "mission-progress-config";

export function ConfigProvider({ children }: { children: ReactNode }) {
  const [config, setConfigState] = useState<Config | null>(null);
  const [isReady, setIsReady] = useState(false);

  useEffect(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) {
      try {
        const parsed = JSON.parse(saved) as Config;
        if (parsed.apiUrl && parsed.apiKey) {
          setConfigState(parsed);
          if (parsed.teamId) {
            initClient(parsed.apiUrl, parsed.apiKey, parsed.teamId);
          }
        }
      } catch {
        localStorage.removeItem(STORAGE_KEY);
      }
    }
    setIsReady(true);
  }, []);

  const setConfig = (c: Config) => {
    setConfigState(c);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(c));
    if (c.teamId) {
      initClient(c.apiUrl, c.apiKey, c.teamId);
    } else {
      clearClient();
    }
  };

  const logout = () => {
    setConfigState(null);
    localStorage.removeItem(STORAGE_KEY);
    clearClient();
  };

  return (
    <ConfigContext.Provider value={{ config, setConfig, logout, isReady }}>
      {children}
    </ConfigContext.Provider>
  );
}

export function useConfig() {
  return useContext(ConfigContext);
}

export { getClient };
