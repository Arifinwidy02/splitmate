"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { LayoutDashboard, Users } from "lucide-react";

import type { Dict } from "@/lib/i18n/id";

function isActive(pathname: string, href: string): boolean {
  if (href === "/") return pathname === "/";
  return pathname === href || pathname.startsWith(`${href}/`);
}

export default function Sidebar({ dict }: { dict: Dict }) {
  const pathname = usePathname();

  const navItems = [
    { href: "/", label: dict.nav.dashboard, icon: LayoutDashboard },
    { href: "/groups", label: dict.nav.groups, icon: Users },
  ];

  return (
    <>
      <aside className="fixed inset-y-0 left-0 hidden w-60 flex-col border-r border-slate-200 bg-white px-4 py-6 lg:flex">
        <Link href="/" className="flex items-center gap-2 px-2">
          <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-green-600 text-sm font-bold text-white">
            S
          </span>
          <span className="text-lg font-bold text-slate-900">SplitMate</span>
        </Link>

        <nav className="mt-8 flex flex-col gap-1" aria-label={dict.nav.mainNavigation}>
          {navItems.map(({ href, label, icon: Icon }) => {
            const active = isActive(pathname, href);
            return (
              <Link
                key={href}
                href={href}
                aria-current={active ? "page" : undefined}
                className={`flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition ${
                  active
                    ? "bg-green-50 text-green-700"
                    : "text-slate-600 hover:bg-slate-50 hover:text-slate-900"
                }`}
              >
                <Icon className="h-4.5 w-4.5" aria-hidden="true" />
                {label}
              </Link>
            );
          })}
        </nav>
      </aside>

      <nav
        className="fixed inset-x-0 bottom-0 z-20 flex border-t border-slate-200 bg-white lg:hidden"
        aria-label={dict.nav.mainNavigation}
      >
        {navItems.map(({ href, label, icon: Icon }) => {
          const active = isActive(pathname, href);
          return (
            <Link
              key={href}
              href={href}
              aria-current={active ? "page" : undefined}
              className={`flex flex-1 flex-col items-center gap-1 py-2.5 text-xs font-medium ${
                active ? "text-green-700" : "text-slate-500"
              }`}
            >
              <Icon className="h-5 w-5" aria-hidden="true" />
              {label}
            </Link>
          );
        })}
      </nav>
    </>
  );
}
