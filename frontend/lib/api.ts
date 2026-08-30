export const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export type User = {
  id: string;
  name: string;
  email: string;
};

export type GroupSummary = {
  id: string;
  name: string;
  description: string | null;
  currency: string;
  role: "admin" | "member";
  memberCount: number;
  hasLogo: boolean;
  createdAt: string;
};

export type Member = {
  id: string;
  name: string;
  email: string;
  role: "admin" | "member";
  joinedAt: string;
};

export type ExpenseListItem = {
  id: string;
  groupId: string;
  description: string;
  amount: string;
  currency: string;
  paidBy: string;
  payerName: string;
  createdBy: string;
  category: string;
  expenseDate: string;
  participantCount: number;
  hasReceipt: boolean;
  createdAt: string;
};

export type ExpenseDetail = ExpenseListItem & {
  note: string | null;
  participants: { userId: string; amount: string }[];
  updatedAt: string;
};

export type BalanceMember = {
  userId: string;
  name: string;
  balance: string;
};

export type Suggestion = {
  fromUserId: string;
  toUserId: string;
  amount: string;
};

export type Settlement = {
  id: string;
  payerId: string;
  payerName: string;
  receiverId: string;
  receiverName: string;
  amount: string;
  settledAt: string;
  createdAt: string;
};

export type InviteLink = {
  token: string;
  expiresAt: string | null;
  url: string;
};

export type GroupPreview = {
  id: string;
  name: string;
  description: string | null;
  currency: string;
  memberCount: number;
  hasLogo: boolean;
  createdAt: string;
};

export type DashboardData = {
  summary: {
    owedToUser: string;
    userOwes: string;
    netBalance: string;
    totalExpense: string;
    settledAmount: string;
  };
  groups: {
    id: string;
    name: string;
    currency: string;
    memberCount: number;
    hasLogo: boolean;
    balance: string;
  }[];
  recentExpenses: {
    id: string;
    groupId: string;
    groupName: string;
    description: string;
    payerName: string;
    amount: string;
    category: string;
    expenseDate: string;
    participantCount: number;
  }[];
  categories: { category: string; total: string }[];
};

export const EXPENSE_CATEGORIES = [
  "Accommodation",
  "Food & Drinks",
  "Transportation",
  "Shopping",
  "Entertainment",
  "Utilities",
  "Other",
] as const;
