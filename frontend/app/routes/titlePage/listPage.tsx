// А это просто список всех карточек с фильтрами

import React from "react";

export default function ListPage() {
  return (
    <div className="flex w-full">
      <div className="w-56 rounded-lg border" />
      <div className="w-230 rounded-lg border" />
    </div>
  )
}

// const Cards = () => {
//   const cards = []
//   for (let i = 1; i <= 10; i++)
//     cards.push(<div className="w-30 m-3 h-40 border rounded-md">Карточка</div>)
//   return cards
// }