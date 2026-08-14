import LogoutButton from "@/components/logout-button";
import Sidebar from "@/components/sidebar";
import { getCurrentUser } from "@/lib/auth";

export default async function AppLayout({ children }: LayoutProps<"/">) {
  const user = await getCurrentUser();

  return (
    <div className="min-h-full bg-slate-50">
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-50 focus:rounded-lg focus:bg-green-600 focus:px-4 focus:py-2 focus:text-sm focus:font-semibold focus:text-white"
      >
        Skip to content
      </a>

      <Sidebar />

      <div className="flex min-h-full flex-col lg:pl-60">
        <header className="flex h-16 items-center justify-between border-b border-slate-200 bg-white px-4 sm:px-6 lg:px-8">
          <div className="flex items-center gap-2 lg:hidden">
            <span className="flex h-7 w-7 items-center justify-center rounded-lg bg-green-600 text-sm font-bold text-white">
              S
            </span>
            <span className="font-bold text-slate-900">SplitMate</span>
          </div>

          <div className="hidden items-center gap-2 lg:flex">
            <span className="flex h-8 w-8 items-center justify-center rounded-full bg-slate-100 text-sm font-semibold text-slate-700">
              {user.name.charAt(0).toUpperCase()}
            </span>
            <span className="text-sm font-medium text-slate-700">{user.name}</span>
          </div>

          <LogoutButton />
        </header>

        <main id="main" className="flex-1 px-4 pb-24 pt-6 sm:px-6 lg:px-8 lg:pb-8">{children}</main>
      </div>
    </div>
  );
}
