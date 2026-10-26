import { Link } from 'react-router';

export function Footer() {
  return (
    <footer className="w-full border-t border-slate-200/80 bg-white py-5 mt-auto mb-16 md:mb-0 text-slate-500 text-xs">
      <div className="max-w-5xl mx-auto px-4 flex flex-col sm:flex-row items-center justify-between gap-3.5">
        <div className="flex items-center gap-2 text-slate-600 font-medium">
          <span>Аукцион цифровых файлов и виртуальных предметов</span>
        </div>

        <div className="flex items-center gap-4 text-slate-500 font-medium">
          <Link
            to="/docs/terms-of-service"
            className="hover:text-teal-700 transition-colors"
          >
            Условия
          </Link>
          <Link
            to="/docs/privacy-policy"
            className="hover:text-teal-700 transition-colors"
          >
            Конфиденциальность
          </Link>
          <Link
            to="/docs/auction-rules"
            className="hover:text-teal-700 transition-colors"
          >
            Правила и Anti-Sniping
          </Link>
          <Link
            to="/docs/cookie-policy"
            className="hover:text-teal-700 transition-colors"
          >
            Cookie
          </Link>
        </div>

        <div className="text-slate-400 text-[11px]">
          Виртуальная валюта Coins 🪙
        </div>
      </div>
    </footer>
  );
}
