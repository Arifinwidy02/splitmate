import RegisterForm from "@/components/register-form";

export default function RegisterPage() {
  return (
    <main className="flex flex-1 items-center justify-center bg-slate-50 p-8">
      <div className="w-full max-w-sm rounded-2xl border border-slate-200 bg-white p-8 shadow-sm">
        <h1 className="text-2xl font-bold text-slate-900">Create your account</h1>
        <p className="mt-1 text-sm text-slate-500">
          Start tracking shared expenses with SplitMate
        </p>
        <RegisterForm />
      </div>
    </main>
  );
}
