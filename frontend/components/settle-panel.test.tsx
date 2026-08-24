import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import SettlePanel from "./settle-panel";
import { en } from "@/lib/i18n/en";

const { createSettlement } = vi.hoisted(() => ({ createSettlement: vi.fn() }));

vi.mock("@/app/actions/settlements", () => ({
  createSettlement,
}));

const members = [
  { id: "u-me", name: "Arifin", email: "arifin@test.com", role: "admin" as const, joinedAt: "" },
  { id: "u-ani", name: "Ani", email: "ani@test.com", role: "member" as const, joinedAt: "" },
  { id: "u-budi", name: "Budi", email: "budi@test.com", role: "member" as const, joinedAt: "" },
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
        dict={en}
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
        dict={en}
      />,
    );

    const recordButtons = screen.getAllByRole("button", { name: "Record payment" });
    await userEventInstance.click(recordButtons[0]);

    expect(createSettlement).toHaveBeenCalledTimes(1);
    const [, , formData] = createSettlement.mock.calls[0];

    expect(formData.get("payerId")).toBe("u-me");
    expect(formData.get("receiverId")).toBe("u-ani");
    expect(formData.get("amount")).toBe("75000.00");
  });

  it("shows all suggestions to an admin with member names", () => {
    render(
      <SettlePanel
        groupId="g-1"
        members={members}
        myUserId="u-me"
        isAdmin
        suggestions={[
          { fromUserId: "u-ani", toUserId: "u-me", amount: "5000.00" },
          { fromUserId: "u-budi", toUserId: "u-ani", amount: "12000.00" },
        ]}
        dict={en}
      />,
    );

    expect(screen.getByText("Ani owes Arifin")).toBeInTheDocument();
    expect(screen.getByText("Budi owes Ani")).toBeInTheDocument();
  });

  it("submits the payer id when an admin records on behalf of a member", async () => {
    const userEventInstance = userEvent.setup();
    render(
      <SettlePanel
        groupId="g-1"
        members={members}
        myUserId="u-me"
        isAdmin
        suggestions={[{ fromUserId: "u-ani", toUserId: "u-me", amount: "5000.00" }]}
        dict={en}
      />,
    );

    const recordButtons = screen.getAllByRole("button", { name: "Record payment" });
    await userEventInstance.click(recordButtons[0]);

    const [, , formData] = createSettlement.mock.calls[0];
    expect(formData.get("payerId")).toBe("u-ani");
    expect(formData.get("receiverId")).toBe("u-me");
  });

  it("only lists other members as settlement receivers", () => {
    render(
      <SettlePanel
        groupId="g-1"
        members={members}
        myUserId="u-me"
        suggestions={[]}
        dict={en}
      />,
    );

    const receiverSelect = screen.getByLabelText("Received by");
    const options = within(receiverSelect)
      .getAllByRole("option")
      .map((o) => (o as HTMLOptionElement).value);
    expect(options).toContain("u-ani");
    expect(options).not.toContain("u-me");
  });

  it("lets an admin pick the payer in the manual form", () => {
    render(
      <SettlePanel
        groupId="g-1"
        members={members}
        myUserId="u-me"
        isAdmin
        suggestions={[]}
        dict={en}
      />,
    );

    const payerSelect = screen.getByLabelText("Paid by");
    const payerOptions = within(payerSelect)
      .getAllByRole("option")
      .map((o) => (o as HTMLOptionElement).value);
    expect(payerOptions).toContain("u-me");
    expect(payerOptions).toContain("u-ani");
    expect(payerOptions).toContain("u-budi");
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
        dict={en}
      />,
    );

    await userEventInstance.selectOptions(screen.getByLabelText("Received by"), "u-ani");
    await userEventInstance.type(screen.getByLabelText("Amount"), "100");
    await userEventInstance.click(screen.getByRole("button", { name: "Record payment" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Amount is invalid");
  });
});
