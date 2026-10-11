import { Link, useLocation } from 'react-router';
import { LayoutGrid, PlusCircle, User as UserIcon } from 'lucide-react';
import { useAuth } from '~/hooks/useAuth';

export function BottomNav() {
  const location = useLocation();
  const { isAuthenticated } = useAuth();

  const isLots = location.pathname.startsWith('/lots') && location.pathname !== '/lots/new';
  const isCreate = location.pathname === '/lots/new';
  const isProfile = location.pathname.startsWith('/profile') || location.pathname.startsWith('/login') || location.pathname.startsWith('/register');

  return (
    <nav className="fixed bottom-0 left-0 right-0 z-40 bg-white/95 backdrop-blur-md border-t border-slate-200/80 md:hidden py-1.5 px-2 grid grid-cols-3 shadow-lg">
      <Link
        to="/lots"
        className={`flex flex-col items-center justify-center py-1 rounded-xl transition-colors ${
          isLots ? 'text-teal-600 font-semibold' : 'text-slate-500 hover:text-slate-800'
        }`}
      >
        <LayoutGrid className="w-5 h-5" />
        <span className="text-[11px] mt-0.5">Каталог</span>
      </Link>

      <Link
        to="/lots/new"
        className={`flex flex-col items-center justify-center py-1 rounded-xl transition-colors ${
          isCreate ? 'text-teal-600 font-semibold' : 'text-slate-500 hover:text-slate-800'
        }`}
      >
        <PlusCircle className="w-5 h-5" />
        <span className="text-[11px] mt-0.5">Создать</span>
      </Link>

      <Link
        to={isAuthenticated ? '/profile' : '/login'}
        className={`flex flex-col items-center justify-center py-1 rounded-xl transition-colors ${
          isProfile ? 'text-teal-600 font-semibold' : 'text-slate-500 hover:text-slate-800'
        }`}
      >
        <UserIcon className="w-5 h-5" />
        <span className="text-[11px] mt-0.5">Профиль</span>
      </Link>
    </nav>
  );
}
