import { act, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import Toast from "./toast";

const replace = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace }),
}));

describe("Toast", () => {
  beforeEach(() => {
    replace.mockClear();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders nothing without a success param", () => {
    const { container } = render(<Toast />);
    expect(container).toBeEmptyDOMElement();
  });

  it("renders the success message and dismisses on click", async () => {
    const user = userEvent.setup();
    render(<Toast success="expense-added" />);

    expect(screen.getByRole("status")).toHaveTextContent("Expense added successfully.");

    await user.click(screen.getByRole("button", { name: /dismiss/i }));
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
  });

  it("auto-dismisses after 4 seconds and cleans the query param", () => {
    vi.useFakeTimers();
    window.history.replaceState({}, "", "/groups?success=group-deleted");

    render(<Toast success="group-deleted" />);

    expect(screen.getByRole("status")).toHaveTextContent("Group deleted.");

    act(() => {
      vi.advanceTimersByTime(4000);
    });
    expect(screen.queryByRole("status")).not.toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(300);
    });
    expect(replace).toHaveBeenCalledWith("/groups", { scroll: false });
  });

  it("ignores unknown success values", () => {
    const { container } = render(<Toast success="nonsense" />);
    expect(container).toBeEmptyDOMElement();
  });
});
