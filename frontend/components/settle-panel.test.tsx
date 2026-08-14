import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import SettlePanel from "./settle-panel";

const { createSettlement } = vi.hoisted(() => ({ createSettlement: vi.fn() }));

vi.mock("@/app/actions/settlements", () => ({
  createSettlement,
}));

const members = [
  { id: "u-me", name: "Arifin", email: "arifin@test.com", role: "admin" as const, joinedAt: "" },
  { id: "u-ani", name: "Ani", email: "ani@test.com", role: "member" as const, joinedAt: "" },
];

describe("SettlePanel", () => {
  it("shows quick settle suggestions owed by the current user", () => {
    render(
      <SettlePanel
        groupId="g-1"
        members={members}
        myUserId="u-me"
        suggestions={[
          { fromUserId: "u-me", toUserId: "u-ani", amount: "75000.00" },
          { fromUserId: "u-ani", toUserId: "u-me", amount: "5000.00" },
        ]}
      />,
    );

    expect(screen.getByText("You owe Ani")).toBeInTheDocument();
    expect(screen.getByText("Rp75.000")).toBeInTheDocument();
    expect(screen.getAllByText("Record payment")).toHaveLength(2);
  });

  it("submits the suggestion amount via the quick settle action", async () => {
    const userEventInstance = userEvent.setup();
    render(
      <SettlePanel
        groupId="g-1"
        members={members}
        myUserId="u-me"
        suggestions={[{ fromUserId: "u-me", toUserId: "u-ani", amount: "75000.00" }]}
      />,
    );

    const recordButtons = screen.getAllByRole("button", { name: "Record payment" });
    await userEventInstance.click(recordButtons[0]);

    expect(createSettlement).toHaveBeenCalledTimes(1);
    const [, , formData] = createSettlement.mock.calls[0];
    
    expect(formData.get("receiverId")).toBe("u-ani");
    expect(formData.get("amount")).toBe("75000.00");
  });

  it("only lists other members as settlement receivers", () => {
    render(
      <SettlePanel
        groupId="g-1"
        members={members}
        myUserId="u-me"
        suggestions={[]}
      />,
    );

    const options = screen
      .getAllByRole("option")
      .map((o) => (o as HTMLOptionElement).value);
    expect(options).toContain("u-ani");
    expect(options).not.toContain("u-me");
  });

  it("shows a server error on the manual settlement form", async () => {
    createSettlement.mockImplementationOnce(async () => ({ error: "Amount is invalid" }));

    const userEventInstance = userEvent.setup();
    render(
      <SettlePanel
        groupId="g-1"
        members={members}
        myUserId="u-me"
        suggestions={[]}
      />,
    );

    await userEventInstance.selectOptions(screen.getByLabelText("I paid back"), "u-ani");
    await userEventInstance.type(screen.getByLabelText("Amount"), "abc");
    await userEventInstance.click(screen.getByRole("button", { name: "Record payment" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Amount is invalid");
  });
});
