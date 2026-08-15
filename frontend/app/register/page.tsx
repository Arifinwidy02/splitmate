import LanguageSwitcher from "@/components/language-switcher";
import RegisterForm from "@/components/register-form";
import { getDict } from "@/lib/i18n";

export default async function RegisterPage() {
  const dict = await getDict();

  return (
    <main className="flex flex-1 items-center justify-center bg-slate-50 p-8">
      <div className="relative w-full max-w-sm rounded-2xl border border-slate-200 bg-white p-8 shadow-sm">
        <div className="absolute right-6 top-6">
          <LanguageSwitcher current={dict.locale} />
        </div>
        <h1 className="text-2xl font-bold text-slate-900">{dict.auth.createAccount}</h1>
        <p className="mt-1 text-sm text-slate-500">{dict.auth.registerSubtitle}</p>
        <RegisterForm dict={dict} />
      </div>
    </main>
  );
}