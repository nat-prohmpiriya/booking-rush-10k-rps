export interface Seat {
  id: string
  row: string
  number: number
  zone: string
  status: "available" | "sold" | "reserved"
}

export interface SeatZone {
  id: string
  name: string
  price: number
  colorClass: string
  borderClass: string
}

export const ZONES: SeatZone[] = [
  {
    id: "vip",
    name: "VIP Front",
    price: 450,
    colorClass: "bg-amber-500",
    borderClass: "border-amber-500",
  },
  {
    id: "premium",
    name: "Premium",
    price: 295,
    colorClass: "bg-rose-500",
    borderClass: "border-rose-500",
  },
  {
    id: "standard",
    name: "Standard",
    price: 175,
    colorClass: "bg-sky-500",
    borderClass: "border-sky-500",
  },
  {
    id: "economy",
    name: "Economy",
    price: 85,
    colorClass: "bg-emerald-500",
    borderClass: "border-emerald-500",
  },
]

export function generateSeats(): Seat[] {
  const seats: Seat[] = []
  const rows = ["A", "B", "C", "D", "E", "F", "G", "H", "J", "K", "L", "M"]

  const getZone = (rowIndex: number): string => {
    if (rowIndex < 2) return "vip"
    if (rowIndex < 4) return "premium"
    if (rowIndex < 8) return "standard"
    return "economy"
  }

  const getSeatsPerRow = (rowIndex: number): number => {
    // Curved seating - fewer seats near stage, more in back
    if (rowIndex < 2) return 12
    if (rowIndex < 4) return 16
    if (rowIndex < 8) return 20
    return 24
  }

  rows.forEach((row, rowIndex) => {
    const seatsInRow = getSeatsPerRow(rowIndex)
    for (let i = 1; i <= seatsInRow; i++) {
      const isSold = Math.random() < 0.35 // 35% sold
      seats.push({
        id: `${row}-${i}`,
        row,
        number: i,
        zone: getZone(rowIndex),
        status: isSold ? "sold" : "available",
      })
    }
  })

  return seats
}
