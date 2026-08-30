import AuthHeader from "@/components/auth-header";
import LoginForm from "@/components/login-form";
import Toast from "@/components/toast";
import { getDict } from "@/lib/i18n";

export default async function LoginPage({
  searchParams,
}: {
  searchParams: Promise<{ google?: string; success?: string; next?: string }>;
}) {
  const params = await searchParams;
  const dict = await getDict();

  return (
    <>
      <AuthHeader locale={dict.locale} />
      <main className="flex flex-1 items-center justify-center bg-slate-50 p-8">
        <div className="w-full max-w-sm rounded-2xl border border-slate-200 bg-white p-8 shadow-card">
          <h1 className="text-2xl font-bold tracking-tight text-slate-900">{dict.auth.welcomeBack}</h1>
          <p className="mt-1 text-sm text-slate-500">{dict.auth.signInSubtitle}</p>
          <div className="mt-6 flex flex-col">
            {params.google === "error" && (
              <p
                role="alert"
                className="mb-4 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700"
              >
                {dict.auth.googleFailed}
              </p>
            )}
            <LoginForm dict={dict} next={params.next} />
          </div>
        </div>
      </main>
      <Toast success={params.success} dict={dict} />
    </>
  );
}