import React from 'react';
import type { Lot } from '~/api/types';
import { LotCard } from './LotCard';

interface LotGridProps {
  lots: Lot[];
}

export function LotGrid({ lots }: LotGridProps) {
  return (
    <div className="grid grid-cols-2 sm:grid-cols-2 lg:grid-cols-3 gap-2.5 sm:gap-4">
      {lots.map((lot) => (
        <LotCard key={lot.id} lot={lot} />
      ))}
    </div>
  );
}
