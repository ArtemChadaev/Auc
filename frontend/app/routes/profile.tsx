import { useState, useEffect, useCallback } from 'react';
import { Link, useNavigate } from 'react-router';
import { useAuth } from '~/hooks/useAuth';
import { listUserItems } from '~/api/items';
import { listLots, getMyLots } from '~/api/lots';
import { listTransactions } from '~/api/profile';
import type { Item, Lot, LedgerEntry } from '~/api/types';
import { useUserStream } from '~/realtime/useUserStream';
import { BalanceCard } from '~/components/profile/BalanceCard';
import { MyItems } from '~/components/profile/MyItems';
import { MyLots } from '~/components/profile/MyLots';
import { MyBids } from '~/components/profile/MyBids';
import { TransactionsTable } from '~/components/profile/TransactionsTable';
import { ErrorAlert } from '~/components/feedback/ErrorAlert';
import { getParticipatingLotIds } from '~/lib/participating-lots';

export default function ProfilePage() {
  const { user, isAuthenticated, loading: authLoading, claimDailyCoins, logout } = useAuth();
  const navigate = useNavigate();

  const [items, setItems] = useState<Item[]>([]);
  const [myLots, setMyLots] = useState<Lot[]>([]);
  const [participatingLots, setParticipatingLots] = useState<Lot[]>([]);
  const [transactions, setTransactions] = useState<LedgerEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown | null>(null);

  const loadProfileData = useCallback(async () => {
    if (!user) return;
    setLoading(true);
    setError(null);
    try {
      const [itemsData, lotsData, txData, myLotsData] = await Promise.all([
        listUserItems(),
        listLots({ limit: 50 }),
        listTransactions(),
        getMyLots().catch(() => []),
      ]);

      setItems(itemsData);
      setTransactions(txData);

      // Лоты продавца (все статусы, включая draft, pending_review, approved)
      setMyLots(myLotsData);

      // Лоты, где текущий пользователь участвовал со ставкой (из локальной записи)
      const participatedIds = getParticipatingLotIds(user.id);
      const bidded = lotsData.items.filter(
        (l) => participatedIds.includes(l.id) || l.winner?.id === user.id
      );
      setParticipatingLots(bidded);
    } catch (err) {
      setError(err);
    } finally {
      setLoading(false);
    }
  }, [user]);

  useEffect(() => {
    if (!authLoading && !isAuthenticated) {
      navigate('/login');
      return;
    }
    if (isAuthenticated) {
      loadProfileData();
    }
  }, [authLoading, isAuthenticated, loadProfileData, navigate]);

  // Реалтайм-слушатель персональных событий (баланс, новые транзакции)
  useUserStream(() => {
    // При изменении баланса перезапрашиваем свежие транзакции
    listTransactions().then(setTransactions).catch(() => {});
  });

  if (authLoading || (!isAuthenticated && loading)) {
    return (
      <div className="py-12 flex flex-col items-center justify-center space-y-3">
        <div className="w-8 h-8 border-3 border-teal-600 border-t-transparent rounded-full animate-spin" />
        <p className="text-xs text-slate-500 font-medium">Загрузка профиля...</p>
      </div>
    );
  }

  if (!user) return null;

  return (
    <div className="space-y-6 pb-12">
      <div className="flex flex-col gap-1">
        <h1 className="text-xl sm:text-2xl font-bold tracking-tight text-slate-900">
          Личный кабинет
        </h1>
      </div>

      {error ? <ErrorAlert error={error} onDismiss={() => setError(null)} /> : null}

      {/* 1. Карточка профиля и баланса */}
      <BalanceCard
        user={user}
        onClaimDaily={claimDailyCoins}
        onLogout={async () => {
          await logout();
          navigate('/login');
        }}
      />

      {/* 2. Мои предметы */}
      <div id="items" className="scroll-mt-20">
        <MyItems
          items={items}
          onItemAdded={(newItem) => setItems((prev) => [newItem, ...prev])}
          onItemRemoved={(removedId) => setItems((prev) => prev.filter((it) => it.id !== removedId))}
        />
      </div>

      {/* 3. Мои лоты */}
      <MyLots lots={myLots} onRefresh={loadProfileData} />

      {/* 4. Мои ставки */}
      <MyBids lots={participatingLots} />

      {/* 5. История операций */}
      <TransactionsTable transactions={transactions} />
    </div>
  );
}
