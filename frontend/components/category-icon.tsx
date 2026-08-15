import {
  BedDouble,
  CarFront,
  Clapperboard,
  Lightbulb,
  ShoppingBag,
  Tag,
  UtensilsCrossed,
  type LucideIcon,
} from "lucide-react";

export const CATEGORY_ICONS: Record<string, LucideIcon> = {
  Accommodation: BedDouble,
  "Food & Drinks": UtensilsCrossed,
  Transportation: CarFront,
  Shopping: ShoppingBag,
  Entertainment: Clapperboard,
  Utilities: Lightbulb,
  Other: Tag,
};

export function CategoryIcon({
  category,
  className,
}: {
  category: string;
  className?: string;
}) {
  const Icon = CATEGORY_ICONS[category] ?? Tag;
  return <Icon className={className} aria-hidden="true" />;
}