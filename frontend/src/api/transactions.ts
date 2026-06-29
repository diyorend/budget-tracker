import api from "./client";

export interface Transaction {
  id: string;
  amount: number;
  category: string;
  description: string;
  date: string;
  created_at: string;
}

export interface BudgetStatus {
  id: string;
  category: string;
  limit_amount: number;
  spent: number;
  remaining: number;
  percentage: number;
}

export const getTransactions = () =>
  api.get<Transaction[]>("/api/transactions");

export const createTransaction = (data: {
  amount: number;
  category: string;
  description?: string;
  date?: string;
}) => api.post<Transaction>("/api/transactions", data);

export const getBudgetStatus = (month?: string) =>
  api.get<BudgetStatus[]>("/api/budgets/status", { params: { month } });

export const upsertBudget = (data: {
  category: string;
  limit_amount: number;
  month?: string;
}) => api.post("/api/budgets", data);
