import React from "react";

export default function StartPage() {

  return (
    <div>
      <Cards/>
    </div>
  )
}

const Cards = () => {
  const cards = []
  for (let i = 1; i <= 10; i++)
    cards.push(<div className="w-30 m-3 h-40 border rounded-md">Карточка</div>)
  return cards
}