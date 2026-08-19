import { apiFetch } from "./api-client";
import type { User, GroupSummary, Member, ExpenseListItem, ExpenseDetail, BalanceMember, Suggestion, Settlement, DashboardData } from "./api";

export async function getCurrentUser(): Promise<User> {
  const res = await apiFetch("/api/v1/me");
  if (!res.ok) {
    throw new Error("Failed to get current user");
  }
  const body = await res.json();
  return body.data.user;
}

export async function getGroups(): Promise<GroupSummary[]> {
  const res = await apiFetch("/api/v1/groups");
  if (!res.ok) {
    throw new Error("Failed to get groups");
  }
  const body = await res.json();
  return body.data.groups;
}

export async function getGroup(groupId: string): Promise<GroupSummary> {
  const res = await apiFetch(`/api/v1/groups/${groupId}`);
  if (!res.ok) {
    throw new Error("Failed to get group");
  }
  const body = await res.json();
  return body.data;
}

export async function getGroupMembers(groupId: string): Promise<Member[]> {
  const res = await apiFetch(`/api/v1/groups/${groupId}/members`);
  if (!res.ok) {
    throw new Error("Failed to get group members");
  }
  const body = await res.json();
  return body.data.members;
}

export async function getGroupExpenses(groupId: string, page = 1, limit = 20): Promise<{ expenses: ExpenseListItem[]; total: number; page: number; limit: number }> {
  const res = await apiFetch(`/api/v1/groups/${groupId}/expenses?page=${page}&limit=${limit}`);
  if (!res.ok) {
    throw new Error("Failed to get group expenses");
  }
  const body = await res.json();
  return body.data;
}

export async function getExpense(expenseId: string): Promise<ExpenseDetail> {
  const res = await apiFetch(`/api/v1/expenses/${expenseId}`);
  if (!res.ok) {
    throw new Error("Failed to get expense");
  }
  const body = await res.json();
  return body.data;
}

export async function getGroupBalances(groupId: string): Promise<BalanceMember[]> {
  const res = await apiFetch(`/api/v1/groups/${groupId}/balances`);
  if (!res.ok) {
    throw new Error("Failed to get group balances");
  }
  const body = await res.json();
  return body.data.members;
}

export async function getSettlementSuggestions(groupId: string): Promise<Suggestion[]> {
  const res = await apiFetch(`/api/v1/groups/${groupId}/settlement-suggestions`);
  if (!res.ok) {
    throw new Error("Failed to get settlement suggestions");
  }
  const body = await res.json();
  return body.data.settlements;
}

export async function getGroupSettlements(groupId: string): Promise<Settlement[]> {
  const res = await apiFetch(`/api/v1/groups/${groupId}/settlements`);
  if (!res.ok) {
    throw new Error("Failed to get group settlements");
  }
  const body = await res.json();
  return body.data.settlements;
}

export async function getDashboard(): Promise<DashboardData> {
  const res = await apiFetch("/api/v1/dashboard");
  if (!res.ok) {
    throw new Error("Failed to get dashboard");
  }
  const body = await res.json();
  return body.data;
}

export async function createGroup(data: { name: string; description?: string; currency: string }): Promise<GroupSummary> {
  const res = await apiFetch("/api/v1/groups", {
    method: "POST",
    body: JSON.stringify(data),
  });
  if (!res.ok) {
    throw new Error("Failed to create group");
  }
  const body = await res.json();
  return body.data;
}

export type CreateExpenseInput = {
  description: string;
  amount: string;
  currency: string;
  paidBy: string;
  category: string;
  expenseDate: string;
  splitType: string;
  note?: string;
  participants: string[];
};

export async function createExpense(groupId: string, data: CreateExpenseInput): Promise<ExpenseDetail> {
  const res = await apiFetch(`/api/v1/groups/${groupId}/expenses`, {
    method: "POST",
    body: JSON.stringify(data),
  });
  if (!res.ok) {
    throw new Error("Failed to create expense");
  }
  const body = await res.json();
  return body.data.expense;
}

export async function createSettlement(groupId: string, data: { payerId: string; receiverId: string; amount: string; settledAt?: string }): Promise<Settlement> {
  const res = await apiFetch(`/api/v1/groups/${groupId}/settlements`, {
    method: "POST",
    body: JSON.stringify(data),
  });
  if (!res.ok) {
    throw new Error("Failed to create settlement");
  }
  const body = await res.json();
  return body.data;
}

export async function deleteExpense(expenseId: string): Promise<void> {
  const res = await apiFetch(`/api/v1/expenses/${expenseId}`, {
    method: "DELETE",
  });
  if (!res.ok) {
    throw new Error("Failed to delete expense");
  }
}

export async function deleteGroup(groupId: string): Promise<void> {
  const res = await apiFetch(`/api/v1/groups/${groupId}`, {
    method: "DELETE",
  });
  if (!res.ok) {
    throw new Error("Failed to delete group");
  }
}
