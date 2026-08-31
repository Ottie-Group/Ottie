// Typed API Client for Ottie Go Backend

export interface ApiResponse<T = any> {
  success?: boolean;
  error?: string;
  data?: T;
}

class ApiClient {
  private csrfToken: string = '';

  setCsrfToken(token: string) {
    this.csrfToken = token;
  }

  getCsrfToken() {
    return this.csrfToken;
  }

  async request<T = any>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const headers = new Headers(options.headers || {});
    headers.set('Accept', 'application/json');

    if (this.csrfToken && options.method && options.method !== 'GET' && options.method !== 'HEAD') {
      headers.set('X-CSRF-Token', this.csrfToken);
    }

    if (options.body && typeof options.body === 'string' && !headers.has('Content-Type')) {
      headers.set('Content-Type', 'application/json');
    }

    const response = await fetch(endpoint, {
      ...options,
      headers,
      credentials: 'same-origin',
    });

    const csrfHeader = response.headers.get('X-CSRF-Token');
    if (csrfHeader) {
      this.csrfToken = csrfHeader;
    }

    if (response.status === 204) {
      return {} as T;
    }

    const contentType = response.headers.get('Content-Type') || '';
    if (!contentType.includes('application/json')) {
      if (!response.ok) {
        throw new Error(`Server returned error status ${response.status}`);
      }
      return (await response.text()) as unknown as T;
    }

    const json = await response.json();
    if (!response.ok) {
      throw new Error(json.error || `Request failed with status ${response.status}`);
    }

    return json;
  }

  get<T = any>(endpoint: string) {
    return this.request<T>(endpoint, { method: 'GET' });
  }

  post<T = any>(endpoint: string, body?: any) {
    return this.request<T>(endpoint, {
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
    });
  }
}

export const api = new ApiClient();
