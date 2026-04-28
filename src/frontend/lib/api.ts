import {
  MissionDetailResponse,
  ListMissionsResponse,
  ProgressResponse,
  PresignedURLResponse,
  ViewURLResponse,
  APIError,
} from "./types";

class ApiClient {
  private baseUrl: string;
  private apiKey: string;
  private teamId: string;

  constructor(baseUrl: string, apiKey: string, teamId: string) {
    this.baseUrl = baseUrl.replace(/\/+$/, "");
    this.apiKey = apiKey;
    this.teamId = teamId;
  }

  private headers(json = false): Record<string, string> {
    const h: Record<string, string> = {
      "x-api-key": this.apiKey,
      "X-Rescue-Team-ID": this.teamId,
    };
    if (json) h["Content-Type"] = "application/json";
    return h;
  }

  private async request<T>(path: string, init?: RequestInit): Promise<T> {
    const res = await fetch(`${this.baseUrl}${path}`, init);
    const body = await res.json();
    if (!res.ok) {
      const err = body as APIError;
      throw new Error(err.message || `Request failed: ${res.status}`);
    }
    return body as T;
  }

  async getMission(requestId: string): Promise<MissionDetailResponse> {
    return this.request<MissionDetailResponse>(
      `/missions/${encodeURIComponent(requestId)}`,
      { headers: this.headers() },
    );
  }

  async listMissions(status?: string): Promise<ListMissionsResponse> {
    const params = status ? `?status=${encodeURIComponent(status)}` : "";
    return this.request<ListMissionsResponse>(`/missions${params}`, {
      headers: this.headers(),
    });
  }

  async reportProgress(
    requestId: string,
    body: {
      new_status: string;
      note?: string;
      current_location?: string;
      new_impact_level?: number;
      image_key?: string;
    },
  ): Promise<ProgressResponse> {
    return this.request<ProgressResponse>(
      `/missions/${encodeURIComponent(requestId)}/progress`,
      {
        method: "POST",
        headers: this.headers(true),
        body: JSON.stringify(body),
      },
    );
  }

  async getPresignedUrl(
    requestId: string,
    fileName: string,
    contentType: string,
  ): Promise<PresignedURLResponse> {
    return this.request<PresignedURLResponse>(
      `/missions/${encodeURIComponent(requestId)}/presigned-url`,
      {
        method: "POST",
        headers: this.headers(true),
        body: JSON.stringify({
          file_name: fileName,
          content_type: contentType,
        }),
      },
    );
  }

  async getViewUrl(
    requestId: string,
    imageKey: string,
  ): Promise<ViewURLResponse> {
    return this.request<ViewURLResponse>(
      `/missions/${encodeURIComponent(requestId)}/presigned-url?image_key=${encodeURIComponent(imageKey)}`,
      { headers: this.headers() },
    );
  }

  async uploadFile(uploadUrl: string, file: File): Promise<void> {
    const res = await fetch(uploadUrl, {
      method: "PUT",
      headers: { "Content-Type": file.type },
      body: file,
    });
    if (!res.ok) {
      throw new Error(`Upload failed: ${res.status}`);
    }
  }
}

let clientInstance: ApiClient | null = null;

export function initClient(
  baseUrl: string,
  apiKey: string,
  teamId: string,
): ApiClient {
  clientInstance = new ApiClient(baseUrl, apiKey, teamId);
  return clientInstance;
}

export function getClient(): ApiClient | null {
  return clientInstance;
}

export function clearClient(): void {
  clientInstance = null;
}
