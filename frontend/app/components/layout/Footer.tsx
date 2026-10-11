import { useState } from 'react';
import { FileText, X } from 'lucide-react';
import { SEED_DOCUMENTS } from '~/api/mocks/seed';

export function Footer() {
  const [activeDocName, setActiveDocName] = useState<string | null>(null);

  const activeDoc = SEED_DOCUMENTS.find((d) => d.name === activeDocName);

  return (
    <>
      <footer className="w-full border-t border-slate-200/80 bg-white py-5 mt-auto mb-16 md:mb-0 text-slate-500 text-xs">
        <div className="max-w-5xl mx-auto px-4 flex flex-col sm:flex-row items-center justify-between gap-3.5">
          <div className="flex items-center gap-2 text-slate-600 font-medium">
            <span>Аукцион цифровых файлов и виртуальных предметов</span>
          </div>

          <div className="flex items-center gap-4 text-slate-500 font-medium">
            <button
              type="button"
              onClick={() => setActiveDocName('terms')}
              className="hover:text-teal-700 transition-colors cursor-pointer"
            >
              Условия
            </button>
            <button
              type="button"
              onClick={() => setActiveDocName('privacy')}
              className="hover:text-teal-700 transition-colors cursor-pointer"
            >
              Конфиденциальность
            </button>
            <button
              type="button"
              onClick={() => setActiveDocName('rules')}
              className="hover:text-teal-700 transition-colors cursor-pointer"
            >
              Правила и Anti-Sniping
            </button>
          </div>

          <div className="text-slate-400 text-[11px]">
            Виртуальная валюта Coins 🪙
          </div>
        </div>
      </footer>

      {/* Модальное окно для чтения документов (Popup Dialog) */}
      {activeDocName && activeDoc && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/40 backdrop-blur-xs">
          <div className="bg-white rounded-2xl border border-slate-200 p-5 sm:p-6 w-full max-w-xl max-h-[85vh] flex flex-col shadow-2xl space-y-4 animate-in fade-in zoom-in-95 duration-150">
            {/* Хедер модалки */}
            <div className="flex items-center justify-between pb-3 border-b border-slate-100">
              <div className="flex items-center gap-2.5">
                <div className="w-8 h-8 rounded-lg bg-teal-50 text-teal-700 flex items-center justify-center">
                  <FileText className="w-4.5 h-4.5" />
                </div>
                <div>
                  <h3 className="font-bold text-sm sm:text-base text-slate-900">
                    {activeDoc.name === 'terms'
                      ? 'Пользовательское соглашение'
                      : activeDoc.name === 'privacy'
                      ? 'Политика конфиденциальности'
                      : 'Правила платформы и Anti-Sniping'}
                  </h3>
                  <span className="text-[10px] text-slate-400">Версия {activeDoc.version}</span>
                </div>
              </div>

              <button
                type="button"
                onClick={() => setActiveDocName(null)}
                className="p-1.5 rounded-lg text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition-colors cursor-pointer"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {/* Вкладки быстрого переключения документов в поп-апе */}
            <div className="flex items-center gap-1.5 p-1 bg-slate-100 rounded-xl text-xs font-semibold">
              <button
                type="button"
                onClick={() => setActiveDocName('terms')}
                className={`flex-1 py-1 px-2 rounded-lg transition-all ${
                  activeDocName === 'terms' ? 'bg-white text-slate-900 shadow-xs' : 'text-slate-500 hover:text-slate-800'
                }`}
              >
                Условия
              </button>
              <button
                type="button"
                onClick={() => setActiveDocName('privacy')}
                className={`flex-1 py-1 px-2 rounded-lg transition-all ${
                  activeDocName === 'privacy' ? 'bg-white text-slate-900 shadow-xs' : 'text-slate-500 hover:text-slate-800'
                }`}
              >
                Конфиденциальность
              </button>
              <button
                type="button"
                onClick={() => setActiveDocName('rules')}
                className={`flex-1 py-1 px-2 rounded-lg transition-all ${
                  activeDocName === 'rules' ? 'bg-white text-slate-900 shadow-xs' : 'text-slate-500 hover:text-slate-800'
                }`}
              >
                Anti-Sniping
              </button>
            </div>

            {/* Содержимое документа */}
            <div className="overflow-y-auto flex-1 pr-1 text-xs sm:text-sm text-slate-700 leading-relaxed whitespace-pre-line space-y-3">
              {activeDoc.text}
            </div>

            {/* Футер модалки */}
            <div className="pt-3 border-t border-slate-100 flex justify-end">
              <button
                type="button"
                onClick={() => setActiveDocName(null)}
                className="px-4 py-2 bg-teal-600 hover:bg-teal-700 text-white font-semibold text-xs rounded-xl shadow-xs transition-colors cursor-pointer"
              >
                Закрыть
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
