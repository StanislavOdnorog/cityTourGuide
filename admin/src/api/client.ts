import axios from 'axios';
import { API_BASE_URL } from '../constants';
import { useAuthStore } from '../store/authStore';
import type { LoginResponse, TokenPair } from '../types';

const apiClient = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor: attach JWT token
apiClient.interceptors.request.use((config) => {
  const { token } = useAuthStore.getState();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Response interceptor: handle 401 with token refresh
apiClient.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;

    if (
      error.response?.status === 401 &&
      !originalRequest._retry &&
      originalRequest.url !== '/api/v1/auth/login' &&
      originalRequest.url !== '/api/v1/auth/refresh'
    ) {
      originalRequest._retry = true;

      const { refreshToken, logout } = useAuthStore.getState();
      if (!refreshToken) {
        logout();
        return Promise.reject(error);
      }

      try {
        const { data } = await axios.post<{ tokens: TokenPair }>(
          `${API_BASE_URL}/api/v1/auth/refresh`,
          { refresh_token: refreshToken },
        );

        const { token: currentToken } = useAuthStore.getState();
        if (currentToken) {
          // Re-fetch user info is not needed; just update tokens
          useAuthStore.setState({
            token: data.tokens.access_token,
            refreshToken: data.tokens.refresh_token,
          });
        }

        originalRequest.headers.Authorization = `Bearer ${data.tokens.access_token}`;
        return apiClient(originalRequest);
      } catch {
        logout();
        return Promise.reject(error);
      }
    }

    return Promise.reject(error);
  },
);

export async function login(email: string, password: string): Promise<LoginResponse> {
  const { data } = await apiClient.post<LoginResponse>('/api/v1/auth/login', {
    email,
    password,
  });
  return data;
}

export default apiClient;
