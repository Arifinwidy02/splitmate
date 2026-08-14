import { render, screen } from "@testing-library/react";
import userEvent, { type UserEvent } from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import AddExpenseForm from "./add-expense-form";

const { createExpense } = vi.hoisted(() => ({ createExpense: vi.fn() }));

vi.mock("@/app/actions/expenses", () => ({
  createExpense,
}));

const members = [
  { id: "u-me", name: "Arifin", email: "arifin@test.com", role: "admin" as const, joinedAt: "" },
  { id: "u-ani", name: "Ani", email: "ani@test.com", role: "member" as const, joinedAt: "" },
  { id: "u-budi", name: "Budi", email: "budi@test.com", role: "member" as const, joinedAt: "" },
];

const user = { id: "u-me", name: "Arifin", email: "arifin@test.com" };

function fillExpense(userEventInstance: UserEvent) {
  return {
    async submit() {
      await userEventInstance.click(screen.getByRole("button", { name: "Add Expense" }));
    },
  };
}

describe("AddExpenseForm", () => {
  it("defaults to equal split and the current user as payer/participant", () => {
    render(<AddExpenseForm groupId="g-1" currency="IDR" members={members} user={user} />);

    expect(screen.getByLabelText("Paid by")).toHaveValue("u-me");
    expect(screen.getByRole("button", { name: "Split equally" })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(screen.getByLabelText(/Arifin/)).toBeChecked();
    expect(screen.getByLabelText(/Ani/)).not.toBeChecked();
  });

  it("submits the expense with amount, participants and hidden fields", async () => {
    const userEventInstance = userEvent.setup();
    render(<AddExpenseForm groupId="g-1" currency="IDR" members={members} user={user} />);

    await userEventInstance.type(screen.getByLabelText("Description"), "Dinner");
    await userEventInstance.type(screen.getByLabelText("Amount"), "150000");
    await userEventInstance.click(screen.getByLabelText("Ani"));

    const form = fillExpense(userEventInstance);
    await form.submit();

    expect(createExpense).toHaveBeenCalledTimes(1);
    const [, , formData] = createExpense.mock.calls[0];
    expect(formData.get("description")).toBe("Dinner");
    expect(formData.get("amount")).toBe("150000");
    expect(formData.get("currency")).toBe("IDR");
    expect(formData.get("splitType")).toBe("equal");
    expect(formData.getAll("participant")).toEqual(["u-me", "u-ani"]);
    expect(formData.get("expenseDate")).toBeTruthy();
  });

  it("collects custom share amounts per selected participant", async () => {
    const userEventInstance = userEvent.setup();
    render(<AddExpenseForm groupId="g-1" currency="IDR" members={members} user={user} />);

    await userEventInstance.click(screen.getByRole("button", { name: "Custom amounts" }));
    await userEventInstance.type(screen.getByLabelText("Description"), "Dinner");
    await userEventInstance.type(screen.getByLabelText("Amount"), "150000");
    await userEventInstance.click(screen.getByLabelText("Ani"));
    await userEventInstance.type(screen.getByLabelText("Share for Arifin"), "100000");
    await userEventInstance.type(screen.getByLabelText("Share for Ani"), "50000");

    fillExpense(userEventInstance);
    await userEventInstance.click(screen.getByRole("button", { name: "Add Expense" }));

    expect(createExpense).toHaveBeenCalledTimes(1);
    const [, , formData] = createExpense.mock.calls[0];
    expect(formData.get("splitType")).toBe("custom");
    expect(formData.get("split-u-me")).toBe("100000");
    expect(formData.get("split-u-ani")).toBe("50000");
    expect(formData.get("split-u-budi")).toBeNull();
  });

  it("shows server validation errors", async () => {
    createExpense.mockImplementationOnce(async () => ({ error: "Splits do not add up" }));

    const userEventInstance = userEvent.setup();
    render(<AddExpenseForm groupId="g-1" currency="IDR" members={members} user={user} />);

    await userEventInstance.type(screen.getByLabelText("Description"), "Dinner");
    await userEventInstance.type(screen.getByLabelText("Amount"), "150000");
    await userEventInstance.click(screen.getByRole("button", { name: "Add Expense" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Splits do not add up");
  });
});
