export function formatCurrency(amount: string, currency = "IDR"): string {
  const value = Number(amount);
  if (Number.isNaN(value)) return amount;

  const abs = Math.abs(value);
  const formatted = new Intl.NumberFormat("id-ID", {
    minimumFractionDigits: Number.isInteger(abs) ? 0 : 2,
    maximumFractionDigits: 2,
  }).format(abs);

  const prefix = currency === "IDR" ? "Rp" : `${currency} `;
  return `${value < 0 ? "-" : ""}${prefix}${formatted}`;
}

export function formatSignedCurrency(amount: string, currency = "IDR"): string {
  const value = Number(amount);
  const formatted = formatCurrency(amount, currency);
  if (value === 0) return formatted;
  return value > 0 ? `+${formatted}` : `-${formatted.slice(1)}`;
}

export function formatDate(iso: string): string {
  return new Intl.DateTimeFormat("id-ID", {
    day: "numeric",
    month: "short",
    year: "numeric",
  }).format(new Date(iso));
}

export function formatDateTime(iso: string): string {
  return new Intl.DateTimeFormat("id-ID", {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(new Date(iso));
}

export function toRFC3339(datetimeLocal: string): string {
  if (!/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(datetimeLocal)) {
    return datetimeLocal;
  }

  const d = new Date(datetimeLocal);
  if (Number.isNaN(d.getTime())) return datetimeLocal;

  const offsetMinutes = -d.getTimezoneOffset();
  const sign = offsetMinutes >= 0 ? "+" : "-";
  const abs = Math.abs(offsetMinutes);
  const offset = `${sign}${String(Math.floor(abs / 60)).padStart(2, "0")}:${String(
    abs % 60,
  ).padStart(2, "0")}`;

  return `${datetimeLocal}:00${offset}`;
}
