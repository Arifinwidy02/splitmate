import Link from "next/link";
import { Plus, Users } from "lucide-react";

import CreateGroupForm from "@/components/create-group-form";
import DeleteGroupButton from "@/components/delete-group-button";
import JoinGroupForm from "@/components/join-group-form";
import Toast from "@/components/toast";
import { getCurrentUser } from "@/lib/auth";
import { apiFetch } from "@/lib/server-api";
import { formatDate } from "@/lib/format";
import type { GroupSummary } from "@/lib/api";

export default async function GroupsPage({ searchParams }: PageProps<"/groups">) {
  const sp = await searchParams;
  const success = typeof sp.success === "string" ? sp.success : undefined;

  await getCurrentUser();
  const { groups } = await apiFetch<{ groups: GroupSummary[] }>("/api/v1/groups");

  return (
    <div className="mx-auto w-full max-w-[1440px]">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-[32px] font-bold text-slate-900">Groups</h1>
          <p className="mt-1 text-sm text-slate-500">
            Share expenses with friends, family, and teams.
          </p>
        </div>
        <Link
          href="#create-group"
          className="inline-flex items-center justify-center gap-2 rounded-lg bg-green-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition hover:bg-green-700"
        >
          <Plus className="h-4 w-4" aria-hidden="true" />
          New group
        </Link>
      </div>

      <div className="mt-6 grid grid-cols-1 gap-6 xl:grid-cols-3">
        <section
          aria-labelledby="group-list-heading"
          className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm xl:col-span-2"
        >
          <h2 id="group-list-heading" className="text-xl font-semibold text-slate-900">
            Your groups
          </h2>

          {groups.length === 0 ? (
            <div className="py-12 text-center">
              <Users className="mx-auto h-10 w-10 text-slate-300" aria-hidden="true" />
              <p className="mt-3 font-medium text-slate-700">No groups yet.</p>
              <p className="mt-1 text-sm text-slate-500">
                Create your first group to start sharing expenses.
              </p>
            </div>
          ) : (
            <ul className="mt-4 flex flex-col">
              {groups.map((g, i) => (
                <li
                  key={g.id}
                  className={`flex items-center gap-1 ${i > 0 ? "border-t border-slate-100" : ""}`}
                >
                  <Link
                    href={`/groups/${g.id}`}
                    className="flex min-w-0 flex-1 items-center justify-between gap-3 rounded-lg px-1 py-3 transition hover:bg-slate-50"
                  >
                    <div className="flex min-w-0 items-center gap-3">
                      <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-green-50 text-base font-bold text-green-700">
                        {g.name.charAt(0).toUpperCase()}
                      </span>
                      <div className="min-w-0">
                        <p className="truncate font-medium text-slate-900">{g.name}</p>
                        <p className="truncate text-sm text-slate-500">
                          {g.memberCount} members · created {formatDate(g.createdAt)}
                        </p>
                      </div>
                    </div>
                    <span className="shrink-0 text-sm font-medium text-slate-500">
                      {g.currency}
                    </span>
                  </Link>
                  {g.role === "admin" && (
                    <span className="shrink-0">
                      <DeleteGroupButton groupId={g.id} groupName={g.name} />
                    </span>
                  )}
                </li>
              ))}
            </ul>
          )}
        </section>

        <div className="flex flex-col gap-6">
          <section
            id="create-group"
            aria-labelledby="create-group-heading"
            className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"
          >
            <h2 id="create-group-heading" className="text-xl font-semibold text-slate-900">
              Create a group
            </h2>
            <div className="mt-4">
              <CreateGroupForm />
            </div>
          </section>

          <section
            aria-labelledby="join-group-heading"
            className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm"
          >
            <h2 id="join-group-heading" className="text-xl font-semibold text-slate-900">
              Join a group
            </h2>
            <p className="mt-1 text-sm text-slate-500">
              Ask a group admin for an invitation token.
            </p>
            <div className="mt-4">
              <JoinGroupForm />
            </div>
          </section>
        </div>
      </div>

      <Toast key={success} success={success} />
    </div>
  );
}
