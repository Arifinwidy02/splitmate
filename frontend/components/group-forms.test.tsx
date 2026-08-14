import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import CreateGroupForm from "./create-group-form";
import JoinGroupForm from "./join-group-form";

const { createGroup, acceptInvitation } = vi.hoisted(() => ({
  createGroup: vi.fn(),
  acceptInvitation: vi.fn(),
}));

vi.mock("@/app/actions/groups", () => ({
  createGroup,
  acceptInvitation,
}));

describe("CreateGroupForm", () => {
  it("submits name and defaults to IDR", async () => {
    const user = userEvent.setup();
    render(<CreateGroupForm />);

    await user.type(screen.getByLabelText("Group name"), "Bali Trip");
    await user.click(screen.getByRole("button", { name: "Create group" }));

    expect(createGroup).toHaveBeenCalledTimes(1);
    const [, formData] = createGroup.mock.calls[0];
    expect(formData.get("name")).toBe("Bali Trip");
    expect(formData.get("currency")).toBe("IDR");
  });

  it("shows a server error", async () => {
    createGroup.mockImplementationOnce(async () => ({ error: "Name is required" }));

    const user = userEvent.setup();
    render(<CreateGroupForm />);

    await user.type(screen.getByLabelText("Group name"), "Trip");
    await user.click(screen.getByRole("button", { name: "Create group" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Name is required");
  });
});

describe("JoinGroupForm", () => {
  it("submits the invitation token", async () => {
    const user = userEvent.setup();
    render(<JoinGroupForm />);

    await user.type(screen.getByLabelText("Invitation token"), "tok-123");
    await user.click(screen.getByRole("button", { name: "Join group" }));

    expect(acceptInvitation).toHaveBeenCalledTimes(1);
    const [, formData] = acceptInvitation.mock.calls[0];
    expect(formData.get("token")).toBe("tok-123");
  });

  it("shows a server error", async () => {
    acceptInvitation.mockImplementationOnce(async () => ({ error: "Invalid token" }));

    const user = userEvent.setup();
    render(<JoinGroupForm />);

    await user.type(screen.getByLabelText("Invitation token"), "bad");
    await user.click(screen.getByRole("button", { name: "Join group" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Invalid token");
  });
});
