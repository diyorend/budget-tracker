import api from "./client";

export interface LoginResponse {
  token: string;
  user: { id: string; email: string };
}

export const register = (email: string, password: string) =>
  api.post<LoginResponse>("/api/auth/register", { email, password });

export const login = (email: string, password: string) =>
  api.post<LoginResponse>("/api/auth/login", { email, password });
