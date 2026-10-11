import React from 'react';

export function LotSkeleton() {
  return (
    <div className="bg-white rounded-2xl border border-slate-200/80 p-3.5 shadow-xs flex flex-col gap-3 animate-pulse">
      <div className="w-full aspect-4/3 rounded-xl bg-slate-200" />
      <div className="space-y-2 mt-1">
        <div className="h-4 bg-slate-200 rounded-md w-3/4" />
        <div className="h-3 bg-slate-100 rounded-md w-1/2" />
      </div>
      <div className="mt-auto pt-3 border-t border-slate-100 flex items-center justify-between">
        <div className="h-5 bg-slate-200 rounded-md w-24" />
        <div className="h-5 bg-slate-100 rounded-md w-16" />
      </div>
    </div>
  );
}

export function LotGridSkeleton({ count = 6 }: { count?: number }) {
  return (
    <div className="grid grid-cols-2 sm:grid-cols-2 lg:grid-cols-3 gap-2.5 sm:gap-4">
      {Array.from({ length: count }).map((_, i) => (
        <LotSkeleton key={i} />
      ))}
    </div>
  );
}
