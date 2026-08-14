import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import ConfirmDialog from "./confirm-dialog";

describe("ConfirmDialog", () => {
  it("renders title and message when open", () => {
    render(
      <ConfirmDialog
        open
        title="Delete this expense?"
        message="This cannot be undone."
        confirmLabel="Delete expense"
        pending={false}
        onConfirm={vi.fn()}
        onClose={vi.fn()}
      />,
    );

    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("Delete this expense?")).toBeInTheDocument();
    expect(screen.getByText("This cannot be undone.")).toBeInTheDocument();
  });

  it("renders nothing when closed", () => {
    const { container } = render(
      <ConfirmDialog
        open={false}
        title="Delete this expense?"
        message="This cannot be undone."
        confirmLabel="Delete expense"
        pending={false}
        onConfirm={vi.fn()}
        onClose={vi.fn()}
      />,
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("calls onConfirm with the destructive button", async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();

    render(
      <ConfirmDialog
        open
        title="Delete this expense?"
        message="This cannot be undone."
        confirmLabel="Delete expense"
        pending={false}
        onConfirm={onConfirm}
        onClose={vi.fn()}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Delete expense" }));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it("calls onClose with Cancel", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();

    render(
      <ConfirmDialog
        open
        title="Delete this expense?"
        message="This cannot be undone."
        confirmLabel="Delete expense"
        pending={false}
        onConfirm={vi.fn()}
        onClose={onClose}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("disables both buttons while pending", () => {
    render(
      <ConfirmDialog
        open
        title="Delete this expense?"
        message="This cannot be undone."
        confirmLabel="Delete expense"
        pending
        onConfirm={vi.fn()}
        onClose={vi.fn()}
      />,
    );

    expect(screen.getByRole("button", { name: "Deleting..." })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeDisabled();
  });
});
