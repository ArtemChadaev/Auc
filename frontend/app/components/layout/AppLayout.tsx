import { Outlet } from 'react-router';
import { Header } from './Header';
import { Footer } from './Footer';
import { BottomNav } from './BottomNav';
import { AuthProvider } from '~/hooks/useAuth';

export default function AppLayout() {
  return (
    <AuthProvider>
      <div className="min-h-screen flex flex-col bg-slate-50 text-slate-900">
        <Header />
        <main className="flex-1 w-full max-w-5xl mx-auto px-4 py-4 pb-24 md:py-6 md:pb-6">
          <Outlet />
        </main>
        <Footer />
        <BottomNav />
      </div>
    </AuthProvider>
  );
}
