import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import DeleteExpenseButton from "./delete-expense-button";
import { en } from "@/lib/i18n/en";

const { deleteExpense } = vi.hoisted(() => ({ deleteExpense: vi.fn() }));

vi.mock("@/app/actions/expenses", () => ({
  deleteExpense,
}));

describe("DeleteExpenseButton", () => {
  it("opens a confirmation dialog and deletes on confirm", async () => {
    const user = userEvent.setup();
    render(
      <DeleteExpenseButton
        groupId="group-1"
        expenseId="expense-1"
        description="Makan bareng"
        dict={en}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Delete expense Makan bareng" }));
    expect(screen.getByRole("dialog")).toHaveTextContent("Delete this expense?");

    await user.click(screen.getByRole("button", { name: "Delete expense" }));
    expect(deleteExpense).toHaveBeenCalledWith("group-1", "expense-1");
  });

  it("does not call the action when cancelled", async () => {
    const user = userEvent.setup();
    render(
      <DeleteExpenseButton
        groupId="group-1"
        expenseId="expense-1"
        description="Makan bareng"
        dict={en}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Delete expense Makan bareng" }));
    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(deleteExpense).not.toHaveBeenCalled();
  });

  it("shows a server error instead of deleting", async () => {
    deleteExpense.mockImplementationOnce(async () => ({ error: "Forbidden" }));

    const user = userEvent.setup();
    render(
      <DeleteExpenseButton
        groupId="group-1"
        expenseId="expense-1"
        description="Makan bareng"
        dict={en}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Delete expense Makan bareng" }));
    await user.click(screen.getByRole("button", { name: "Delete expense" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Forbidden");
  });
});
