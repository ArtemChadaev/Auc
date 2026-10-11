import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router';
import { ArrowLeft, FileText } from 'lucide-react';
import { getDocument } from '~/api/docs';
import type { Document } from '~/api/types';
import { ErrorAlert } from '~/components/feedback/ErrorAlert';

export default function DocPage() {
  const { slug } = useParams();
  const [doc, setDoc] = useState<Document | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown | null>(null);

  useEffect(() => {
    if (!slug) return;
    setLoading(true);
    setError(null);
    getDocument(slug)
      .then(setDoc)
      .catch(setError)
      .finally(() => setLoading(false));
  }, [slug]);

  return (
    <div className="max-w-3xl mx-auto py-6 sm:py-8 space-y-6">
      <Link
        to="/lots"
        className="inline-flex items-center gap-1.5 text-xs font-semibold text-slate-600 hover:text-teal-700 transition-colors"
      >
        <ArrowLeft className="w-4 h-4" />
        <span>Вернуться в каталог</span>
      </Link>

      {error ? <ErrorAlert error={error} /> : null}

      {loading ? (
        <div className="py-12 flex flex-col items-center justify-center space-y-3">
          <div className="w-8 h-8 border-3 border-teal-600 border-t-transparent rounded-full animate-spin" />
          <p className="text-xs text-slate-500 font-medium">Загрузка документа...</p>
        </div>
      ) : doc ? (
        <div className="bg-white rounded-2xl border border-slate-200/80 p-6 sm:p-8 shadow-xs space-y-5">
          <div className="flex items-center gap-3 pb-4 border-b border-slate-100">
            <div className="w-10 h-10 rounded-xl bg-teal-50 text-teal-700 flex items-center justify-center">
              <FileText className="w-5 h-5" />
            </div>
            <div>
              <h1 className="text-lg sm:text-xl font-bold text-slate-900">
                {doc.name === 'terms'
                  ? 'Пользовательское соглашение'
                  : doc.name === 'privacy'
                  ? 'Политика конфиденциальности'
                  : 'Правила платформы и Anti-Sniping'}
              </h1>
              <p className="text-xs text-slate-400 mt-0.5">Версия {doc.version}</p>
            </div>
          </div>

          <div className="text-sm text-slate-700 leading-relaxed whitespace-pre-line space-y-4">
            {doc.text}
          </div>
        </div>
      ) : null}
    </div>
  );
}
