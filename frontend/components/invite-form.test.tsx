import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import InviteForm from "./invite-form";
import { en } from "@/lib/i18n/en";

const { inviteMembers } = vi.hoisted(() => ({
  inviteMembers: vi.fn(),
}));

vi.mock("@/app/actions/groups", () => ({
  inviteMembers,
}));

describe("InviteForm", () => {
  it("submits emails split by commas and new lines", async () => {
    const user = userEvent.setup();
    render(<InviteForm groupId="g1" dict={en} />);

    await user.type(
      screen.getByLabelText("Emails"),
      "a@test.com, b@test.com\nc@test.com",
    );
    await user.click(screen.getByRole("button", { name: "Invite" }));

    expect(inviteMembers).toHaveBeenCalledTimes(1);
    const formData = inviteMembers.mock.calls[0][2];
    expect(formData.get("emails")).toBe("a@test.com, b@test.com\nc@test.com");
  });

  it("shows a server error", async () => {
    inviteMembers.mockImplementationOnce(async () => ({ error: "Enter at least one email address" }));

    const user = userEvent.setup();
    render(<InviteForm groupId="g1" dict={en} />);

    await user.type(screen.getByLabelText("Emails"), "x@test.com");
    await user.click(screen.getByRole("button", { name: "Invite" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Enter at least one email address",
    );
  });

  it("lists tokens and reports failures", async () => {
    inviteMembers.mockImplementationOnce(async () => ({
      invitations: [
        { email: "a@test.com", token: "tok-a" },
        { email: "b@test.com", token: "tok-b" },
      ],
      failed: [
        { email: "member@test.com", reason: "MEMBER_EXISTS" },
        { email: "dup@test.com", reason: "DUPLICATE" },
      ],
    }));

    const user = userEvent.setup();
    render(<InviteForm groupId="g1" dict={en} />);

    await user.type(screen.getByLabelText("Emails"), "a@test.com, b@test.com");
    await user.click(screen.getByRole("button", { name: "Invite" }));

    expect(await screen.findByText("Invitations created — share the tokens below.")).toBeTruthy();
    expect(screen.getByText("tok-a")).toBeTruthy();
    expect(screen.getByText("tok-b")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Copy all invitation tokens" })).toBeTruthy();
    expect(screen.getAllByRole("button", { name: "Copy invitation token" })).toHaveLength(2);

    const failures = screen.getByText("Some emails could not be invited:").closest("div");
    expect(within(failures!).getByText("member@test.com")).toBeTruthy();
    expect(within(failures!).getByText("Already a member")).toBeTruthy();
    expect(within(failures!).getByText("Duplicate in the list")).toBeTruthy();
  });
});