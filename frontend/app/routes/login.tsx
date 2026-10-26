import { useState } from 'react';
import { Link, useNavigate } from 'react-router';
import { LogIn, Loader2 } from 'lucide-react';
import { useAuth } from '~/hooks/useAuth';
import { ErrorAlert } from '~/components/feedback/ErrorAlert';

export default function LoginPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<unknown | null>(null);

  const { login } = useAuth();
  const navigate = useNavigate();

  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const errors: Record<string, string> = {};
    if (!email.trim()) {
      errors.email = 'Пожалуйста, укажите email адрес';
    }
    if (!password) {
      errors.password = 'Пожалуйста, укажите пароль';
    }
    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      return;
    }

    setFieldErrors({});
    setSubmitting(true);
    setError(null);
    try {
      await login(email.trim(), password);
      navigate('/lots');
    } catch (err) {
      setError(err);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="max-w-md mx-auto py-8 sm:py-12 space-y-6">
      <div className="text-center space-y-1">
        <h1 className="text-2xl font-bold tracking-tight text-slate-900">Вход в аккаунт</h1>
        <p className="text-xs sm:text-sm text-slate-500">
          Войдите, чтобы делать ставки и управлять лотами
        </p>
      </div>

      <div className="bg-white rounded-2xl border border-slate-200/80 p-6 shadow-xs space-y-5">
        {error ? <ErrorAlert error={error} onDismiss={() => setError(null)} /> : null}

        <form onSubmit={handleSubmit} noValidate className="space-y-4">
          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">
              Электронная почта
            </label>
            <input
              type="email"
              value={email}
              onChange={(e) => {
                setEmail(e.target.value);
                if (fieldErrors.email) setFieldErrors((prev) => ({ ...prev, email: '' }));
              }}
              placeholder="name@example.com"
              className={`w-full px-3.5 py-2.5 text-sm bg-slate-50 border rounded-xl focus:outline-hidden focus:ring-2 ${
                fieldErrors.email
                  ? 'border-rose-400 bg-rose-50/20 focus:ring-rose-400/20'
                  : 'border-slate-200 focus:ring-teal-500/20 focus:border-teal-500'
              }`}
            />
            {fieldErrors.email ? (
              <p className="mt-1 text-xs text-rose-600 font-medium">{fieldErrors.email}</p>
            ) : null}
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">Пароль</label>
            <input
              type="password"
              value={password}
              onChange={(e) => {
                setPassword(e.target.value);
                if (fieldErrors.password) setFieldErrors((prev) => ({ ...prev, password: '' }));
              }}
              placeholder="••••••••"
              className={`w-full px-3.5 py-2.5 text-sm bg-slate-50 border rounded-xl focus:outline-hidden focus:ring-2 ${
                fieldErrors.password
                  ? 'border-rose-400 bg-rose-50/20 focus:ring-rose-400/20'
                  : 'border-slate-200 focus:ring-teal-500/20 focus:border-teal-500'
              }`}
            />
            {fieldErrors.password ? (
              <p className="mt-1 text-xs text-rose-600 font-medium">{fieldErrors.password}</p>
            ) : null}
          </div>

          <button
            type="submit"
            disabled={submitting}
            className="w-full py-2.5 bg-teal-600 hover:bg-teal-700 text-white font-bold text-xs sm:text-sm rounded-xl shadow-xs transition-colors flex items-center justify-center gap-2 disabled:opacity-50"
          >
            {submitting ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                <span>Авторизация...</span>
              </>
            ) : (
              <>
                <LogIn className="w-4 h-4" />
                <span>Войти</span>
              </>
            )}
          </button>
        </form>
      </div>

      <p className="text-center text-xs text-slate-500">
        Нет аккаунта?{' '}
        <Link to="/register" className="text-teal-600 font-semibold hover:underline">
          Зарегистрироваться
        </Link>
      </p>
    </div>
  );
}
