import { render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import Toast from "./toast";
import { en } from "@/lib/i18n/en";

const { replace, toastSuccess } = vi.hoisted(() => ({
  replace: vi.fn(),
  toastSuccess: vi.fn(),
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace }),
}));

vi.mock("sonner", () => ({
  toast: { success: toastSuccess },
}));

describe("Toast", () => {
  beforeEach(() => {
    replace.mockClear();
    toastSuccess.mockClear();
    window.history.replaceState({}, "", "/groups?success=group-deleted");
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("renders nothing without a success param", () => {
    const { container } = render(<Toast dict={en} />);
    expect(container).toBeEmptyDOMElement();
    expect(toastSuccess).not.toHaveBeenCalled();
  });

  it("shows a success toast and cleans the query param", () => {
    render(<Toast success="group-deleted" dict={en} />);

    expect(toastSuccess).toHaveBeenCalledWith("Group deleted.");
    expect(replace).toHaveBeenCalledWith("/groups", { scroll: false });
  });

  it("does not show the same toast twice", () => {
    const { rerender } = render(<Toast success="group-deleted" dict={en} />);
    rerender(<Toast success="group-deleted" dict={en} />);

    expect(toastSuccess).toHaveBeenCalledTimes(1);
  });

  it("ignores unknown success values", () => {
    render(<Toast success="nonsense" dict={en} />);
    expect(toastSuccess).not.toHaveBeenCalled();
  });
});